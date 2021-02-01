package filestack

import (
	"encoding/json"
	"io/ioutil"
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

func (fs *Webhook) Register(router *mux.Router, acc cua.Accumulator) {
	router.HandleFunc(fs.Path, fs.eventHandler).Methods("POST")

	log.Printf("I! Started the webhooks_filestack on %s\n", fs.Path)
	fs.acc = acc
}

func (fs *Webhook) eventHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event := &Event{}
	err = json.Unmarshal(body, event)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fs.acc.AddFields("filestack_webhooks", event.Fields(), event.Tags(), time.Unix(event.TimeStamp, 0))

	w.WriteHeader(http.StatusOK)
}
