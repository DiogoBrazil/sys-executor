package inference

import (
	"context"
	"sort"
	"strings"

	"google.golang.org/genai"
)

// Estágios de um modelo (V4 §47.5).
const (
	EstagioEstavel      = "estavel"
	EstagioPrevia       = "previa"
	EstagioExperimental = "experimental"
	EstagioApelido      = "apelido" // "-latest": troca de versão sem aviso no app
)

// ModeloGemini é um modelo de texto oferecido ao usuário.
type ModeloGemini struct {
	ID         string `json:"id"` // sem o prefixo "models/"
	Nome       string `json:"nome"`
	Descricao  string `json:"descricao"`
	Estagio    string `json:"estagio"`
	EntradaMax int    `json:"entradaMax"`
	SaidaMax   int    `json:"saidaMax"`
	Pensamento bool   `json:"pensamento"`
}

// variantesExcluidas: modelos que não servem para redigir julgamentos
// (imagem, áudio, voz em tempo real, embeddings, robótica, uso de computador).
var variantesExcluidas = []string{"image", "imagen", "tts", "audio", "live", "embedding",
	"robotics", "computer-use", "veo", "aqa", "omni"}

// ListarModelos devolve os modelos Gemini de texto que a chave enxerga e que
// aceitam generateContent. Também serve para validar a chave sem gerar
// tokens (chave recusada → IA-002).
func ListarModelos(ctx context.Context, c ConfigGemini) ([]ModeloGemini, error) {
	cli, hc, err := novoCliente(ctx, c)
	if err != nil {
		return nil, err
	}
	defer hc.CloseIdleConnections()
	m := &MotorGemini{chave: string(c.Chave), espera: c.espera}
	if m.espera == nil {
		m.espera = esperar
	}
	var lista []*genai.Model
	err = m.repetir(ctx, func(ctx context.Context) error {
		lista = lista[:0]
		for mod, e := range cli.Models.All(ctx) {
			if e != nil {
				return e
			}
			lista = append(lista, mod)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out []ModeloGemini
	for _, mod := range lista {
		if g, ok := filtrarModelo(mod); ok {
			out = append(out, g)
		}
	}
	ordem := map[string]int{EstagioEstavel: 0, EstagioApelido: 1, EstagioPrevia: 2, EstagioExperimental: 3}
	sort.SliceStable(out, func(i, j int) bool {
		if ordem[out[i].Estagio] != ordem[out[j].Estagio] {
			return ordem[out[i].Estagio] < ordem[out[j].Estagio]
		}
		return out[i].ID > out[j].ID // versões mais novas primeiro
	})
	return out, nil
}

func filtrarModelo(mod *genai.Model) (ModeloGemini, bool) {
	if mod == nil {
		return ModeloGemini{}, false
	}
	id := strings.TrimPrefix(mod.Name, "models/")
	if !strings.HasPrefix(id, "gemini-") || !reModelo.MatchString(id) {
		return ModeloGemini{}, false
	}
	gera := false
	for _, a := range mod.SupportedActions {
		if a == "generateContent" {
			gera = true
		}
	}
	if !gera {
		return ModeloGemini{}, false
	}
	for _, v := range variantesExcluidas {
		if strings.Contains(id, v) {
			return ModeloGemini{}, false
		}
	}
	nome := mod.DisplayName
	if nome == "" {
		nome = id
	}
	return ModeloGemini{
		ID: id, Nome: nome, Descricao: mod.Description, Estagio: estagio(id),
		EntradaMax: int(mod.InputTokenLimit), SaidaMax: int(mod.OutputTokenLimit), Pensamento: mod.Thinking,
	}, true
}

func estagio(id string) string {
	switch {
	case strings.HasSuffix(id, "-latest"):
		return EstagioApelido
	case strings.Contains(id, "-exp") || strings.Contains(id, "experimental"):
		return EstagioExperimental
	case strings.Contains(id, "preview"):
		return EstagioPrevia
	}
	return EstagioEstavel
}

// TestarModelo faz uma geração mínima, sem dados dos autos, para saber se a
// chave tem acesso e cota para o modelo. Só roda a pedido do usuário.
func TestarModelo(ctx context.Context, c ConfigGemini) error {
	m, err := NovoMotorGemini(ctx, c)
	if err != nil {
		return err
	}
	defer m.Fechar()
	_, err = m.Gerar(ctx, Pedido{
		Usuario:   `Responda somente com o JSON {"ok": true}.`,
		Schema:    []byte(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}`),
		MaxTokens: 32,
	})
	return err
}
