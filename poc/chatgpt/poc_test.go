package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func servidorFalso(t *testing.T, h http.HandlerFunc) (corpo *map[string]any) {
	t.Helper()
	var visto map[string]any
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &visto)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	velhoURL, velhoCli := urlRespostas, clienteAPI
	urlRespostas = srv.URL + "/v1/responses"
	clienteAPI = func(time.Duration) *http.Client { return srv.Client() }
	t.Cleanup(func() { urlRespostas, clienteAPI = velhoURL, velhoCli })
	return &visto
}

func sse(w http.ResponseWriter, eventos ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, e := range eventos {
		var m map[string]any
		_ = json.Unmarshal([]byte(e), &m)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", m["type"], e)
		w.(http.Flusher).Flush()
	}
}

func TestConcluido(t *testing.T) {
	visto := servidorFalso(t, func(w http.ResponseWriter, r *http.Request) {
		sse(w, `{"type":"response.created"}`, `{"type":"response.output_text.delta","delta":"o"}`, `{"type":"response.output_text.delta","delta":"k"}`,
			`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":3,"output_tokens":1},"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}}`)
	})
	r, err := gerar(context.Background(), "x", Pedido{Modelo: "m", Instrucoes: "i", Entrada: []map[string]any{msgUsuario("u")}}, 5*time.Second)
	if err != nil || r.Texto != "ok" || r.TextoFinal != "ok" || r.Status != "completed" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	b := *visto
	if b["store"] != false || b["stream"] != true || b["instructions"] != "i" {
		t.Fatalf("corpo: %v", b)
	}
	for _, proibido := range []string{"max_output_tokens", "temperature", "top_p", "metadata", "truncation", "user", "previous_response_id"} {
		if _, ok := b[proibido]; ok {
			t.Fatalf("campo proibido enviado: %s", proibido)
		}
	}
}

func TestSemConclusaoEFalhas(t *testing.T) {
	casos := map[string]struct {
		eventos []string
		quer    string
	}{
		"sem completed": {[]string{`{"type":"response.output_text.delta","delta":"parcial"}`}, "sem response.completed"},
		"incompleto":    {[]string{`{"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`}, "incompleta: max_output_tokens"},
		"failed":        {[]string{`{"type":"response.failed","response":{"error":{"code":"subscription_sharing_usage_limit_exceeded","message":"x"}}}`}, "usage_limit_exceeded"},
		"error":         {[]string{`{"type":"error","code":"server_error","message":"y"}`}, "server_error"},
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			servidorFalso(t, func(w http.ResponseWriter, r *http.Request) { sse(w, c.eventos...) })
			_, err := gerar(context.Background(), "x", Pedido{Modelo: "m"}, 5*time.Second)
			if err == nil || !strings.Contains(err.Error(), c.quer) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestErroAntesDoFluxo(t *testing.T) {
	servidorFalso(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-request-id", "req_1")
		w.WriteHeader(400)
		fmt.Fprint(w, `{"error":{"code":"subscription_sharing_unsupported_capability","param":"max_output_tokens","message":"no"}}`)
	})
	_, err := gerar(context.Background(), "x", Pedido{Modelo: "m"}, 5*time.Second)
	var ea *ErroAPI
	if !errors.As(err, &ea) || ea.Param != "max_output_tokens" || ea.RequestID != "x-request-id=req_1" || ea.NoFluxo {
		t.Fatalf("err=%v", err)
	}
}

func TestInatividade(t *testing.T) {
	servidorFalso(t, func(w http.ResponseWriter, r *http.Request) {
		sse(w, `{"type":"response.created"}`)
		<-r.Context().Done()
	})
	_, err := gerar(context.Background(), "x", Pedido{Modelo: "m"}, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "interrompido") {
		t.Fatalf("err=%v", err)
	}
}

func TestTransporteRestrito(t *testing.T) {
	cli := novoCliente(time.Second, hostAPI)
	_, err := cli.Get("https://exemplo.invalid/")
	if !errors.Is(err, errHostBloqueado) {
		t.Fatalf("err=%v", err)
	}
	if _, err := cli.Get("http://api.openai.com/v1/models"); !errors.Is(err, errHostBloqueado) {
		t.Fatalf("http deveria ser bloqueado: %v", err)
	}
}

func TestValidador(t *testing.T) {
	var sch map[string]any
	_ = json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["a","l"],"properties":{"a":{"type":"string","maxLength":3,"enum":["x","yy"]},"l":{"type":"array","maxItems":1,"items":{"type":"integer","minimum":1,"maximum":2}}}}`), &sch)
	var bom, ruim any
	_ = json.Unmarshal([]byte(`{"a":"yy","l":[2]}`), &bom)
	_ = json.Unmarshal([]byte(`{"a":"zzzz","l":[0,3],"x":1}`), &ruim)
	if v := validar(sch, bom, "$"); len(v) != 0 {
		t.Fatal(v)
	}
	if v := validar(sch, ruim, "$"); len(v) < 5 {
		t.Fatalf("esperava ≥5 violações: %v", v)
	}
	l := limpar(sch, []string{"maxLength"}).(map[string]any)
	if _, ok := l["properties"].(map[string]any)["a"].(map[string]any)["maxLength"]; ok {
		t.Fatal("maxLength não removido")
	}
}

func TestPKCE(t *testing.T) {
	// RFC 7636, apêndice B
	if got := desafioPKCE("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); got != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Fatal(got)
	}
}
