// Pacote llamafalso imita o llama-server nos testes: o binário de teste se
// reexecuta com SYSEXEC_LLAMA_FALSO=<modo> e responde em loopback com a
// porta e a chave recebidas (/health, /v1/chat/completions, /tokenize).
//
// Modos: ok, morre (falha na carga), morre-no-pedido.
package llamafalso

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Variavel ativa o modo de servidor falso.
const Variavel = "SYSEXEC_LLAMA_FALSO"

// Executar deve ser chamado no TestMain: se a variável estiver definida, o
// processo vira o servidor falso e devolve true (o TestMain deve sair).
func Executar() bool {
	modo := os.Getenv(Variavel)
	if modo == "" {
		return false
	}
	servir(modo)
	return true
}

type mensagem struct {
	Papel    string `json:"role"`
	Conteudo string `json:"content"`
}

func arg(nome string) string {
	for i, a := range os.Args {
		if a == nome && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}

func servir(modo string) {
	if modo == "morre" {
		fmt.Fprintln(os.Stderr, "falha ao carregar o modelo")
		os.Exit(3)
	}
	porta, chave := arg("--port"), arg("--api-key")
	pronto := time.Now().Add(300 * time.Millisecond)
	mux := http.NewServeMux()
	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+chave {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if time.Now().Before(pronto) {
			http.Error(w, "loading", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/v1/chat/completions", auth(func(w http.ResponseWriter, r *http.Request) {
		var c struct {
			Messages []mensagem `json:"messages"`
			Format   struct {
				Type string `json:"type"`
			} `json:"response_format"`
			MaxTokens int `json:"max_tokens"`
		}
		json.NewDecoder(r.Body).Decode(&c)
		if modo == "morre-no-pedido" {
			os.Exit(4)
		}
		ultimo := c.Messages[len(c.Messages)-1].Conteudo
		conteudo := fmt.Sprintf(`{"eco":%q,"formato":%q,"mensagens":%d}`, ultimo, c.Format.Type, len(c.Messages))
		fim := "stop"
		if c.MaxTokens == 1 {
			fim = "length"
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": conteudo}, "finish_reason": fim}},
			"usage":   map[string]any{"prompt_tokens": len(ultimo), "completion_tokens": 7},
			"timings": map[string]any{"predicted_per_second": 12.5},
		})
	}))
	mux.HandleFunc("/tokenize", auth(func(w http.ResponseWriter, r *http.Request) {
		var c struct{ Content string }
		json.NewDecoder(r.Body).Decode(&c)
		toks := make([]int, len(strings.Fields(c.Content)))
		json.NewEncoder(w).Encode(map[string]any{"tokens": toks})
	}))
	srv := &http.Server{Addr: "127.0.0.1:" + porta, Handler: mux}
	if _, err := strconv.Atoi(porta); err != nil {
		os.Exit(5)
	}
	srv.ListenAndServe()
}
