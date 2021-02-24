package rollbar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/gorilla/mux"
)

type Webhook struct {
	Path string
	acc  cua.Accumulator
}

func (rb *Webhook) Register(router *mux.Router, acc cua.Accumulator) {
	router.HandleFunc(rb.Path, rb.eventHandler).Methods("POST")
	log.Printf("I! Started the webhooks_rollbar on %s\n", rb.Path)
	rb.acc = acc
}

func (rb *Webhook) eventHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dummyEvent := &DummyEvent{}
	err = json.Unmarshal(data, dummyEvent)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := NewEvent(dummyEvent, data)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	rb.acc.AddFields("rollbar_webhooks", event.Fields(), event.Tags(), time.Now())

	w.WriteHeader(http.StatusOK)
}

func generateEvent(event Event, data []byte) (Event, error) {
	err := json.Unmarshal(data, event)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	return event, nil
}

func NewEvent(dummyEvent *DummyEvent, data []byte) (Event, error) {
	switch dummyEvent.EventName {
	case "new_item":
		return generateEvent(&NewItem{}, data)
	case "occurrence":
		return generateEvent(&Occurrence{}, data)
	case "deploy":
		return generateEvent(&Deploy{}, data)
	default:
		return nil, errors.New("Not implemented type: " + dummyEvent.EventName)
	}
}
