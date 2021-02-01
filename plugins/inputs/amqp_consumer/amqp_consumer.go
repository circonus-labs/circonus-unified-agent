package amqpconsumer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/parsers"
	"github.com/streadway/amqp"
)

const (
	defaultMaxUndeliveredMessages = 1000
)

type empty struct{}
type semaphore chan empty

// AMQPConsumer is the top level struct for this plugin
type AMQPConsumer struct {
	URL                    string            `toml:"url"` // deprecated in 1.7; use brokers
	Brokers                []string          `toml:"brokers"`
	Username               string            `toml:"username"`
	Password               string            `toml:"password"`
	Exchange               string            `toml:"exchange"`
	ExchangeType           string            `toml:"exchange_type"`
	ExchangeDurability     string            `toml:"exchange_durability"`
	ExchangePassive        bool              `toml:"exchange_passive"`
	ExchangeArguments      map[string]string `toml:"exchange_arguments"`
	MaxUndeliveredMessages int               `toml:"max_undelivered_messages"`

	// Queue Name
	Queue           string `toml:"queue"`
	QueueDurability string `toml:"queue_durability"`
	QueuePassive    bool   `toml:"queue_passive"`

	// Binding Key
	BindingKey string `toml:"binding_key"`

	// Controls how many messages the server will try to keep on the network
	// for consumers before receiving delivery acks.
	PrefetchCount int

	// AMQP Auth method
	AuthMethod string
	tls.ClientConfig

	ContentEncoding string `toml:"content_encoding"`
	Log             cua.Logger

	deliveries map[cua.TrackingID]amqp.Delivery

	parser  parsers.Parser
	conn    *amqp.Connection
	wg      *sync.WaitGroup
	cancel  context.CancelFunc
	decoder internal.ContentDecoder
}

type externalAuth struct{}

func (a *externalAuth) Mechanism() string {
	return "EXTERNAL"
}
func (a *externalAuth) Response() string {
	return "\000"
}

const (
	DefaultAuthMethod = "PLAIN"

	DefaultBroker = "amqp://localhost:5672/influxdb"

	DefaultExchangeType       = "topic"
	DefaultExchangeDurability = "durable"

	DefaultQueueDurability = "durable"

	DefaultPrefetchCount = 50
)

func (a *AMQPConsumer) SampleConfig() string {
	return `
  ## Broker to consume from.
  ##   deprecated in 1.7; use the brokers option
  # url = "amqp://localhost:5672/influxdb"

  ## Brokers to consume from.  If multiple brokers are specified a random broker
  ## will be selected anytime a connection is established.  This can be
  ## helpful for load balancing when not using a dedicated load balancer.
  brokers = ["amqp://localhost:5672/influxdb"]

  ## Authentication credentials for the PLAIN auth_method.
  # username = ""
  # password = ""

  ## Name of the exchange to declare.  If unset, no exchange will be declared.
  exchange = "circonus"

  ## Exchange type; common types are "direct", "fanout", "topic", "header", "x-consistent-hash".
  # exchange_type = "topic"

  ## If true, exchange will be passively declared.
  # exchange_passive = false

  ## Exchange durability can be either "transient" or "durable".
  # exchange_durability = "durable"

  ## Additional exchange arguments.
  # exchange_arguments = { }
  # exchange_arguments = {"hash_property" = "timestamp"}

  ## AMQP queue name.
  queue = "circonus"

  ## AMQP queue durability can be "transient" or "durable".
  queue_durability = "durable"

  ## If true, queue will be passively declared.
  # queue_passive = false

  ## A binding between the exchange and queue using this binding key is
  ## created.  If unset, no binding is created.
  binding_key = "#"

  ## Maximum number of messages server should give to the worker.
  # prefetch_count = 50

  ## Maximum messages to read from the broker that have not been written by an
  ## output.  For best throughput set based on the number of metrics within
  ## each message and the size of the output's metric_batch_size.
  ##
  ## For example, if each message from the queue contains 10 metrics and the
  ## output metric_batch_size is 1000, setting this to 100 will ensure that a
  ## full batch is collected and the write is triggered immediately without
  ## waiting until the next flush_interval.
  # max_undelivered_messages = 1000

  ## Auth method. PLAIN and EXTERNAL are supported
  ## Using EXTERNAL requires enabling the rabbitmq_auth_mechanism_ssl plugin as
  ## described here: https://www.rabbitmq.com/plugins.html
  # auth_method = "PLAIN"

  ## Optional TLS Config
  # tls_ca = "/etc/circonus/ca.pem"
  # tls_cert = "/etc/circonus/cert.pem"
  # tls_key = "/etc/circonus/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false

  ## Content encoding for message payloads, can be set to "gzip" to or
  ## "identity" to apply no encoding.
  # content_encoding = "identity"

  ## Data format to consume.
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/circonus-labs/circonus-unified-agent/blob/master/docs/DATA_FORMATS_INPUT.md
  data_format = "influx"
`
}

func (a *AMQPConsumer) Description() string {
	return "AMQP consumer plugin"
}

func (a *AMQPConsumer) SetParser(parser parsers.Parser) {
	a.parser = parser
}

// All gathering is done in the Start function
func (a *AMQPConsumer) Gather(_ cua.Accumulator) error {
	return nil
}

func (a *AMQPConsumer) createConfig() (*amqp.Config, error) {
	// make new tls config
	tls, err := a.ClientConfig.TLSConfig()
	if err != nil {
		return nil, err
	}

	var auth []amqp.Authentication
	if strings.ToUpper(a.AuthMethod) == "EXTERNAL" {
		auth = []amqp.Authentication{&externalAuth{}}
	} else if a.Username != "" || a.Password != "" {
		auth = []amqp.Authentication{
			&amqp.PlainAuth{
				Username: a.Username,
				Password: a.Password,
			},
		}
	}

	config := amqp.Config{
		TLSClientConfig: tls,
		SASL:            auth, // if nil, it will be PLAIN
	}
	return &config, nil
}

// Start satisfies the cua.ServiceInput interface
func (a *AMQPConsumer) Start(acc cua.Accumulator) error {
	amqpConf, err := a.createConfig()
	if err != nil {
		return err
	}

	a.decoder, err = internal.NewContentDecoder(a.ContentEncoding)
	if err != nil {
		return err
	}

	msgs, err := a.connect(amqpConf)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	a.wg = &sync.WaitGroup{}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.process(ctx, msgs, acc)
	}()

	go func() {
		for {
			err := <-a.conn.NotifyClose(make(chan *amqp.Error))
			if err == nil {
				break
			}

			a.Log.Infof("Connection closed: %s; trying to reconnect", err)
			for {
				msgs, err := a.connect(amqpConf)
				if err != nil {
					a.Log.Errorf("AMQP connection failed: %s", err)
					time.Sleep(10 * time.Second)
					continue
				}

				a.wg.Add(1)
				go func() {
					defer a.wg.Done()
					a.process(ctx, msgs, acc)
				}()
				break
			}
		}
	}()

	return nil
}

func (a *AMQPConsumer) connect(amqpConf *amqp.Config) (<-chan amqp.Delivery, error) {
	brokers := a.Brokers
	if len(brokers) == 0 {
		brokers = []string{a.URL}
	}

	p := rand.Perm(len(brokers))
	for _, n := range p {
		broker := brokers[n]
		a.Log.Debugf("Connecting to %q", broker)
		conn, err := amqp.DialConfig(broker, *amqpConf)
		if err == nil {
			a.conn = conn
			a.Log.Debugf("Connected to %q", broker)
			break
		}
		a.Log.Debugf("Error connecting to %q", broker)
	}

	if a.conn == nil {
		return nil, errors.New("could not connect to any broker")
	}

	ch, err := a.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Failed to open a channel: %w", err)
	}

	if a.Exchange != "" {
		var exchangeDurable bool
		switch a.ExchangeDurability {
		case "transient":
			exchangeDurable = false
		default:
			exchangeDurable = true
		}

		exchangeArgs := make(amqp.Table, len(a.ExchangeArguments))
		for k, v := range a.ExchangeArguments {
			exchangeArgs[k] = v
		}

		err = declareExchange(
			ch,
			a.Exchange,
			a.ExchangeType,
			a.ExchangePassive,
			exchangeDurable,
			exchangeArgs)
		if err != nil {
			return nil, err
		}
	}

	q, err := declareQueue(
		ch,
		a.Queue,
		a.QueueDurability,
		a.QueuePassive)
	if err != nil {
		return nil, err
	}

	if a.BindingKey != "" {
		err = ch.QueueBind(
			q.Name,       // queue
			a.BindingKey, // binding-key
			a.Exchange,   // exchange
			false,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("Failed to bind a queue: %w", err)
		}
	}

	err = ch.Qos(
		a.PrefetchCount,
		0,     // prefetch-size
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to set QoS: %w", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("Failed establishing connection to queue: %w", err)
	}

	return msgs, err
}

func declareExchange(
	channel *amqp.Channel,
	exchangeName string,
	exchangeType string,
	exchangePassive bool,
	exchangeDurable bool,
	exchangeArguments amqp.Table,
) error {
	var err error
	if exchangePassive {
		err = channel.ExchangeDeclarePassive(
			exchangeName,
			exchangeType,
			exchangeDurable,
			false, // delete when unused
			false, // internal
			false, // no-wait
			exchangeArguments,
		)
	} else {
		err = channel.ExchangeDeclare(
			exchangeName,
			exchangeType,
			exchangeDurable,
			false, // delete when unused
			false, // internal
			false, // no-wait
			exchangeArguments,
		)
	}
	if err != nil {
		return fmt.Errorf("Error declaring exchange: %w", err)
	}
	return nil
}

func declareQueue(
	channel *amqp.Channel,
	queueName string,
	queueDurability string,
	queuePassive bool,
) (*amqp.Queue, error) {
	var queue amqp.Queue
	var err error

	var queueDurable bool
	switch queueDurability {
	case "transient":
		queueDurable = false
	default:
		queueDurable = true
	}

	if queuePassive {
		queue, err = channel.QueueDeclarePassive(
			queueName,    // queue
			queueDurable, // durable
			false,        // delete when unused
			false,        // exclusive
			false,        // no-wait
			nil,          // arguments
		)
	} else {
		queue, err = channel.QueueDeclare(
			queueName,    // queue
			queueDurable, // durable
			false,        // delete when unused
			false,        // exclusive
			false,        // no-wait
			nil,          // arguments
		)
	}
	if err != nil {
		return nil, fmt.Errorf("Error declaring queue: %w", err)
	}
	return &queue, nil
}

// Read messages from queue and add them to the Accumulator
func (a *AMQPConsumer) process(ctx context.Context, msgs <-chan amqp.Delivery, ac cua.Accumulator) {
	a.deliveries = make(map[cua.TrackingID]amqp.Delivery)

	acc := ac.WithTracking(a.MaxUndeliveredMessages)
	sem := make(semaphore, a.MaxUndeliveredMessages)

	for {
		select {
		case <-ctx.Done():
			return
		case track := <-acc.Delivered():
			if a.onDelivery(track) {
				<-sem
			}
		case sem <- empty{}:
			select {
			case <-ctx.Done():
				return
			case track := <-acc.Delivered():
				if a.onDelivery(track) {
					<-sem
					<-sem
				}
			case d, ok := <-msgs:
				if !ok {
					return
				}
				err := a.onMessage(acc, d)
				if err != nil {
					acc.AddError(err)
					<-sem
				}
			}
		}
	}
}

func (a *AMQPConsumer) onMessage(acc cua.TrackingAccumulator, d amqp.Delivery) error {
	onError := func() {
		// Discard the message from the queue; will never be able to process
		// this message.
		rejErr := d.Ack(false)
		if rejErr != nil {
			a.Log.Errorf("Unable to reject message: %d: %v", d.DeliveryTag, rejErr)
			a.conn.Close()
		}
	}

	body, err := a.decoder.Decode(d.Body)
	if err != nil {
		onError()
		return err
	}

	metrics, err := a.parser.Parse(body)
	if err != nil {
		onError()
		return err
	}

	id := acc.AddTrackingMetricGroup(metrics)
	a.deliveries[id] = d
	return nil
}

func (a *AMQPConsumer) onDelivery(track cua.DeliveryInfo) bool {
	delivery, ok := a.deliveries[track.ID()]
	if !ok {
		// Added by a previous connection
		return false
	}

	if track.Delivered() {
		err := delivery.Ack(false)
		if err != nil {
			a.Log.Errorf("Unable to ack written delivery: %d: %v", delivery.DeliveryTag, err)
			a.conn.Close()
		}
	} else {
		err := delivery.Reject(false)
		if err != nil {
			a.Log.Errorf("Unable to reject failed delivery: %d: %v", delivery.DeliveryTag, err)
			a.conn.Close()
		}
	}

	delete(a.deliveries, track.ID())
	return true
}

func (a *AMQPConsumer) Stop() {
	a.cancel()
	a.wg.Wait()
	err := a.conn.Close()
	if err != nil && errors.Is(err, amqp.ErrClosed) {
		a.Log.Errorf("Error closing AMQP connection: %s", err)
		return
	}
}

func init() {
	inputs.Add("amqp_consumer", func() cua.Input {
		return &AMQPConsumer{
			URL:                    DefaultBroker,
			AuthMethod:             DefaultAuthMethod,
			ExchangeType:           DefaultExchangeType,
			ExchangeDurability:     DefaultExchangeDurability,
			QueueDurability:        DefaultQueueDurability,
			PrefetchCount:          DefaultPrefetchCount,
			MaxUndeliveredMessages: defaultMaxUndeliveredMessages,
		}
	})
}
