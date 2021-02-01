package webhooks

import (
	"reflect"
	"testing"

	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/github"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/papertrail"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/particle"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks/rollbar"
)

func TestAvailableWebhooks(t *testing.T) {
	wb := NewWebhooks()
	expected := make([]Webhook, 0)
	if !reflect.DeepEqual(wb.AvailableWebhooks(), expected) {
		t.Errorf("expected to %v.\nGot %v", expected, wb.AvailableWebhooks())
	}

	wb.Github = &github.Webhook{Path: "/github"}
	expected = append(expected, wb.Github)
	if !reflect.DeepEqual(wb.AvailableWebhooks(), expected) {
		t.Errorf("expected to be %v.\nGot %v", expected, wb.AvailableWebhooks())
	}

	wb.Rollbar = &rollbar.Webhook{Path: "/rollbar"}
	expected = append(expected, wb.Rollbar)
	if !reflect.DeepEqual(wb.AvailableWebhooks(), expected) {
		t.Errorf("expected to be %v.\nGot %v", expected, wb.AvailableWebhooks())
	}

	wb.Papertrail = &papertrail.Webhook{Path: "/papertrail"}
	expected = append(expected, wb.Papertrail)
	if !reflect.DeepEqual(wb.AvailableWebhooks(), expected) {
		t.Errorf("expected to be %v.\nGot %v", expected, wb.AvailableWebhooks())
	}

	wb.Particle = &particle.Webhook{Path: "/particle"}
	expected = append(expected, wb.Particle)
	if !reflect.DeepEqual(wb.AvailableWebhooks(), expected) {
		t.Errorf("expected to be %v.\nGot %v", expected, wb.AvailableWebhooks())
	}
}
