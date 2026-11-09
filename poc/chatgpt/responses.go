package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

var (
	urlModelos   = "https://api.openai.com/v1/models"
	urlRespostas = "https://api.openai.com/v1/responses"
	// clienteAPI é trocado só nos testes (servidor falso).
	clienteAPI = func(timeout time.Duration) *http.Client { return novoCliente(timeout, hostAPI) }
)

type Modelo struct {
	Slug          string          `json:"slug"`
	DisplayName   string          `json:"display_name"`
	Visibility    string          `json:"visibility"`
	ContextWindow json.RawMessage `json:"context_window"`
	MaxContext    json.RawMessage `json:"max_context_window"`
	Raciocinio    json.RawMessage `json:"supported_reasoning_levels"`
	RacPadrao     json.RawMessage `json:"default_reasoning_level"`
	NaAPI         json.RawMessage `json:"supported_in_api"`
	Entradas      json.RawMessage `json:"input_modalities"`
	Planos        json.RawMessage `json:"available_in_plans"`
}

// ErroAPI preserva status, código, parâmetro e request id (doc "Errors and recovery").
type ErroAPI struct {
	Status    int
	Codigo    string
	Param     string
	Tipo      string
	Mensagem  string
	RequestID string
	NoFluxo   bool
}

func (e *ErroAPI) Error() string {
	onde := "antes do fluxo"
	if e.NoFluxo {
		onde = "no fluxo"
	}
	return fmt.Sprintf("[%s] HTTP %d code=%q param=%q type=%q msg=%q req=%s", onde, e.Status, e.Codigo, e.Param, e.Tipo, cortar(e.Mensagem, 300), e.RequestID)
}

func lerErroAPI(resp *http.Response) *ErroAPI {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	e := &ErroAPI{Status: resp.StatusCode, RequestID: idRequisicao(resp.Header)}
	var v struct {
		Error *struct {
			Code    any    `json:"code"`
			Param   any    `json:"param"`
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		Detail any `json:"detail"`
	}
	if json.Unmarshal(b, &v) == nil && v.Error != nil {
		e.Codigo, e.Param, e.Tipo, e.Mensagem = texto(v.Error.Code), texto(v.Error.Param), v.Error.Type, v.Error.Message
	} else if v.Detail != nil {
		e.Mensagem = "detail: " + fmt.Sprint(v.Detail)
	} else {
		e.Mensagem = string(b)
	}
	return e
}

func texto(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func listarModelos(ctx context.Context, token string) ([]Modelo, []string, error) {
	cli := clienteAPI(30 * time.Second)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, urlModelos, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, lerErroAPI(resp)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, nil, err
	}
	var bruto map[string]json.RawMessage
	if err := json.Unmarshal(b, &bruto); err != nil {
		return nil, nil, fmt.Errorf("lista de modelos ilegível: %w", err)
	}
	var campos []string
	var ms []Modelo
	if raw, ok := bruto["models"]; ok {
		var itens []map[string]any
		_ = json.Unmarshal(raw, &itens)
		if len(itens) > 0 {
			campos = chaves(itens[0])
		}
		_ = json.Unmarshal(raw, &ms)
	} else if raw, ok := bruto["data"]; ok { // formato da API convencional
		var itens []map[string]any
		_ = json.Unmarshal(raw, &itens)
		if len(itens) > 0 {
			campos = append([]string{"(data[])"}, chaves(itens[0])...)
		}
		for _, it := range itens {
			id, _ := it["id"].(string)
			ms = append(ms, Modelo{Slug: id, DisplayName: id, Visibility: "?"})
		}
	}
	return ms, campos, nil
}

// Pedido mínimo da Responses API no modo SIWC. Campos proibidos nunca existem
// aqui (max_output_tokens, temperature, top_p, metadata…), exceto quando o
// teste os injeta de propósito via Extra.
type Pedido struct {
	Modelo     string
	Instrucoes string
	Entrada    []map[string]any
	Formato    map[string]any // text.format
	Raciocinio map[string]any // reasoning
	Extra      map[string]any // campos para testar recusa
}

type Resultado struct {
	Texto        string
	TextoFinal   string // montado a partir de response.completed (conferência)
	Status       string
	Incompleto   string
	Eventos      map[string]int
	Uso          map[string]any
	RequestID    string
	PrimeiroByte time.Duration
	ContentType  string
	Cabecalhos   []string
	Duracao      time.Duration
}

func corpo(p Pedido) map[string]any {
	b := map[string]any{"model": p.Modelo, "input": p.Entrada, "store": false, "stream": true}
	if p.Instrucoes != "" {
		b["instructions"] = p.Instrucoes
	}
	if p.Formato != nil {
		b["text"] = map[string]any{"format": p.Formato}
	}
	if p.Raciocinio != nil {
		b["reasoning"] = p.Raciocinio
	}
	for k, v := range p.Extra {
		b[k] = v
	}
	return b
}

var errSemConclusao = errors.New("fluxo encerrado sem response.completed")

// gerar envia o pedido e consome o SSE. Só há sucesso com response.completed.
// inatividade: prazo máximo sem nenhum byte do fluxo.
func gerar(ctx context.Context, token string, p Pedido, inatividade time.Duration) (*Resultado, error) {
	body, err := json.Marshal(corpo(p))
	if err != nil {
		return nil, err
	}
	ctx, cancelar := context.WithCancel(ctx)
	defer cancelar()
	cli := clienteAPI(0)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, urlRespostas, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	ini := time.Now()

	// vigia de inatividade
	vivo := make(chan struct{}, 1)
	go func() {
		t := time.NewTimer(inatividade)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-vivo:
				if !t.Stop() {
					<-t.C
				}
				t.Reset(inatividade)
			case <-t.C:
				cancelar()
				return
			}
		}
	}()

	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	r := &Resultado{Eventos: map[string]int{}, RequestID: idRequisicao(resp.Header)}
	if resp.StatusCode != http.StatusOK {
		return r, lerErroAPI(resp)
	}
	r.ContentType = resp.Header.Get("Content-Type")
	r.Cabecalhos = nomesCabecalhos(resp.Header)

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	var dados strings.Builder
	var txt strings.Builder
	concluido := false
	var erroFluxo error
	tratar := func(d string) {
		if d == "" || d == "[DONE]" {
			return
		}
		var ev map[string]any
		if json.Unmarshal([]byte(d), &ev) != nil {
			r.Eventos["(ilegível)"]++
			return
		}
		tipo, _ := ev["type"].(string)
		r.Eventos[tipo]++
		switch tipo {
		case "response.output_text.delta":
			if s, ok := ev["delta"].(string); ok {
				txt.WriteString(s)
			}
		case "response.completed":
			concluido = true
			lerResposta(ev, r)
		case "response.incomplete":
			lerResposta(ev, r)
			erroFluxo = fmt.Errorf("resposta incompleta: %s", r.Incompleto)
		case "response.failed":
			lerResposta(ev, r)
			erroFluxo = erroDoEvento(ev, "response", r.RequestID)
		case "error":
			erroFluxo = erroDoEvento(ev, "", r.RequestID)
		}
	}
	for sc.Scan() {
		select {
		case vivo <- struct{}{}:
		default:
		}
		if r.PrimeiroByte == 0 {
			r.PrimeiroByte = time.Since(ini)
		}
		linha := sc.Text()
		switch {
		case linha == "":
			tratar(dados.String())
			dados.Reset()
		case strings.HasPrefix(linha, "data:"):
			if dados.Len() > 0 {
				dados.WriteByte('\n')
			}
			dados.WriteString(strings.TrimPrefix(strings.TrimPrefix(linha, "data:"), " "))
		}
	}
	tratar(dados.String())
	r.Duracao = time.Since(ini)
	r.Texto = txt.String()
	if erroFluxo != nil {
		return r, erroFluxo
	}
	if err := sc.Err(); err != nil {
		if ctx.Err() != nil {
			return r, fmt.Errorf("fluxo interrompido (inatividade > %s ou cancelamento): %w", inatividade, err)
		}
		return r, fmt.Errorf("fluxo interrompido: %w", err)
	}
	if !concluido {
		return r, errSemConclusao
	}
	return r, nil
}

func lerResposta(ev map[string]any, r *Resultado) {
	resp, _ := ev["response"].(map[string]any)
	if resp == nil {
		return
	}
	r.Status, _ = resp["status"].(string)
	if u, ok := resp["usage"].(map[string]any); ok {
		r.Uso = u
	}
	if d, ok := resp["incomplete_details"].(map[string]any); ok {
		r.Incompleto = fmt.Sprint(d["reason"])
	}
	var sb strings.Builder
	if out, ok := resp["output"].([]any); ok {
		for _, it := range out {
			m, _ := it.(map[string]any)
			if m["type"] != "message" {
				continue
			}
			cont, _ := m["content"].([]any)
			for _, c := range cont {
				cm, _ := c.(map[string]any)
				if cm["type"] == "output_text" {
					s, _ := cm["text"].(string)
					sb.WriteString(s)
				}
			}
		}
	}
	r.TextoFinal = sb.String()
}

func erroDoEvento(ev map[string]any, dentro string, req string) error {
	src := ev
	if dentro != "" {
		if m, ok := ev[dentro].(map[string]any); ok {
			src = m
		}
	}
	e := &ErroAPI{NoFluxo: true, RequestID: req}
	if m, ok := src["error"].(map[string]any); ok {
		e.Codigo, e.Param, e.Mensagem = texto(m["code"]), texto(m["param"]), texto(m["message"])
	} else {
		e.Codigo, e.Param, e.Mensagem = texto(src["code"]), texto(src["param"]), texto(src["message"])
	}
	return e
}

func resumoEventos(m map[string]int) string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	var p []string
	for _, k := range ks {
		p = append(p, fmt.Sprintf("%s×%d", k, m[k]))
	}
	return strings.Join(p, " ")
}

func msgUsuario(t string) map[string]any { return map[string]any{"role": "user", "content": t} }

func msgAssistente(t string) map[string]any {
	return map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": t}}}
}

// idRequisicao procura o identificador da requisição em cabeçalhos conhecidos.
func idRequisicao(h http.Header) string {
	for _, k := range []string{"x-request-id", "openai-request-id", "x-oai-request-id", "cf-ray"} {
		if v := h.Get(k); v != "" {
			return k + "=" + v
		}
	}
	return ""
}

func nomesCabecalhos(h http.Header) []string {
	ks := make([]string, 0, len(h))
	for k := range h {
		ks = append(ks, strings.ToLower(k))
	}
	sort.Strings(ks)
	return ks
}
