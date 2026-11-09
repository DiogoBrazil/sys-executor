package inference

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// falhaGemini é a decisão sobre uma falha de chamada: o erro do catálogo e
// se vale repetir (e após quanto tempo, quando o provedor indica).
type falhaGemini struct {
	erro      error
	repetir   bool
	aguardar  time.Duration // RetryInfo do provedor; 0 = backoff próprio
	cancelado bool
}

// esperaMaxRetryInfo: acima disso a cota não volta logo; não se espera.
const esperaMaxRetryInfo = 60 * time.Second

// classificar converte o erro do SDK/rede em erro do catálogo (IA-0xx) sem
// expor a chave, e diz se a falha é transitória.
func classificar(ctx context.Context, err error, chave string) falhaGemini {
	if err == nil {
		return falhaGemini{}
	}
	if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
		return falhaGemini{erro: context.Canceled, cancelado: true}
	}
	var bloq ErrHostBloqueado
	if errors.As(err, &bloq) {
		return falhaGemini{erro: erros.Com(erros.InternoInesperado, bloq)}
	}
	var api genai.APIError
	if errors.As(err, &api) {
		return classificarAPI(api, chave)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return falhaGemini{erro: erros.Com(erros.IATempoEsgotado, sanear(err, chave))}
	}
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return falhaGemini{erro: erros.Com(erros.IATempoEsgotado, sanear(err, chave))}
	}
	var uerr *url.Error
	var operr *net.OpError
	var dnserr *net.DNSError
	if errors.As(err, &dnserr) || errors.As(err, &operr) || errors.As(err, &uerr) {
		return falhaGemini{erro: erros.Com(erros.IASemConexao, sanear(err, chave)), repetir: true}
	}
	// Não reconhecido: o tipo entra na causa (só diagnóstico em memória).
	return falhaGemini{erro: erros.Com(erros.IAIndisponivel, sanear(fmt.Errorf("%T: %w", err, err), chave))}
}

func classificarAPI(e genai.APIError, chave string) falhaGemini {
	causa := sanear(fmt.Errorf("gemini %d %s: %s", e.Code, e.Status, e.Message), chave)
	motivo := razao(e)
	// Créditos pré-pagos esgotados ou faturamento pendente: não adianta
	// repetir nem esperar (o código HTTP varia; vale o texto e o status).
	if semCreditos(e, motivo) {
		return falhaGemini{erro: erros.Com(erros.IASemCreditos, causa)}
	}
	switch {
	case e.Code == 400 && (motivo == "API_KEY_INVALID" || strings.Contains(strings.ToLower(e.Message), "api key not valid")):
		return falhaGemini{erro: erros.Com(erros.IAChaveInvalida, causa)}
	case e.Code == 401 || e.Status == "UNAUTHENTICATED":
		return falhaGemini{erro: erros.Com(erros.IAChaveInvalida, causa)}
	case e.Code == 400 && e.Status == "FAILED_PRECONDITION":
		// Ex.: nível gratuito indisponível na região, faturamento exigido.
		return falhaGemini{erro: erros.Com(erros.IAPermissaoNegada, causa)}
	case e.Code == 403 || e.Status == "PERMISSION_DENIED":
		if motivo == "API_KEY_INVALID" || strings.Contains(motivo, "API_KEY") {
			return falhaGemini{erro: erros.Com(erros.IAChaveInvalida, causa)}
		}
		return falhaGemini{erro: erros.Com(erros.IAPermissaoNegada, causa)}
	case e.Code == 404 || e.Status == "NOT_FOUND":
		return falhaGemini{erro: erros.Com(erros.IAModeloIndisponivel, causa)}
	case (e.Code == 429 || e.Status == "RESOURCE_EXHAUSTED") && cotaZero(e):
		// "limit: 0": o modelo não faz parte da cota deste nível de conta
		// (ex.: nível gratuito). Esperar não resolve.
		return falhaGemini{erro: erros.Com(erros.IAModeloIndisponivel, causa)}
	case e.Code == 429 || e.Status == "RESOURCE_EXHAUSTED":
		espera, temEspera := retryInfo(e)
		if cotaDiaria(e) || (temEspera && espera > esperaMaxRetryInfo) {
			return falhaGemini{erro: erros.Com(erros.IACota, causa)}
		}
		return falhaGemini{erro: erros.Com(erros.IACota, causa), repetir: true, aguardar: espera}
	case e.Code == 500 || e.Code == 502 || e.Code == 503 || e.Code == 504 ||
		e.Status == "UNAVAILABLE" || e.Status == "INTERNAL" || e.Status == "DEADLINE_EXCEEDED":
		return falhaGemini{erro: erros.Com(erros.IAIndisponivel, causa), repetir: true}
	case e.Code == 400:
		// Pedido rejeitado (schema, argumento): defeito do aplicativo, não do usuário.
		return falhaGemini{erro: erros.Com(erros.InternoInesperado, causa)}
	}
	return falhaGemini{erro: erros.Com(erros.IAIndisponivel, causa)}
}

var reCotaZero = regexp.MustCompile(`(?i)\blimit:\s*0\b`)

// cotaZero: a mensagem de cota informa limite zero para o modelo.
func cotaZero(e genai.APIError) bool { return reCotaZero.MatchString(e.Message) }

// semCreditos reconhece a recusa por falta de créditos ou de faturamento.
func semCreditos(e genai.APIError, motivo string) bool {
	m := strings.ToLower(e.Message)
	for _, t := range []string{"prepayment", "credits are depleted", "billing", "insufficient credit", "out of credits"} {
		if strings.Contains(m, t) {
			return true
		}
	}
	return strings.Contains(motivo, "BILLING")
}

// razao lê google.rpc.ErrorInfo.reason dos detalhes.
func razao(e genai.APIError) string {
	for _, d := range e.Details {
		if t, _ := d["@type"].(string); strings.HasSuffix(t, "google.rpc.ErrorInfo") {
			if r, _ := d["reason"].(string); r != "" {
				return r
			}
		}
	}
	return ""
}

// retryInfo lê google.rpc.RetryInfo.retryDelay ("17s", "0.5s").
func retryInfo(e genai.APIError) (time.Duration, bool) {
	for _, d := range e.Details {
		if t, _ := d["@type"].(string); strings.HasSuffix(t, "google.rpc.RetryInfo") {
			if s, _ := d["retryDelay"].(string); s != "" {
				if f, err := strconv.ParseFloat(strings.TrimSuffix(s, "s"), 64); err == nil && f >= 0 {
					return time.Duration(f * float64(time.Second)), true
				}
			}
		}
	}
	return 0, false
}

// cotaDiaria: google.rpc.QuotaFailure com quotaId/quotaMetric "PerDay".
func cotaDiaria(e genai.APIError) bool {
	for _, d := range e.Details {
		t, _ := d["@type"].(string)
		if !strings.HasSuffix(t, "google.rpc.QuotaFailure") {
			continue
		}
		vs, _ := d["violations"].([]any)
		for _, v := range vs {
			m, _ := v.(map[string]any)
			for _, k := range []string{"quotaId", "quotaMetric"} {
				if s, _ := m[k].(string); strings.Contains(strings.ToLower(s), "perday") {
					return true
				}
			}
		}
	}
	return false
}

// sanear remove a chave de qualquer texto de erro (a causa fica só em
// memória, mas nunca deve conter segredo; V4 §47.4).
func sanear(err error, chave string) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	if chave != "" {
		s = strings.ReplaceAll(s, chave, "***")
	}
	return errors.New(s)
}
