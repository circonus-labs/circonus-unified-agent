// Package x509_cert reports metrics from an SSL certificate.
package x509cert

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	_tls "github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

const sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## List certificate sources
  sources = ["/etc/ssl/certs/ssl-cert-snakeoil.pem", "tcp://example.org:443"]

  ## Timeout for SSL connection
  # timeout = "5s"

  ## Pass a different name into the TLS request (Server Name Indication)
  ##   example: server_name = "myhost.example.org"
  # server_name = ""

  ## Optional TLS Config
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.pem"
  # tls_key = "/etc/circonus-unified-agent/key.pem"
`
const description = "Reads metrics from a SSL certificate"

// X509Cert holds the configuration of the plugin.
type X509Cert struct {
	tlsCfg     *tls.Config
	ServerName string `toml:"server_name"`
	_tls.ClientConfig
	Sources []string          `toml:"sources"`
	Timeout internal.Duration `toml:"timeout"`
}

// Description returns description of the plugin.
func (c *X509Cert) Description() string {
	return description
}

// SampleConfig returns configuration sample for the plugin.
func (c *X509Cert) SampleConfig() string {
	return sampleConfig
}

func (c *X509Cert) locationToURL(location string) (*url.URL, error) {
	if strings.HasPrefix(location, "/") {
		location = "file://" + location
	}
	if strings.Index(location, ":\\") == 1 {
		location = "file://" + filepath.ToSlash(location)
	}

	u, err := url.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cert location - %w", err)
	}

	return u, nil
}

func (c *X509Cert) getCert(u *url.URL, timeout time.Duration) ([]*x509.Certificate, error) {
	switch u.Scheme {
	case "https":
		u.Scheme = "tcp"
		fallthrough
	case "udp", "udp4", "udp6":
		fallthrough
	case "tcp", "tcp4", "tcp6":
		ipConn, err := net.DialTimeout(u.Scheme, u.Host, timeout)
		if err != nil {
			return nil, fmt.Errorf("dial (%s %s): %w", u.Scheme, u.Host, err)
		}
		defer ipConn.Close()

		if c.ServerName == "" {
			c.tlsCfg.ServerName = u.Hostname()
		} else {
			c.tlsCfg.ServerName = c.ServerName
		}

		c.tlsCfg.InsecureSkipVerify = true
		conn := tls.Client(ipConn, c.tlsCfg)
		defer conn.Close()

		hsErr := conn.Handshake()
		if hsErr != nil {
			return nil, fmt.Errorf("conn handshake: %w", hsErr)
		}

		certs := conn.ConnectionState().PeerCertificates

		return certs, nil
	case "file":
		content, err := os.ReadFile(u.Path)
		if err != nil {
			return nil, fmt.Errorf("readfile (%s): %w", u.Path, err)
		}
		var certs []*x509.Certificate
		for {
			block, rest := pem.Decode(bytes.TrimSpace(content))
			if block == nil {
				return nil, fmt.Errorf("failed to parse certificate PEM")
			}

			if block.Type == "CERTIFICATE" {
				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return nil, fmt.Errorf("parse cert: %w", err)
				}
				certs = append(certs, cert)
			}
			if len(rest) == 0 {
				break
			}
			content = rest
		}
		return certs, nil
	default:
		return nil, fmt.Errorf("unsupported scheme '%s' in location %s", u.Scheme, u.String())
	}
}

func getFields(cert *x509.Certificate, now time.Time) map[string]interface{} {
	age := int(now.Sub(cert.NotBefore).Seconds())
	expiry := int(cert.NotAfter.Sub(now).Seconds())
	startdate := cert.NotBefore.Unix()
	enddate := cert.NotAfter.Unix()

	fields := map[string]interface{}{
		"age":       age,
		"expiry":    expiry,
		"startdate": startdate,
		"enddate":   enddate,
	}

	return fields
}

func getTags(cert *x509.Certificate, location string) map[string]string {
	tags := map[string]string{
		"source":               location,
		"common_name":          cert.Subject.CommonName,
		"serial_number":        cert.SerialNumber.Text(16),
		"signature_algorithm":  cert.SignatureAlgorithm.String(),
		"public_key_algorithm": cert.PublicKeyAlgorithm.String(),
	}

	if len(cert.Subject.Organization) > 0 {
		tags["organization"] = cert.Subject.Organization[0]
	}
	if len(cert.Subject.OrganizationalUnit) > 0 {
		tags["organizational_unit"] = cert.Subject.OrganizationalUnit[0]
	}
	if len(cert.Subject.Country) > 0 {
		tags["country"] = cert.Subject.Country[0]
	}
	if len(cert.Subject.Province) > 0 {
		tags["province"] = cert.Subject.Province[0]
	}
	if len(cert.Subject.Locality) > 0 {
		tags["locality"] = cert.Subject.Locality[0]
	}

	tags["issuer_common_name"] = cert.Issuer.CommonName
	tags["issuer_serial_number"] = cert.Issuer.SerialNumber

	san := append(cert.DNSNames, cert.EmailAddresses...) //nolint:gocritic
	for _, ip := range cert.IPAddresses {
		san = append(san, ip.String())
	}
	for _, uri := range cert.URIs {
		san = append(san, uri.String())
	}
	tags["san"] = strings.Join(san, ",")

	return tags
}

// Gather adds metrics into the accumulator.
func (c *X509Cert) Gather(ctx context.Context, acc cua.Accumulator) error {
	now := time.Now()

	for _, location := range c.Sources {
		u, err := c.locationToURL(location)
		if err != nil {
			acc.AddError(err)
			return nil
		}

		certs, err := c.getCert(u, c.Timeout.Duration*time.Second)
		if err != nil {
			acc.AddError(fmt.Errorf("cannot get SSL cert '%s': %w", location, err))
		}

		for i, cert := range certs {
			fields := getFields(cert, now)
			tags := getTags(cert, location)

			// The first certificate is the leaf/end-entity certificate which needs DNS
			// name validation against the URL hostname.
			opts := x509.VerifyOptions{
				Intermediates: x509.NewCertPool(),
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
			}
			if i == 0 {
				if c.ServerName == "" {
					opts.DNSName = u.Hostname()
				} else {
					opts.DNSName = c.ServerName
				}
				for j, cert := range certs {
					if j != 0 {
						opts.Intermediates.AddCert(cert)
					}
				}
			}
			if c.tlsCfg.RootCAs != nil {
				opts.Roots = c.tlsCfg.RootCAs
			}

			_, err = cert.Verify(opts)
			if err == nil {
				tags["verification"] = "valid"
				fields["verification_code"] = 0
			} else {
				tags["verification"] = "invalid"
				fields["verification_code"] = 1
				fields["verification_error"] = err.Error()
			}

			acc.AddFields("x509_cert", fields, tags)
		}
	}

	return nil
}

func (c *X509Cert) Init() error {
	tlsCfg, err := c.ClientConfig.TLSConfig()
	if err != nil {
		return fmt.Errorf("TLSConfig: %w", err)
	}
	if tlsCfg == nil {
		tlsCfg = &tls.Config{MinVersion: tls.VersionTLS12} // #nosec G402 // G402: TLS MinVersion too low.
	}

	c.tlsCfg = tlsCfg

	return nil
}

func init() {
	inputs.Add("x509_cert", func() cua.Input {
		return &X509Cert{
			Sources: []string{},
			Timeout: internal.Duration{Duration: 5},
		}
	})
}
