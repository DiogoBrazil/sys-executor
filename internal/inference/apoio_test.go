package inference

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ggufSintetico monta um cabeçalho GGUF v3 mínimo (sem tensores reais).
func ggufSintetico(arq string, chat bool, pooling uint32, extra int) []byte {
	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, binary.LittleEndian, v) }
	str := func(s string) { w(uint64(len(s))); b.WriteString(s) }
	kvs := 0 // os pares são escritos primeiro; o cabeçalho, depois
	kvStr := func(k, v string) { str(k); w(uint32(8)); str(v); kvs++ }
	kvU32 := func(k string, v uint32) { str(k); w(uint32(4)); w(v); kvs++ }
	kvStr("general.architecture", arq)
	kvStr("general.name", "Sintético "+arq)
	kvStr("general.license", "apache-2.0")
	kvU32("general.file_type", 15)
	kvU32(arq+".block_count", 8)
	kvU32(arq+".context_length", 32768)
	kvU32(arq+".embedding_length", 512)
	kvU32(arq+".attention.head_count", 8)
	kvU32(arq+".attention.head_count_kv", 2)
	if pooling != 0 {
		kvU32(arq+".pooling_type", pooling)
	}
	// Array de strings (como o vocabulário) e array numérico: devem ser pulados.
	str("tokenizer.ggml.tokens")
	w(uint32(9))
	w(uint32(8))
	w(uint64(3))
	str("a")
	str("bb")
	str("ccc")
	kvs++
	str("tokenizer.ggml.scores")
	w(uint32(9))
	w(uint32(6))
	w(uint64(4))
	for i := 0; i < 4; i++ {
		w(float32(i))
	}
	kvs++
	if chat {
		kvStr("tokenizer.chat_template", "{{ messages }}")
	}
	pares := b.Bytes()
	b = bytes.Buffer{}
	w(uint32(magicGGUF))
	w(uint32(3))
	w(uint64(1))
	w(uint64(kvs))
	b.Write(pares)
	b.Write(bytes.Repeat([]byte{0}, extra)) // "pesos"
	return b.Bytes()
}

// Servidor falso: o próprio binário de teste, reexecutado com
// SYSEXEC_LLAMA_FALSO=<modo>, imita o llama-server (porta, chave, /health,
// /v1/chat/completions, /tokenize).
func TestMain(m *testing.M) {
	if modo := os.Getenv("SYSEXEC_LLAMA_FALSO"); modo != "" {
		servidorFalso(modo)
		return
	}
	os.Exit(m.Run())
}

func arg(nome string) string {
	for i, a := range os.Args {
		if a == nome && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}

func servidorFalso(modo string) {
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
			Messages []Mensagem `json:"messages"`
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
