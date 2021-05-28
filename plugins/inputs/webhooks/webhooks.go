package webhooks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"reflect"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/filestack"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/github"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/mandrill"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/papertrail"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/particle"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/rollbar"
	"github.com/gorilla/mux"
)

type Webhook interface {
	Register(router *mux.Router, acc cua.Accumulator)
}

func init() {
	inputs.Add("webhooks", func() cua.Input { return NewWebhooks() })
}

type Webhooks struct {
	ServiceAddress string

	Github     *github.Webhook
	Filestack  *filestack.Webhook
	Mandrill   *mandrill.Webhook
	Rollbar    *rollbar.Webhook
	Papertrail *papertrail.Webhook
	Particle   *particle.Webhook

	srv *http.Server
}

func NewWebhooks() *Webhooks {
	return &Webhooks{}
}

func (*Webhooks) SampleConfig() string {
	return `
  ## Address and port to host Webhook listener on
  service_address = ":1619"

  [inputs.webhooks.filestack]
    path = "/filestack"

  [inputs.webhooks.github]
    path = "/github"
    # secret = ""

  [inputs.webhooks.mandrill]
    path = "/mandrill"

  [inputs.webhooks.rollbar]
    path = "/rollbar"

  [inputs.webhooks.papertrail]
    path = "/papertrail"

  [inputs.webhooks.particle]
    path = "/particle"
`
}

func (*Webhooks) Description() string {
	return "A Webhooks Event collector"
}

func (*Webhooks) Gather(_ context.Context, _ cua.Accumulator) error {
	return nil
}

// Looks for fields which implement Webhook interface
func (wh *Webhooks) AvailableWebhooks() []Webhook {
	webhooks := make([]Webhook, 0)
	s := reflect.ValueOf(wh).Elem()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)

		if !f.CanInterface() {
			continue
		}

		if wbPlugin, ok := f.Interface().(Webhook); ok {
			if !reflect.ValueOf(wbPlugin).IsNil() {
				webhooks = append(webhooks, wbPlugin)
			}
		}
	}

	return webhooks
}

func (wh *Webhooks) Start(ctx context.Context, acc cua.Accumulator) error {
	r := mux.NewRouter()

	for _, webhook := range wh.AvailableWebhooks() {
		webhook.Register(r, acc)
	}

	wh.srv = &http.Server{Handler: r}

	ln, err := net.Listen("tcp", wh.ServiceAddress)
	if err != nil {
		log.Fatalf("E! Error starting server: %v", err)
		return fmt.Errorf("listen (%s): %w", wh.ServiceAddress, err)

	}

	go func() {
		if err := wh.srv.Serve(ln); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				acc.AddError(fmt.Errorf("E! Error listening: %w", err))
			}
		}
	}()

	log.Printf("I! Started the webhooks service on %s\n", wh.ServiceAddress)

	return nil
}

func (wh *Webhooks) Stop() {
	wh.srv.Close()
	log.Println("I! Stopping the Webhooks service")
}
