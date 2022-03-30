package icinga2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type Icinga2 struct {
	Server          string
	ObjectType      string
	Username        string
	Password        string
	ResponseTimeout internal.Duration
	tls.ClientConfig

	Log cua.Logger

	client *http.Client
}

type Result struct {
	Results []Object `json:"results"`
}

type Object struct {
	Attrs Attribute  `json:"attrs"`
	Name  string     `json:"name"`
	Joins struct{}   `json:"joins"`
	Meta  struct{}   `json:"meta"`
	Type  ObjectType `json:"type"`
}

type Attribute struct {
	CheckCommand string  `json:"check_command"`
	DisplayName  string  `json:"display_name"`
	Name         string  `json:"name"`
	State        float64 `json:"state"`
	HostName     string  `json:"host_name"`
}

var levels = []string{"ok", "warning", "critical", "unknown"}

const objTypeServices = "services"

type ObjectType string

var sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## Required Icinga2 server address
  # server = "https://localhost:5665"
  
  ## Required Icinga2 object type ("services" or "hosts")
  # object_type = "services"

  ## Credentials for basic HTTP authentication
  # username = "admin"
  # password = "admin"

  ## Maximum time to receive response.
  # response_timeout = "5s"

  ## Optional TLS Config
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.pem"
  # tls_key = "/etc/circonus-unified-agent/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = true
  `

func (i *Icinga2) Description() string {
	return "Gather Icinga2 status"
}

func (i *Icinga2) SampleConfig() string {
	return sampleConfig
}

func (i *Icinga2) GatherStatus(acc cua.Accumulator, checks []Object) {
	for _, check := range checks {
		url, err := url.Parse(i.Server)
		if err != nil {
			i.Log.Error(err.Error())
			continue
		}

		state := int64(check.Attrs.State)

		fields := map[string]interface{}{
			"name":       check.Attrs.Name,
			"state_code": state,
		}

		// source is dependent on 'services' or 'hosts' check
		source := check.Attrs.Name
		if i.ObjectType == objTypeServices {
			source = check.Attrs.HostName
		}

		tags := map[string]string{
			"display_name":  check.Attrs.DisplayName,
			"check_command": check.Attrs.CheckCommand,
			"source":        source,
			"state":         levels[state],
			"server":        url.Hostname(),
			"scheme":        url.Scheme,
			"port":          url.Port(),
		}

		acc.AddFields(fmt.Sprintf("icinga2_%s", i.ObjectType), fields, tags)
	}
}

func (i *Icinga2) createHTTPClient() (*http.Client, error) {
	tlsCfg, err := i.ClientConfig.TLSConfig()
	if err != nil {
		return nil, fmt.Errorf("TLSConfig: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
		Timeout: i.ResponseTimeout.Duration,
	}

	return client, nil
}

func (i *Icinga2) Gather(ctx context.Context, acc cua.Accumulator) error {
	if i.ResponseTimeout.Duration < time.Second {
		i.ResponseTimeout.Duration = time.Second * 5
	}

	if i.client == nil {
		client, err := i.createHTTPClient()
		if err != nil {
			return err
		}
		i.client = client
	}

	requestURL := "%s/v1/objects/%s?attrs=name&attrs=display_name&attrs=state&attrs=check_command"

	// Note: attrs=host_name is only valid for 'services' requests, using check.Attrs.HostName for the host
	//       'hosts' requests will need to use attrs=name only, using check.Attrs.Name for the host
	if i.ObjectType == objTypeServices {
		requestURL += "&attrs=host_name"
	}

	url := fmt.Sprintf(requestURL, i.Server, i.ObjectType)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("new request (%s): %w", url, err)
	}

	if i.Username != "" {
		req.SetBasicAuth(i.Username, i.Password)
	}

	resp, err := i.client.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}

	defer resp.Body.Close()

	result := Result{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}

	i.GatherStatus(acc, result.Results)

	return nil
}

func init() {
	inputs.Add("icinga2", func() cua.Input {
		return &Icinga2{
			Server:          "https://localhost:5665",
			ObjectType:      "services",
			ResponseTimeout: internal.Duration{Duration: time.Second * 5},
		}
	})
}
