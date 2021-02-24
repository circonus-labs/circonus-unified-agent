package testutil

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
)

type PKI struct {
	path string
}

func NewPKI(path string) *PKI {
	return &PKI{path: path}
}

func (p *PKI) TLSClientConfig() *tls.ClientConfig {
	return &tls.ClientConfig{
		TLSCA:   p.CACertPath(),
		TLSCert: p.ClientCertPath(),
		TLSKey:  p.ClientKeyPath(),
	}
}

func (p *PKI) TLSServerConfig() *tls.ServerConfig {
	return &tls.ServerConfig{
		TLSAllowedCACerts: []string{p.CACertPath()},
		TLSCert:           p.ServerCertPath(),
		TLSKey:            p.ServerKeyPath(),
		TLSCipherSuites:   []string{p.CipherSuite()},
		TLSMinVersion:     p.TLSMinVersion(),
		TLSMaxVersion:     p.TLSMaxVersion(),
	}
}

func (p *PKI) ReadCACert() string {
	return readCertificate(p.CACertPath())
}

func (p *PKI) CACertPath() string {
	return path.Join(p.path, "cacert.pem")
}

func (p *PKI) CipherSuite() string {
	return "TLS_RSA_WITH_3DES_EDE_CBC_SHA"
}

func (p *PKI) TLSMinVersion() string {
	return "TLS11"
}

func (p *PKI) TLSMaxVersion() string {
	return "TLS12"
}

func (p *PKI) ReadClientCert() string {
	return readCertificate(p.ClientCertPath())
}

func (p *PKI) ClientCertPath() string {
	return path.Join(p.path, "clientcert.pem")
}

func (p *PKI) ReadClientKey() string {
	return readCertificate(p.ClientKeyPath())
}

func (p *PKI) ClientKeyPath() string {
	return path.Join(p.path, "clientkey.pem")
}

func (p *PKI) ReadServerCert() string {
	return readCertificate(p.ServerCertPath())
}

func (p *PKI) ServerCertPath() string {
	return path.Join(p.path, "servercert.pem")
}

func (p *PKI) ReadServerKey() string {
	return readCertificate(p.ServerKeyPath())
}

func (p *PKI) ServerKeyPath() string {
	return path.Join(p.path, "serverkey.pem")
}

func readCertificate(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		panic(fmt.Sprintf("opening %q: %v", filename, err))
	}
	octets, err := io.ReadAll(file)
	if err != nil {
		panic(fmt.Sprintf("reading %q: %v", filename, err))
	}
	return string(octets)
}
