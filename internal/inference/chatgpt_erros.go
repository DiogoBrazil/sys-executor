package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// falhaChatGPT carrega o erro do catálogo e o que fazer com ele.
type falhaChatGPT struct {
	erro    error
	repetir bool // falha transitória: nova tentativa com espera crescente
	renovar bool // 401: renovar o token uma vez e repetir
}

func (f falhaChatGPT) Error() string { return f.erro.Error() }
func (f falhaChatGPT) Unwrap() error { return f.erro }

// classificarHTTP: erro antes do fluxo. O corpo pode ser {"error":{…}} ou
// {"detail": …} (sem código; visto na PoC de 09/11/2026). O texto do
// provedor vai só para a causa técnica (diagnóstico), nunca para o usuário.
func classificarHTTP(status int, corpo []byte, reqID string) error {
	var v struct {
		Error *struct {
			Code    any    `json:"code"`
			Param   any    `json:"param"`
			Message string `json:"message"`
		} `json:"error"`
		Detail any `json:"detail"`
	}
	_ = json.Unmarshal(corpo, &v)
	codigo, param, msg := "", "", ""
	if v.Error != nil {
		if v.Error.Code != nil {
			codigo = fmt.Sprint(v.Error.Code)
		}
		if v.Error.Param != nil {
			param = fmt.Sprint(v.Error.Param)
		}
		msg = v.Error.Message
	} else if v.Detail != nil {
		msg = "detail: " + fmt.Sprint(v.Detail)
	}
	if codigo != "" {
		if f, ok := porCodigo(codigo, msg, param, reqID, status); ok {
			return f
		}
	}
	causa := causaChatGPT(status, codigo, param, msg, reqID)
	switch {
	case status == 401:
		return falhaChatGPT{erro: erros.Com(erros.IASessaoExpirada, causa), renovar: true}
	case status == 403:
		return falhaChatGPT{erro: erros.Com(erros.IAPermissaoNegada, causa)}
	case status == 404:
		return falhaChatGPT{erro: erros.Com(erros.IAModeloIndisponivel, causa)}
	case status == 429:
		return falhaChatGPT{erro: erros.Com(erros.IACota, causa), repetir: true}
	case status >= 500:
		return falhaChatGPT{erro: erros.Com(erros.IAIndisponivel, causa), repetir: true}
	case status == 400:
		// Parâmetro recusado: defeito do aplicativo (não do usuário).
		return falhaChatGPT{erro: erros.Com(erros.InternoInesperado, causa)}
	}
	return falhaChatGPT{erro: erros.Com(erros.IAIndisponivel, causa)}
}

// classificarCodigo: erro durante o fluxo (response.failed, error).
func classificarCodigo(codigo, msg, param, reqID string, _ bool) error {
	if f, ok := porCodigo(codigo, msg, param, reqID, 0); ok {
		return f
	}
	return falhaChatGPT{erro: erros.Com(erros.IAIndisponivel, causaChatGPT(0, codigo, param, msg, reqID))}
}

// porCodigo segue a documentação "Errors and recovery" do Sign in with
// ChatGPT: limite do plano não se repete nem se contorna; conta não
// elegível não entra em laço de OAuth.
func porCodigo(codigo, msg, param, reqID string, status int) (falhaChatGPT, bool) {
	causa := causaChatGPT(status, codigo, param, msg, reqID)
	switch codigo {
	case "subscription_sharing_usage_limit_exceeded":
		return falhaChatGPT{erro: erros.Com(erros.IALimitePlano, causa)}, true
	case "subscription_sharing_user_not_eligible":
		return falhaChatGPT{erro: erros.Com(erros.IAContaNaoElegivel, causa)}, true
	case "subscription_sharing_usage_unavailable", "subscription_sharing_user_unavailable", "server_error":
		return falhaChatGPT{erro: erros.Com(erros.IAIndisponivel, causa), repetir: true}, true
	case "subscription_sharing_unsupported_capability", "subscription_sharing_route_not_supported", "invalid_request_error":
		return falhaChatGPT{erro: erros.Com(erros.InternoInesperado, causa)}, true
	case "subscription_sharing_invalid_user", "token_revoked", "invalid_token", "token_expired":
		return falhaChatGPT{erro: erros.Com(erros.IASessaoExpirada, causa), renovar: true}, true
	case "chatpass_v2_scope_not_authorized", "chatpass_v2_invalid_authorization_context":
		return falhaChatGPT{erro: erros.Com(erros.IAPlanoNaoAutorizado, causa)}, true
	case "model_not_found":
		return falhaChatGPT{erro: erros.Com(erros.IAModeloIndisponivel, causa)}, true
	case "rate_limit_exceeded":
		return falhaChatGPT{erro: erros.Com(erros.IACota, causa), repetir: true}, true
	}
	return falhaChatGPT{}, false
}

func causaChatGPT(status int, codigo, param, msg, reqID string) error {
	r := []rune(msg)
	if len(r) > 300 {
		msg = string(r[:300]) + "…"
	}
	return fmt.Errorf("chatgpt %d code=%q param=%q req=%s: %s", status, codigo, param, reqID, msg)
}

// classificarRede: falha de transporte, cancelamento ou inatividade.
func classificarRede(ctx context.Context, err error, porInatividade <-chan struct{}) error {
	select {
	case <-porInatividade:
		// Tempo esgotado não se repete (o plano tem limite de uso).
		return falhaChatGPT{erro: erros.Com(erros.IATempoEsgotado, errors.New("chatgpt: fluxo sem atividade"))}
	default:
	}
	var insp ErrInspecao
	if errors.As(err, &insp) {
		return falhaChatGPT{erro: erros.Com(erros.IABloqueioPrivacidade, insp)}
	}
	var bloq ErrHostBloqueado
	if errors.As(err, &bloq) {
		return falhaChatGPT{erro: erros.Com(erros.InternoInesperado, bloq)}
	}
	if ctx.Err() != nil {
		return context.Canceled
	}
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return falhaChatGPT{erro: erros.Com(erros.IATempoEsgotado, err)}
	}
	var uerr *url.Error
	var operr *net.OpError
	var dnserr *net.DNSError
	if errors.As(err, &dnserr) || errors.As(err, &operr) || errors.As(err, &uerr) {
		return falhaChatGPT{erro: erros.Com(erros.IASemConexao, err), repetir: true}
	}
	return falhaChatGPT{erro: erros.Com(erros.IAIndisponivel, fmt.Errorf("%T: %w", err, err))}
}
