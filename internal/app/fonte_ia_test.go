package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
)

type provedorFalso struct {
	nome  string
	abriu []inference.Alvo
}

func (p *provedorFalso) Alvo() (inference.Alvo, error) {
	return inference.Alvo{Provedor: p.nome, Rotulo: p.nome, Modelo: "m-" + p.nome}, nil
}
func (p *provedorFalso) Abrir(_ context.Context, a inference.Alvo, _ func(string), _ func([]byte) error) (inference.Motor, func(), error) {
	p.abriu = append(p.abriu, a)
	return nil, func() {}, nil
}

// V4 §48.1: o provedor é o escolhido pelo usuário; provedor escolhido e
// indisponível é erro, nunca desvio automático para o outro.
// Regra: TRV-IA-014
func TestFonteIAEscolhaExplicitaSemTrocaAutomatica(t *testing.T) {
	loja, _ := prefs.Abrir(filepath.Join(t.TempDir(), "p.json"))
	g := &provedorFalso{nome: inference.ProvedorGemini}
	f := &FonteIA{Loja: loja, Gemini: g}

	if _, err := f.Alvo(); !errors.Is(err, erros.Novo(erros.IAModeloIndisponivel)) {
		t.Fatalf("sem provedor escolhido: %v", err)
	}
	// Preferências antigas: modelo Gemini sem provedor → Gemini.
	_, _ = loja.Atualizar(func(p *prefs.Preferencias) { p.ModeloGemini = "gemini-3.6-flash" })
	if a, err := f.Alvo(); err != nil || a.Provedor != inference.ProvedorGemini {
		t.Fatalf("%+v %v", a, err)
	}
	// ChatGPT escolhido sem conta conectada: IA-015, sem cair no Gemini.
	_, _ = loja.Atualizar(func(p *prefs.Preferencias) { p.ProvedorIA = prefs.ProvedorChatGPT })
	if _, err := f.Alvo(); !errors.Is(err, erros.Novo(erros.IAContaDesconectada)) {
		t.Fatalf("esperado IA-015: %v", err)
	}
	if _, _, err := f.Abrir(context.Background(), inference.Alvo{Provedor: inference.ProvedorChatGPT, Modelo: "x"}, func(string) {}, nil); err == nil || len(g.abriu) != 0 {
		t.Fatalf("não pode abrir o Gemini no lugar do ChatGPT: %v %v", err, g.abriu)
	}
	// Com o ChatGPT disponível, abre o provedor do alvo autorizado.
	c := &provedorFalso{nome: inference.ProvedorChatGPT}
	f.ChatGPT = c
	a, err := f.Alvo()
	if err != nil || a.Provedor != inference.ProvedorChatGPT {
		t.Fatalf("%+v %v", a, err)
	}
	if _, _, err := f.Abrir(context.Background(), inference.Alvo{Provedor: inference.ProvedorGemini, Modelo: "gemini-3.6-flash"}, func(string) {}, nil); err != nil || len(g.abriu) != 1 || len(c.abriu) != 0 {
		t.Fatalf("Abrir segue o alvo autorizado, não as preferências atuais: %v", err)
	}
}
