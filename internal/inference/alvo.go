package inference

// Provedores de IA admitidos (V4 §§47, 48.1). A escolha é sempre explícita
// do usuário; não há troca automática entre eles.
const (
	ProvedorGemini  = "GEMINI"
	ProvedorChatGPT = "CHATGPT"
)

// Alvo identifica para onde vai o conteúdo de uma tarefa: o provedor e o
// modelo. A autorização de envio vale para um alvo; se ele mudar, o
// aplicativo pergunta de novo (V4 §48.1, TRV-SEG-010 v2).
type Alvo struct {
	Provedor string // ProvedorGemini ou ProvedorChatGPT
	Rotulo   string // para a interface, ex.: "Gemini API (Google)"
	Modelo   string
}

// RotuloProvedor é o nome do provedor mostrado ao usuário.
func RotuloProvedor(p string) string {
	switch p {
	case ProvedorChatGPT:
		return "ChatGPT (OpenAI), pelo seu plano"
	default:
		return "Gemini API (Google)"
	}
}
