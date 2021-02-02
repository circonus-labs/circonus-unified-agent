package papertrail

import (
	"encoding/json"
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

func (pt *Webhook) Register(router *mux.Router, acc cua.Accumulator) {
	router.HandleFunc(pt.Path, pt.eventHandler).Methods("POST")
	log.Printf("I! Started the papertrail_webhook on %s", pt.Path)
	pt.acc = acc
}

func (pt *Webhook) eventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}

	data := r.PostFormValue("payload")
	if data == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var payload Payload
	err := json.Unmarshal([]byte(data), &payload)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	switch {
	case payload.Events != nil: // Handle event-based payload
		for _, e := range payload.Events {
			// Warning: Duplicate event timestamps will overwrite each other
			tags := map[string]string{
				"host":  e.Hostname,
				"event": payload.SavedSearch.Name,
			}
			fields := map[string]interface{}{
				"count": uint64(1),
			}
			pt.acc.AddFields("papertrail", fields, tags, e.ReceivedAt)
		}
	case payload.Counts != nil: // Handle count-based payload
		for _, c := range payload.Counts {
			for ts, count := range *c.TimeSeries {
				tags := map[string]string{
					"host":  c.SourceName,
					"event": payload.SavedSearch.Name,
				}
				fields := map[string]interface{}{
					"count": count,
				}
				pt.acc.AddFields("papertrail", fields, tags, time.Unix(ts, 0))
			}
		}
	default:
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
