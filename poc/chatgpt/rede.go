package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Hosts permitidos (documentação SIWC + discovery de 09/11/2026).
const (
	hostAuth = "auth.openai.com"
	hostAPI  = "api.openai.com"
)

var errHostBloqueado = errors.New("destino fora da lista permitida")

// transporteRestrito só deixa sair HTTPS para os hosts permitidos.
type transporteRestrito struct {
	permitidos map[string]bool
	base       http.RoundTripper
}

func (t transporteRestrito) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != "https" || !t.permitidos[r.URL.Hostname()] || (r.URL.Port() != "" && r.URL.Port() != "443") {
		return nil, fmt.Errorf("%w: %s://%s", errHostBloqueado, r.URL.Scheme, r.URL.Host)
	}
	return t.base.RoundTrip(r)
}

// novoCliente: TLS 1.2+, sem redirecionamento, só os hosts informados.
// timeout = 0 → sem prazo total (o fluxo SSE é controlado por inatividade).
func novoCliente(timeout time.Duration, hosts ...string) *http.Client {
	perm := map[string]bool{}
	for _, h := range hosts {
		perm[h] = true
	}
	base := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout: 15 * time.Second,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        4,
		IdleConnTimeout:     60 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transporteRestrito{permitidos: perm, base: base},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
