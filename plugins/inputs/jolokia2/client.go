package jolokia2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
)

type Client struct {
	client *http.Client
	config *ClientConfig
	URL    string
}

type ClientConfig struct {
	ProxyConfig *ProxyConfig
	Username    string
	Password    string
	tls.ClientConfig
	ResponseTimeout time.Duration
}

type ProxyConfig struct {
	DefaultTargetUsername string
	DefaultTargetPassword string
	Targets               []ProxyTargetConfig
}

type ProxyTargetConfig struct {
	Username string
	Password string
	URL      string
}

type ReadRequest struct {
	Mbean      string
	Path       string
	Attributes []string
}

type ReadResponse struct {
	Value             interface{}
	RequestMbean      string
	RequestPath       string
	RequestTarget     string
	RequestAttributes []string
	Status            int
}

//	Jolokia JSON request object. Example: {
//	  "type": "read",
//	  "mbean: "java.lang:type="Runtime",
//	  "attribute": "Uptime",
//	  "target": {
//	    "url: "service:jmx:rmi:///jndi/rmi://target:9010/jmxrmi"
//	  }
//	}
type jolokiaRequest struct {
	Attribute interface{}    `json:"attribute,omitempty"`
	Target    *jolokiaTarget `json:"target,omitempty"`
	Type      string         `json:"type"`
	Mbean     string         `json:"mbean"`
	Path      string         `json:"path,omitempty"`
}

type jolokiaTarget struct {
	URL      string `json:"url"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

//	Jolokia JSON response object. Example: {
//	  "request": {
//	    "type": "read"
//	    "mbean": "java.lang:type=Runtime",
//	    "attribute": "Uptime",
//	    "target": {
//	      "url": "service:jmx:rmi:///jndi/rmi://target:9010/jmxrmi"
//	    }
//	  },
//	  "value": 1214083,
//	  "timestamp": 1488059309,
//	  "status": 200
//	}
type jolokiaResponse struct {
	Value   interface{}    `json:"value"`
	Request jolokiaRequest `json:"request"`
	Status  int            `json:"status"`
}

func NewClient(url string, config *ClientConfig) (*Client, error) {
	tlsConfig, err := config.ClientConfig.TLSConfig()
	if err != nil {
		return nil, fmt.Errorf("TLSConfig: %w", err)
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: config.ResponseTimeout,
		TLSClientConfig:       tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.ResponseTimeout,
	}

	return &Client{
		URL:    url,
		config: config,
		client: client,
	}, nil
}

func (c *Client) read(requests []ReadRequest) ([]ReadResponse, error) {
	jrequests := makeJolokiaRequests(requests, c.config.ProxyConfig)
	requestBody, err := json.Marshal(jrequests)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}

	requestURL, err := formatReadURL(c.URL, c.config.Username, c.config.Password)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("unable to create new request '%s': %w", requestURL, err)
	}

	req.Header.Add("Content-type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http req do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Response from url \"%s\" has status code %d (%s), expected %d (%s)",
			c.URL, resp.StatusCode, http.StatusText(resp.StatusCode), http.StatusOK, http.StatusText(http.StatusOK))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("readall: %w", err)
	}

	var jresponses []jolokiaResponse
	if err = json.Unmarshal(responseBody, &jresponses); err != nil {
		return nil, fmt.Errorf("Error decoding JSON response: %w: %s", err, responseBody)
	}

	return makeReadResponses(jresponses), nil
}

func makeJolokiaRequests(rrequests []ReadRequest, proxyConfig *ProxyConfig) []jolokiaRequest {
	jrequests := make([]jolokiaRequest, 0)
	if proxyConfig == nil {
		for _, rr := range rrequests {
			jrequests = append(jrequests, makeJolokiaRequest(rr, nil))
		}
	} else {
		for _, t := range proxyConfig.Targets {
			if t.Username == "" {
				t.Username = proxyConfig.DefaultTargetUsername
			}
			if t.Password == "" {
				t.Password = proxyConfig.DefaultTargetPassword
			}

			for _, rr := range rrequests {
				jtarget := &jolokiaTarget{
					URL:      t.URL,
					User:     t.Username,
					Password: t.Password,
				}

				jrequests = append(jrequests, makeJolokiaRequest(rr, jtarget))
			}
		}
	}

	return jrequests
}

func makeJolokiaRequest(rrequest ReadRequest, jtarget *jolokiaTarget) jolokiaRequest {
	jrequest := jolokiaRequest{
		Type:   "read",
		Mbean:  rrequest.Mbean,
		Path:   rrequest.Path,
		Target: jtarget,
	}

	if len(rrequest.Attributes) == 1 {
		jrequest.Attribute = rrequest.Attributes[0]
	}
	if len(rrequest.Attributes) > 1 {
		jrequest.Attribute = rrequest.Attributes
	}

	return jrequest
}

func makeReadResponses(jresponses []jolokiaResponse) []ReadResponse {
	rresponses := make([]ReadResponse, 0)

	for _, jr := range jresponses {
		rrequest := ReadRequest{
			Mbean:      jr.Request.Mbean,
			Path:       jr.Request.Path,
			Attributes: []string{},
		}

		attrValue := jr.Request.Attribute
		if attrValue != nil {
			attribute, ok := attrValue.(string)
			if ok {
				rrequest.Attributes = []string{attribute}
			} else {
				attributes, _ := attrValue.([]interface{})
				rrequest.Attributes = make([]string, len(attributes))
				for i, attr := range attributes {
					rrequest.Attributes[i] = attr.(string)
				}
			}
		}
		rresponse := ReadResponse{
			Value:             jr.Value,
			Status:            jr.Status,
			RequestMbean:      rrequest.Mbean,
			RequestAttributes: rrequest.Attributes,
			RequestPath:       rrequest.Path,
		}
		if jtarget := jr.Request.Target; jtarget != nil {
			rresponse.RequestTarget = jtarget.URL
		}

		rresponses = append(rresponses, rresponse)
	}

	return rresponses
}

func formatReadURL(configURL, username, password string) (string, error) {
	parsedURL, err := url.Parse(configURL)
	if err != nil {
		return "", fmt.Errorf("url parse (%s): %w", configURL, err)
	}

	readURL := url.URL{
		Host:   parsedURL.Host,
		Scheme: parsedURL.Scheme,
	}

	if username != "" || password != "" {
		readURL.User = url.UserPassword(username, password)
	}

	readURL.Path = path.Join(parsedURL.Path, "read")
	readURL.Query().Add("ignoreErrors", "true")
	return readURL.String(), nil
}
