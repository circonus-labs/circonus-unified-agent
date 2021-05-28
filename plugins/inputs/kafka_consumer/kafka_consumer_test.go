package kafkaconsumer

import (
	"context"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/kafka"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/parsers/value"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/require"
)

type FakeConsumerGroup struct {
	brokers []string
	group   string
	config  *sarama.Config

	handler sarama.ConsumerGroupHandler
	errors  chan error
}

func (g *FakeConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	g.handler = handler
	_ = g.handler.Setup(nil)
	return nil
}

func (g *FakeConsumerGroup) Errors() <-chan error {
	return g.errors
}

func (g *FakeConsumerGroup) Close() error {
	close(g.errors)
	return nil
}

type FakeCreator struct {
	ConsumerGroup *FakeConsumerGroup
}

func (c *FakeCreator) Create(brokers []string, group string, config *sarama.Config) (ConsumerGroup, error) {
	c.ConsumerGroup.brokers = brokers
	c.ConsumerGroup.group = group
	c.ConsumerGroup.config = config
	return c.ConsumerGroup, nil
}

func TestInit(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *KafkaConsumer
		initError bool
		check     func(t *testing.T, plugin *KafkaConsumer)
	}{
		{
			name:   "default config",
			plugin: &KafkaConsumer{},
			check: func(t *testing.T, plugin *KafkaConsumer) {
				require.Equal(t, plugin.ConsumerGroup, defaultConsumerGroup)
				require.Equal(t, plugin.MaxUndeliveredMessages, defaultMaxUndeliveredMessages)
				require.Equal(t, plugin.config.ClientID, "circonus")
				require.Equal(t, plugin.config.Consumer.Offsets.Initial, sarama.OffsetOldest)
			},
		},
		{
			name: "parses valid version string",
			plugin: &KafkaConsumer{
				ReadConfig: kafka.ReadConfig{
					Config: kafka.Config{
						Version: "1.0.0",
					},
				},
				Log: testutil.Logger{},
			},
			check: func(t *testing.T, plugin *KafkaConsumer) {
				require.Equal(t, plugin.config.Version, sarama.V1_0_0_0)
			},
		},
		{
			name: "invalid version string",
			plugin: &KafkaConsumer{
				ReadConfig: kafka.ReadConfig{
					Config: kafka.Config{
						Version: "100",
					},
				},
				Log: testutil.Logger{},
			},
			initError: true,
		},
		{
			name: "custom client_id",
			plugin: &KafkaConsumer{
				ReadConfig: kafka.ReadConfig{
					Config: kafka.Config{
						ClientID: "custom",
					},
				},
				Log: testutil.Logger{},
			},
			check: func(t *testing.T, plugin *KafkaConsumer) {
				require.Equal(t, plugin.config.ClientID, "custom")
			},
		},
		{
			name: "custom offset",
			plugin: &KafkaConsumer{
				Offset: "newest",
				Log:    testutil.Logger{},
			},
			check: func(t *testing.T, plugin *KafkaConsumer) {
				require.Equal(t, plugin.config.Consumer.Offsets.Initial, sarama.OffsetNewest)
			},
		},
		{
			name: "invalid offset",
			plugin: &KafkaConsumer{
				Offset: "middle",
				Log:    testutil.Logger{},
			},
			initError: true,
		},
		{
			name: "default tls without tls config",
			plugin: &KafkaConsumer{
				Log: testutil.Logger{},
			},
			check: func(t *testing.T, plugin *KafkaConsumer) {
				require.False(t, plugin.config.Net.TLS.Enable)
			},
		},
		{
			name: "default tls with a tls config",
			plugin: &KafkaConsumer{
				ReadConfig: kafka.ReadConfig{
					Config: kafka.Config{
						ClientConfig: tls.ClientConfig{
							InsecureSkipVerify: true,
						},
					},
				},
				Log: testutil.Logger{},
			},
			check: func(t *testing.T, plugin *KafkaConsumer) {
				require.True(t, plugin.config.Net.TLS.Enable)
			},
		},
		{
			name: "insecure tls",
			plugin: &KafkaConsumer{
				ReadConfig: kafka.ReadConfig{
					Config: kafka.Config{
						ClientConfig: tls.ClientConfig{
							InsecureSkipVerify: true,
						},
					},
				},
				Log: testutil.Logger{},
			},
			check: func(t *testing.T, plugin *KafkaConsumer) {
				require.True(t, plugin.config.Net.TLS.Enable)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cg := &FakeConsumerGroup{}
			tt.plugin.ConsumerCreator = &FakeCreator{ConsumerGroup: cg}
			err := tt.plugin.Init()
			if tt.initError {
				require.Error(t, err)
				return
			}

			tt.check(t, tt.plugin)
		})
	}
}

func TestStartStop(t *testing.T) {
	cg := &FakeConsumerGroup{errors: make(chan error)}
	plugin := &KafkaConsumer{
		ConsumerCreator: &FakeCreator{ConsumerGroup: cg},
		Log:             testutil.Logger{},
	}
	err := plugin.Init()
	require.NoError(t, err)

	var acc testutil.Accumulator
	err = plugin.Start(context.Background(), &acc)
	require.NoError(t, err)

	plugin.Stop()
}

type FakeConsumerGroupSession struct {
	ctx context.Context
}

func (s *FakeConsumerGroupSession) Claims() map[string][]int32 {
	panic("not implemented")
}

func (s *FakeConsumerGroupSession) MemberID() string {
	panic("not implemented")
}

func (s *FakeConsumerGroupSession) GenerationID() int32 {
	panic("not implemented")
}

func (s *FakeConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
	panic("not implemented")
}

func (s *FakeConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
	panic("not implemented")
}

func (s *FakeConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
}

func (s *FakeConsumerGroupSession) Context() context.Context {
	return s.ctx
}

func (s *FakeConsumerGroupSession) Commit() {
}

type FakeConsumerGroupClaim struct {
	messages chan *sarama.ConsumerMessage
}

func (c *FakeConsumerGroupClaim) Topic() string {
	panic("not implemented")
}

func (c *FakeConsumerGroupClaim) Partition() int32 {
	panic("not implemented")
}

func (c *FakeConsumerGroupClaim) InitialOffset() int64 {
	panic("not implemented")
}

func (c *FakeConsumerGroupClaim) HighWaterMarkOffset() int64 {
	panic("not implemented")
}

func (c *FakeConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return c.messages
}

func TestConsumerGroupHandler_Lifecycle(t *testing.T) {
	acc := &testutil.Accumulator{}
	parser := &value.Parser{MetricName: "cpu", DataType: "int"}
	cg := NewConsumerGroupHandler(acc, 1, parser)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := &FakeConsumerGroupSession{
		ctx: ctx,
	}
	var claim FakeConsumerGroupClaim
	var err error

	err = cg.Setup(session)
	require.NoError(t, err)

	cancel()
	err = cg.ConsumeClaim(session, &claim)
	require.NoError(t, err)

	err = cg.Cleanup(session)
	require.NoError(t, err)
}

func TestConsumerGroupHandler_ConsumeClaim(t *testing.T) {
	acc := &testutil.Accumulator{}
	parser := &value.Parser{MetricName: "cpu", DataType: "int"}
	cg := NewConsumerGroupHandler(acc, 1, parser)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := &FakeConsumerGroupSession{ctx: ctx}
	claim := &FakeConsumerGroupClaim{
		messages: make(chan *sarama.ConsumerMessage, 1),
	}

	err := cg.Setup(session)
	require.NoError(t, err)

	claim.messages <- &sarama.ConsumerMessage{
		Topic: "circonus",
		Value: []byte("42"),
	}

	go func() {
		err := cg.ConsumeClaim(session, claim)
		require.NoError(t, err)
	}()

	acc.Wait(1)
	cancel()

	err = cg.Cleanup(session)
	require.NoError(t, err)

	expected := []cua.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{},
			map[string]interface{}{
				"value": 42,
			},
			time.Now(),
		),
	}

	testutil.RequireMetricsEqual(t, expected, acc.GetCUAMetrics(), testutil.IgnoreTime())
}

func TestConsumerGroupHandler_Handle(t *testing.T) {
	tests := []struct {
		name          string
		maxMessageLen int
		topicTag      string
		msg           *sarama.ConsumerMessage
		expected      []cua.Metric
	}{
		{
			name: "happy path",
			msg: &sarama.ConsumerMessage{
				Topic: "circonus",
				Value: []byte("42"),
			},
			expected: []cua.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"value": 42,
					},
					time.Now(),
				),
			},
		},
		{
			name:          "message to long",
			maxMessageLen: 4,
			msg: &sarama.ConsumerMessage{
				Topic: "circonus",
				Value: []byte("12345"),
			},
			expected: []cua.Metric{},
		},
		{
			name: "parse error",
			msg: &sarama.ConsumerMessage{
				Topic: "circonus",
				Value: []byte("not an integer"),
			},
			expected: []cua.Metric{},
		},
		{
			name:     "add topic tag",
			topicTag: "topic",
			msg: &sarama.ConsumerMessage{
				Topic: "circonus",
				Value: []byte("42"),
			},
			expected: []cua.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{
						"topic": "circonus",
					},
					map[string]interface{}{
						"value": 42,
					},
					time.Now(),
				),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			acc := &testutil.Accumulator{}
			parser := &value.Parser{MetricName: "cpu", DataType: "int"}
			cg := NewConsumerGroupHandler(acc, 1, parser)
			cg.MaxMessageLen = tt.maxMessageLen
			cg.TopicTag = tt.topicTag

			ctx := context.Background()
			session := &FakeConsumerGroupSession{ctx: ctx}

			_ = cg.Reserve(ctx)
			_ = cg.Handle(session, tt.msg)

			testutil.RequireMetricsEqual(t, tt.expected, acc.GetCUAMetrics(), testutil.IgnoreTime())
		})
	}
}
