package inference

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HostGemini é o único destino de rede da IA (TRV-SEG-012, V4 §47.3).
const HostGemini = "generativelanguage.googleapis.com"

// baseURLGemini é fixada no código: o SDK aceitaria GOOGLE_GEMINI_BASE_URL do
// ambiente, o que desviaria os autos para outro servidor.
const baseURLGemini = "https://" + HostGemini + "/"

// ErrHostBloqueado: tentativa de falar com um destino que não é o da Gemini API.
type ErrHostBloqueado struct{ Destino string }

func (e ErrHostBloqueado) Error() string {
	return fmt.Sprintf("destino de rede não permitido para a IA: %s", e.Destino)
}

// TransporteRestrito só deixa passar requisições para a base configurada
// (em produção, https://generativelanguage.googleapis.com). Qualquer outro
// destino — inclusive um redirecionamento — é recusado antes da conexão.
type TransporteRestrito struct {
	Base    http.RoundTripper
	esquema string
	host    string // host[:porta]
	// Inspecionar confere o corpo exato antes do envio (barreira de
	// privacidade, camada 2). Erro → ErrInspecao e nada sai.
	Inspecionar func(corpo []byte) error
}

// ErrInspecao: a barreira de privacidade recusou o corpo da requisição.
type ErrInspecao struct{ Causa error }

func (e ErrInspecao) Error() string {
	return "envio recusado pela barreira de privacidade: " + e.Causa.Error()
}
func (e ErrInspecao) Unwrap() error { return e.Causa }

// NovoTransporteRestrito permite somente a origem de base (esquema + host).
func NovoTransporteRestrito(base http.RoundTripper, origem string) (*TransporteRestrito, error) {
	u, err := url.Parse(origem)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("origem inválida: %q", origem)
	}
	if base == nil {
		base = novoTransporteBase()
	}
	return &TransporteRestrito{Base: base, esquema: u.Scheme, host: u.Host}, nil
}

// RoundTrip confere esquema e host antes de enviar.
func (t *TransporteRestrito) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL == nil || r.URL.Scheme != t.esquema || !strings.EqualFold(r.URL.Host, t.host) {
		destino := ""
		if r.URL != nil {
			destino = r.URL.Scheme + "://" + r.URL.Host
		}
		if r.Body != nil {
			r.Body.Close()
		}
		return nil, ErrHostBloqueado{Destino: destino}
	}
	if t.Inspecionar != nil && r.Body != nil && r.Body != http.NoBody {
		corpo, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return nil, err
		}
		if err := t.Inspecionar(corpo); err != nil {
			return nil, ErrInspecao{Causa: err}
		}
		// Clona o pedido (RoundTrip não deve alterar o original) com o corpo relido.
		r = r.Clone(r.Context())
		r.Body = io.NopCloser(bytes.NewReader(corpo))
		r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(corpo)), nil }
		r.ContentLength = int64(len(corpo))
	}
	return t.Base.RoundTrip(r)
}

// novoTransporteBase: TLS ≥ 1.2, proxy das variáveis de ambiente (HTTPS_PROXY),
// tempos de conexão limitados. Sem cache nem log de nada.
func novoTransporteBase() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 20 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 0, // a geração pode demorar; o limite é o Timeout do pedido
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          4,
		IdleConnTimeout:       90 * time.Second,
	}
}

// novoClienteHTTP monta o cliente com o transporte restrito à origem dada.
func novoClienteHTTP(origem string, inspecionar func([]byte) error) (*http.Client, error) {
	tr, err := NovoTransporteRestrito(nil, origem)
	if err != nil {
		return nil, err
	}
	tr.Inspecionar = inspecionar
	return &http.Client{
		Transport: tr,
		// Redirecionamento nunca é seguido: a Gemini API não redireciona, e um
		// redirecionamento levaria o corpo (autos) a outro destino.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}, nil
}
