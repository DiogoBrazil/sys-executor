package app

import (
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// PreferenciasService expõe as preferências persistentes mínimas.
type PreferenciasService struct {
	loja *prefs.Loja
}

// NovaPreferenciasService cria o serviço.
func NovaPreferenciasService(l *prefs.Loja) *PreferenciasService {
	return &PreferenciasService{loja: l}
}

// RespostaPreferencias é o retorno dos comandos de preferências.
type RespostaPreferencias struct {
	Preferencias prefs.Preferencias  `json:"preferencias"`
	Erro         *workflow.ErroVisao `json:"erro"`
}

// Obter devolve as preferências atuais.
func (s *PreferenciasService) Obter() RespostaPreferencias {
	return RespostaPreferencias{Preferencias: s.loja.Obter()}
}

// DefinirPesquisaWeb liga ou desliga a pesquisa em fontes oficiais (V4 §41).
func (s *PreferenciasService) DefinirPesquisaWeb(ligada bool) RespostaPreferencias {
	p, err := s.loja.Atualizar(func(p *prefs.Preferencias) { p.PesquisaWeb = ligada })
	if err != nil {
		e := erros.Com(erros.ArquivoEscrita, err)
		return RespostaPreferencias{Preferencias: p, Erro: &workflow.ErroVisao{
			Codigo: string(e.Codigo), Titulo: e.Titulo, Mensagem: e.Mensagem, Acao: e.Acao,
		}}
	}
	return RespostaPreferencias{Preferencias: p}
}
