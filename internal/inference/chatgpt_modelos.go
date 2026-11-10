package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// ModeloChatGPT é um modelo disponível para a conta (GET /v1/models no modo
// Sign in with ChatGPT). Nada é fixo no código (V4 §48.7).
type ModeloChatGPT struct {
	ID               string   `json:"id"` // slug
	Nome             string   `json:"nome"`
	Contexto         int      `json:"contexto"`
	Raciocinio       []string `json:"raciocinio"` // níveis aceitos em reasoning.effort
	RaciocinioPadrao string   `json:"raciocinioPadrao"`
}

type itemModeloChatGPT struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Visibility  string `json:"visibility"`
	NaAPI       *bool  `json:"supported_in_api"`
	Contexto    int    `json:"context_window"`
	Niveis      []struct {
		Effort string `json:"effort"`
	} `json:"supported_reasoning_levels"`
	Padrao string `json:"default_reasoning_level"`
}

// ListarModelosChatGPT lista os modelos que aparecem para escolha
// (visibility "list" e uso pela API), na ordem do servidor. Também confere
// que a conta e a permissão do plano funcionam, sem gerar texto.
func ListarModelosChatGPT(ctx context.Context, cfg ConfigChatGPT) ([]ModeloChatGPT, error) {
	if cfg.Tokens == nil {
		return nil, erros.Novo(erros.IAContaDesconectada)
	}
	origem := cfg.origem
	if origem == "" {
		origem = OrigemAPIOpenAI
	}
	cli, err := NovoClienteRestrito(origem, nil)
	if err != nil {
		return nil, erros.Com(erros.InternoInesperado, err)
	}
	cli.Timeout = 60 * time.Second
	defer cli.CloseIdleConnections()
	renovou := false
	for {
		tok, err := cfg.Tokens.Token(ctx)
		if err != nil {
			return nil, err
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, origem+caminhoModelos, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Accept", "application/json")
		resp, err := cli.Do(req)
		if err != nil {
			e := classificarRede(ctx, err, nil)
			var f falhaChatGPT
			if errors.As(e, &f) {
				return nil, f.erro
			}
			return nil, e
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			e := classificarHTTP(resp.StatusCode, b, idRequisicao(resp.Header))
			var f falhaChatGPT
			if errors.As(e, &f) && f.renovar && !renovou {
				renovou = true
				if _, err := cfg.Tokens.Renovar(ctx); err != nil {
					return nil, err
				}
				continue
			}
			if errors.As(e, &f) {
				return nil, f.erro
			}
			return nil, e
		}
		var corpo struct {
			Models []itemModeloChatGPT `json:"models"`
		}
		if err := json.Unmarshal(b, &corpo); err != nil {
			return nil, erros.Com(erros.IAIndisponivel, fmt.Errorf("lista de modelos ilegível: %w", err))
		}
		var out []ModeloChatGPT
		for _, it := range corpo.Models {
			if it.Visibility != "list" || (it.NaAPI != nil && !*it.NaAPI) || !reSlug.MatchString(it.Slug) {
				continue
			}
			m := ModeloChatGPT{ID: it.Slug, Nome: it.DisplayName, Contexto: it.Contexto, RaciocinioPadrao: it.Padrao}
			if m.Nome == "" {
				m.Nome = it.Slug
			}
			for _, n := range it.Niveis {
				if n.Effort != "" {
					m.Raciocinio = append(m.Raciocinio, n.Effort)
				}
			}
			out = append(out, m)
		}
		return out, nil
	}
}

// TestarModeloChatGPT gera uma resposta mínima, sem dados dos autos.
func TestarModeloChatGPT(ctx context.Context, cfg ConfigChatGPT) error {
	m, err := NovoMotorChatGPT(cfg)
	if err != nil {
		return err
	}
	defer m.Fechar()
	_, err = m.Gerar(ctx, Pedido{
		Sistema: "Teste de conexão do aplicativo. Responda somente com o JSON pedido.", Usuario: `Responda {"ok": true}.`,
		NomeSchema: "teste",
		Schema:     json.RawMessage(`{"type":"object","additionalProperties":false,"required":["ok"],"properties":{"ok":{"type":"boolean"}}}`),
	})
	return err
}
