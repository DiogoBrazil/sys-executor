package inference

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Integração com o llama-server real (opcional; CPU). Exige
// SYSEXEC_LLAMA_SERVER e SYSEXEC_MODELO_TESTE (um GGUF gerador pequeno).
func TestIntegracaoLlamaServer(t *testing.T) {
	exe, modelo := os.Getenv("SYSEXEC_LLAMA_SERVER"), os.Getenv("SYSEXEC_MODELO_TESTE")
	if exe == "" || modelo == "" {
		t.Skip("SYSEXEC_LLAMA_SERVER/SYSEXEC_MODELO_TESTE não definidos")
	}
	f, err := os.Open(modelo)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := LerGGUF(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := DetectarHardware()
	s, err := IniciarServidor(context.Background(), ConfigServidor{Executavel: exe, Modelo: modelo, Contexto: 4096, Threads: h.NucleosFisicos, TimeoutCarga: 3 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Fechar()
	r, err := http.Get(s.base + "/v1/models")
	if err != nil || r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("SEG-SID-01: sem chave deveria ser 401: %v %v", err, r)
	}
	r.Body.Close()
	schema := json.RawMessage(`{"type":"object","additionalProperties":false,"required":["artigo","natureza"],
	 "properties":{"artigo":{"type":"integer"},"natureza":{"type":"string","enum":["LEVE","MEDIA","GRAVE"]}}}`)
	res, err := s.Gerar(context.Background(), Pedido{
		Sistema: "Responda somente no JSON do schema.",
		Usuario: "Extraia: 'conduta prevista no art. 16, inciso XVI, do RDPM (natureza média)'.",
		Schema:  schema, MaxTokens: 80, Temperatura: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Artigo   int
		Natureza string
	}
	if err := json.Unmarshal(res.Conteudo, &out); err != nil || out.Artigo != 16 || out.Natureza != "MEDIA" {
		t.Fatalf("%s: %+v %v", res.Conteudo, out, err)
	}
	n, err := s.ContarTokens(context.Background(), "Termo de inquirição de testemunha")
	if err != nil || n < 3 {
		t.Fatalf("tokens: %d %v", n, err)
	}
	t.Logf("%s (%s): %.1f tok/s, %d+%d tokens, %s", meta.Nome, meta.Arquitetura, res.TokensPorSeg, res.TokensEntrada, res.TokensSaida, res.Duracao)
	s.Fechar()
	if s.Vivo() || strings.Contains(s.saida.cauda(10), "a") {
		t.Fatal("processo vivo ou saída não limpa")
	}
}
