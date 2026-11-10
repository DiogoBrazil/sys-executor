package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
)

// FonteProvedor é a fonte de um provedor de IA (Gemini API, ChatGPT).
type FonteProvedor interface {
	Alvo() (inference.Alvo, error)
	Abrir(ctx context.Context, alvo inference.Alvo, atividade func(string), inspecionar func([]byte) error) (inference.Motor, func(), error)
}

// FonteIA implementa pipeline.FonteMotor escolhendo o provedor pelas
// preferências (V4 §48.1). A escolha é sempre explícita do usuário: não há
// troca automática entre provedores — provedor indisponível é erro, nunca
// desvio para o outro.
type FonteIA struct {
	Loja    *prefs.Loja
	Gemini  FonteProvedor
	ChatGPT FonteProvedor // nil até a conexão de conta existir no aplicativo
}

func (f *FonteIA) provedor() string {
	p := f.Loja.Obter()
	switch {
	case p.ProvedorIA != "":
		return p.ProvedorIA
	case p.ModeloGemini != "":
		// Preferências antigas sem provedor, com modelo Gemini escolhido.
		return prefs.ProvedorGemini
	}
	return ""
}

func (f *FonteIA) fonte(provedor string) (FonteProvedor, error) {
	switch provedor {
	case prefs.ProvedorGemini:
		if f.Gemini != nil {
			return f.Gemini, nil
		}
	case prefs.ProvedorChatGPT:
		if f.ChatGPT != nil {
			return f.ChatGPT, nil
		}
		return nil, erros.Novo(erros.IAContaDesconectada)
	case "":
		return nil, erros.Com(erros.IAModeloIndisponivel, errors.New("nenhum serviço de IA escolhido"))
	}
	return nil, erros.Com(erros.IAModeloIndisponivel, fmt.Errorf("provedor %q indisponível", provedor))
}

// Alvo devolve o provedor e o modelo escolhidos, conferidos sem rede.
func (f *FonteIA) Alvo() (inference.Alvo, error) {
	fp, err := f.fonte(f.provedor())
	if err != nil {
		return inference.Alvo{}, err
	}
	return fp.Alvo()
}

// Abrir abre o motor do provedor do alvo autorizado (não o das preferências
// atuais: se elas mudaram, o pipeline já pediu nova autorização).
func (f *FonteIA) Abrir(ctx context.Context, alvo inference.Alvo, atividade func(string), inspecionar func([]byte) error) (inference.Motor, func(), error) {
	fp, err := f.fonte(alvo.Provedor)
	if err != nil {
		return nil, nil, err
	}
	return fp.Abrir(ctx, alvo, atividade, inspecionar)
}
