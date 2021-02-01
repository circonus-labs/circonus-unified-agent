package filestack

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/circonus-labs/circonus-unified-agent/testutil"
)

func postWebhooks(md *Webhook, eventBody string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("POST", "/filestack", strings.NewReader(eventBody))
	w := httptest.NewRecorder()

	md.eventHandler(w, req)

	return w
}

func TestDialogEvent(t *testing.T) {
	var acc testutil.Accumulator
	fs := &Webhook{Path: "/filestack", acc: &acc}
	resp := postWebhooks(fs, DialogOpenJSON())
	if resp.Code != http.StatusOK {
		t.Errorf("POST returned HTTP status code %v.\nExpected %v", resp.Code, http.StatusOK)
	}

	fields := map[string]interface{}{
		"id": "102",
	}

	tags := map[string]string{
		"action": "fp.dialog",
	}

	acc.AssertContainsTaggedFields(t, "filestack_webhooks", fields, tags)
}

func TestParseError(t *testing.T) {
	fs := &Webhook{Path: "/filestack"}
	resp := postWebhooks(fs, "")
	if resp.Code != http.StatusBadRequest {
		t.Errorf("POST returned HTTP status code %v.\nExpected %v", resp.Code, http.StatusBadRequest)
	}
}

func TestUploadEvent(t *testing.T) {
	var acc testutil.Accumulator
	fs := &Webhook{Path: "/filestack", acc: &acc}
	resp := postWebhooks(fs, UploadJSON())
	if resp.Code != http.StatusOK {
		t.Errorf("POST returned HTTP status code %v.\nExpected %v", resp.Code, http.StatusOK)
	}

	fields := map[string]interface{}{
		"id": "100946",
	}

	tags := map[string]string{
		"action": "fp.upload",
	}

	acc.AssertContainsTaggedFields(t, "filestack_webhooks", fields, tags)
}

func TestVideoConversionEvent(t *testing.T) {
	var acc testutil.Accumulator
	fs := &Webhook{Path: "/filestack", acc: &acc}
	resp := postWebhooks(fs, VideoConversionJSON())
	if resp.Code != http.StatusBadRequest {
		t.Errorf("POST returned HTTP status code %v.\nExpected %v", resp.Code, http.StatusBadRequest)
	}
}
