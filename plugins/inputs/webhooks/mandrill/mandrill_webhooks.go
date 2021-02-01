package mandrill

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/gorilla/mux"
)

type Webhook struct {
	Path string
	acc  cua.Accumulator
}

func (md *Webhook) Register(router *mux.Router, acc cua.Accumulator) {
	router.HandleFunc(md.Path, md.returnOK).Methods("HEAD")
	router.HandleFunc(md.Path, md.eventHandler).Methods("POST")

	log.Printf("I! Started the webhooks_mandrill on %s\n", md.Path)
	md.acc = acc
}

func (md *Webhook) returnOK(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (md *Webhook) eventHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data, err := url.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var events []Event
	err = json.Unmarshal([]byte(data.Get("mandrill_events")), &events)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, event := range events {
		md.acc.AddFields("mandrill_webhooks", event.Fields(), event.Tags(), time.Unix(event.TimeStamp, 0))
	}

	w.WriteHeader(http.StatusOK)
}
