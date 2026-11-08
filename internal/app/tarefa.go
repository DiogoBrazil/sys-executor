// Package app expõe ao frontend (bindings Wails) os serviços do aplicativo.
//
// Regra desta camada: métodos exportados **nunca devolvem error**. O Wails
// envia err.Error() ao JavaScript, o que exporia a causa técnica; em vez
// disso, cada método devolve uma Resposta com o erro do catálogo (título,
// mensagem e ação), sem causa (V4 §33).
package app

import (
	"errors"
	"os"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// EventoVisao é o evento emitido a cada mudança da tarefa.
const EventoVisao = "tarefa:visao"

// Resposta é o retorno de todo comando da interface.
type Resposta struct {
	Visao *workflow.Visao     `json:"visao"`
	Erro  *workflow.ErroVisao `json:"erro"`
}

// SeletorPDF abre o diálogo nativo e devolve o caminho escolhido ("" se cancelado).
type SeletorPDF func() (string, error)

// TarefaService conduz a tarefa a partir da interface.
type TarefaService struct {
	g          *workflow.Gerenciador
	selecionar SeletorPDF
	lerArquivo func(string) ([]byte, error)
}

// NovaTarefaService cria o serviço.
func NovaTarefaService(g *workflow.Gerenciador, sel SeletorPDF) *TarefaService {
	return &TarefaService{g: g, selecionar: sel, lerArquivo: os.ReadFile}
}

func (s *TarefaService) responder(v *workflow.Visao, err error) Resposta {
	if err != nil {
		e := erros.De(err)
		return Resposta{Visao: s.g.Visao(), Erro: &workflow.ErroVisao{
			Codigo: string(e.Codigo), Titulo: e.Titulo, Mensagem: e.Mensagem, Acao: e.Acao,
		}}
	}
	return Resposta{Visao: v}
}

// Atual devolve a tarefa atual (nula se não houver).
func (s *TarefaService) Atual() Resposta { return Resposta{Visao: s.g.Visao()} }

// Iniciar começa uma nova tarefa.
func (s *TarefaService) Iniciar() Resposta { return s.responder(s.g.Iniciar()) }

// EscolherTipo: "PADS" ou "SR".
func (s *TarefaService) EscolherTipo(tipo string) Resposta {
	return s.responder(s.g.EscolherTipo(workflow.Tipo(tipo)))
}

// EscolherEntrada: "SEI" ou "UPLOAD".
func (s *TarefaService) EscolherEntrada(entrada string) Resposta {
	return s.responder(s.g.EscolherEntrada(workflow.Entrada(entrada)))
}

// VoltarParaEntrada retorna à escolha da entrada mantendo o tipo.
func (s *TarefaService) VoltarParaEntrada() Resposta { return s.responder(s.g.VoltarParaEntrada()) }

// SelecionarPDF abre o diálogo nativo e envia o arquivo escolhido. Não pede
// credenciais nem acessa o SEI (T02). O nome do arquivo não é guardado.
func (s *TarefaService) SelecionarPDF() Resposta {
	if v := s.g.Visao(); v == nil || v.Estado != workflow.RecebendoUpload {
		return s.responder(nil, erros.Novo(erros.TarefaOrdemInvalida))
	}
	caminho, err := s.selecionar()
	if err != nil {
		return s.responder(nil, erros.Com(erros.ArquivoLeitura, err))
	}
	if caminho == "" { // usuário fechou o diálogo
		return Resposta{Visao: s.g.Visao()}
	}
	dados, err := s.lerArquivo(caminho)
	if err != nil {
		return s.responder(nil, erros.Com(erros.ArquivoLeitura, errors.Unwrap(err)))
	}
	return s.responder(s.g.ReceberUpload(dados))
}

// Responder escolhe uma opção da pergunta pendente.
func (s *TarefaService) Responder(perguntaID, opcaoID string) Resposta {
	return s.responder(s.g.Responder(perguntaID, opcaoID))
}

// Cancelar interrompe a tarefa e descarta os temporários.
func (s *TarefaService) Cancelar() Resposta { return s.responder(s.g.Cancelar()) }

// Descartar fecha a tarefa terminal e volta à tela inicial.
func (s *TarefaService) Descartar() Resposta { return s.responder(nil, s.g.Descartar()) }
