package particle

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/gorilla/mux"
)

type event struct {
	Name        string `json:"event"`
	Data        data   `json:"data"`
	TTL         int    `json:"ttl"`
	PublishedAt string `json:"published_at"`
	Database    string `json:"measurement"`
}

type data struct {
	Tags   map[string]string      `json:"tags"`
	Fields map[string]interface{} `json:"values"`
}

func newEvent() *event {
	return &event{
		Data: data{
			Tags:   make(map[string]string),
			Fields: make(map[string]interface{}),
		},
	}
}

func (e *event) Time() (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z", e.PublishedAt)
}

type Webhook struct {
	Path string
	acc  cua.Accumulator
}

func (rb *Webhook) Register(router *mux.Router, acc cua.Accumulator) {
	router.HandleFunc(rb.Path, rb.eventHandler).Methods("POST")
	rb.acc = acc
}

func (rb *Webhook) eventHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	e := newEvent()
	if err := json.NewDecoder(r.Body).Decode(e); err != nil {
		rb.acc.AddError(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pTime, err := e.Time()
	if err != nil {
		pTime = time.Now()
	}

	rb.acc.AddFields(e.Name, e.Data.Fields, e.Data.Tags, pTime)
	w.WriteHeader(http.StatusOK)
}
