// Pacote inference é a porta de inferência do sys-executor (TRV-IA-001,
// emenda V4 §47): o orquestrador só conhece a interface Motor. A
// implementação de produção é o MotorGemini (Gemini API, chave do usuário).
// Não há inferência local nem fallback para modelo local (D-10).
package inference

import (
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"
)

// Pedido é uma chamada de geração estruturada (sem ferramentas; plano §5.6).
type Pedido struct {
	Sistema     string
	Usuario     string
	NomeSchema  string
	Schema      json.RawMessage // JSON Schema da resposta (imposto pelo provedor e conferido no Go)
	MaxTokens   int
	Temperatura float64
	// Historico permite reenviar com a mensagem do validador (§5.7).
	Historico []Mensagem
}

// Mensagem de chat. Papel: "user" ou "assistant".
type Mensagem struct {
	Papel    string `json:"role"`
	Conteudo string `json:"content"`
}

// Resposta da geração.
type Resposta struct {
	Conteudo         []byte
	Truncada         bool // parou por limite de tokens: nunca vale como resposta concluída
	TokensEntrada    int
	TokensSaida      int
	TokensPensamento int
	TokensPorSeg     float64
	Duracao          time.Duration
}

// Motor é a porta de inferência (TRV-IA-001): o orquestrador só conhece isto.
type Motor interface {
	Gerar(ctx context.Context, p Pedido) (Resposta, error)
	ContarTokens(ctx context.Context, texto string) (int, error)
	Contexto() int
}

// EstimarTokens: estimativa local e conservadora (≈ 3 caracteres por token
// em pt-BR, acima do real dos modelos atuais). Usada no orçamento do pedido
// e no fatiamento; nenhuma chamada de rede é feita para contar tokens.
func EstimarTokens(texto string) int {
	return (utf8.RuneCountInString(texto) + 2) / 3
}
