package circonus

import (
	"crypto/x509"
	"fmt"
	"os"

	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/go-apiclient"
)

var ch *Circonus

type Circonus struct {
	circCfg *config.CirconusConfig
	apiCfg  *apiclient.Config
	ready   bool
}

func Initialize(cfg *config.CirconusConfig, err error) error {
	if ch != nil {
		return nil // already initialized
	}
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("invalid circonus config (nil)")
	}

	c := &Circonus{circCfg: cfg}

	if c.circCfg.APIToken == "" {
		return fmt.Errorf("unable to initialize circonus module: API Token is required")
	}

	if c.circCfg.APIApp == "" {
		c.circCfg.APIApp = "circonus-unified-agent"
	}

	c.apiCfg = &apiclient.Config{
		TokenKey:      c.circCfg.APIToken,
		TokenApp:      c.circCfg.APIApp,
		MaxRetries:    4,
		MinRetryDelay: "10s", // for race where api returns 500 but check is created, if called too fast a duplicate check is created
		MaxRetryDelay: "20s",
	}

	if c.circCfg.APIURL != "" {
		c.apiCfg.URL = c.circCfg.APIURL
	}

	if c.circCfg.APITLSCA != "" {
		cp := x509.NewCertPool()
		cert, err := os.ReadFile(c.circCfg.APITLSCA)
		if err != nil {
			return fmt.Errorf("unable to load api ca file (%s): %w", c.circCfg.APITLSCA, err)
		}
		if !cp.AppendCertsFromPEM(cert) {
			return fmt.Errorf("unable to parse api ca file (%s): %w", c.circCfg.APITLSCA, err)
		}
		c.apiCfg.CACert = cp
	}

	c.ready = true

	ch = c

	return nil
}

func GetAPIClient() (*apiclient.API, error) {
	if !ch.ready {
		return nil, fmt.Errorf("invalid agent circonus config")
	}

	client, err := apiclient.New(ch.apiCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize circonus api client: %w", err)
	}

	return client, nil
}
