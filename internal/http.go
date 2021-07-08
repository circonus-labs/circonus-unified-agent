package internal

import (
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"net/url"
)

type BasicAuthErrorFunc func(rw http.ResponseWriter)

// AuthHandler returns a http handler that requires HTTP basic auth
// credentials to match the given username and password.
func AuthHandler(username, password, realm string, onError BasicAuthErrorFunc) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return &basicAuthHandler{
			username: username,
			password: password,
			realm:    realm,
			onError:  onError,
			next:     h,
		}
	}
}

type basicAuthHandler struct {
	next     http.Handler
	onError  BasicAuthErrorFunc
	username string
	password string
	realm    string
}

func (h *basicAuthHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if h.username != "" || h.password != "" {
		reqUsername, reqPassword, ok := req.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(reqUsername), []byte(h.username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(reqPassword), []byte(h.password)) != 1 {

			rw.Header().Set("WWW-Authenticate", "Basic realm=\""+h.realm+"\"")
			h.onError(rw)
			http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	}

	h.next.ServeHTTP(rw, req)
}

type GenericAuthErrorFunc func(rw http.ResponseWriter)

// GenericAuthHandler returns a http handler that requires `Authorization: <credentials>`
func GenericAuthHandler(credentials string, onError GenericAuthErrorFunc) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return &genericAuthHandler{
			credentials: credentials,
			onError:     onError,
			next:        h,
		}
	}
}

// Generic auth scheme handler - exact match on `Authorization: <credentials>`
type genericAuthHandler struct {
	next        http.Handler
	onError     GenericAuthErrorFunc
	credentials string
}

func (h *genericAuthHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if h.credentials != "" {
		// Scheme checking
		authorization := req.Header.Get("Authorization")
		if subtle.ConstantTimeCompare([]byte(authorization), []byte(h.credentials)) != 1 {

			h.onError(rw)
			http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	}

	h.next.ServeHTTP(rw, req)
}

// ErrorFunc is a callback for writing an error response.
type ErrorFunc func(rw http.ResponseWriter, code int)

// IPRangeHandler returns a http handler that requires the remote address to be
// in the specified network.
func IPRangeHandler(network []*net.IPNet, onError ErrorFunc) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return &ipRangeHandler{
			network: network,
			onError: onError,
			next:    h,
		}
	}
}

type ipRangeHandler struct {
	next    http.Handler
	onError ErrorFunc
	network []*net.IPNet
}

func (h *ipRangeHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if len(h.network) == 0 {
		h.next.ServeHTTP(rw, req)
		return
	}

	remoteIPString, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		h.onError(rw, http.StatusForbidden)
		return
	}

	remoteIP := net.ParseIP(remoteIPString)
	if remoteIP == nil {
		h.onError(rw, http.StatusForbidden)
		return
	}

	for _, net := range h.network {
		if net.Contains(remoteIP) {
			h.next.ServeHTTP(rw, req)
			return
		}
	}

	h.onError(rw, http.StatusForbidden)
}

func OnClientError(client *http.Client, err error) {
	// Close connection after a timeout error. If this is a HTTP2
	// connection this ensures that next interval a new connection will be
	// used and name lookup will be performed.
	//   https://github.com/golang/go/issues/36026
	var uerr *url.Error
	if errors.As(err, &uerr) && uerr.Timeout() {
		client.CloseIdleConnections()
	}
}
