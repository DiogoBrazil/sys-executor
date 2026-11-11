// Package app expõe ao frontend (bindings Wails) os serviços do aplicativo.
//
// Regra desta camada: métodos exportados **nunca devolvem error**. O Wails
// envia err.Error() ao JavaScript, o que exporia a causa técnica; em vez
// disso, cada método devolve uma Resposta com o erro do catálogo (título,
// mensagem e ação), sem causa (V4 §33).
package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/session"
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
	// SelecionarArquivos: diálogo de vários PDFs do cadastro de peças.
	SelecionarArquivos SeletorArquivos
	// Documento gera o DOCX ou o PDF da minuta em revisão.
	Documento GeradorDocumento
	// SelecionarDestino: diálogo "Salvar como" ("" se cancelado).
	SelecionarDestino SeletorDestino
	gravarArquivo     func(string, []byte) error
}

// GeradorDocumento produz o arquivo da minuta e sugere o nome.
type GeradorDocumento func(ctx context.Context, dados any, pasta, formato string) (conteudo []byte, nome string, err error)

// SeletorDestino abre o diálogo "Salvar como" com o nome sugerido.
type SeletorDestino func(nomeSugerido, formato string) (string, error)

// RespostaArquivo é o retorno de SalvarMinuta.
type RespostaArquivo struct {
	Visao *workflow.Visao     `json:"visao"`
	Erro  *workflow.ErroVisao `json:"erro"`
	// Salvo: o arquivo foi gravado (false também quando o usuário fechou o
	// diálogo). Formato: "docx" ou "pdf".
	Salvo   bool   `json:"salvo"`
	Formato string `json:"formato"`
}

// NovaTarefaService cria o serviço.
func NovaTarefaService(g *workflow.Gerenciador, sel SeletorPDF) *TarefaService {
	return &TarefaService{g: g, selecionar: sel, lerArquivo: os.ReadFile,
		gravarArquivo: func(caminho string, b []byte) error { return os.WriteFile(caminho, b, 0o600) }}
}

func (s *TarefaService) responder(v *workflow.Visao, err error) Resposta {
	if err != nil {
		e := erros.De(err)
		return Resposta{Visao: s.g.Visao(), Erro: workflow.VisaoErro(e)}
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
		// Sem o caminho do arquivo (o nome pode conter o número do processo).
		causa := errors.Unwrap(err)
		if causa == nil {
			causa = err
		}
		return s.responder(nil, erros.Com(erros.ArquivoLeitura, causa))
	}
	return s.responder(s.g.ReceberUpload(dados))
}

// Responder escolhe uma opção da pergunta pendente, com os valores dos
// campos do formulário (validados no Go).
func (s *TarefaService) Responder(perguntaID, opcaoID string, valores map[string]string) Resposta {
	return s.responder(s.g.Responder(perguntaID, opcaoID, valores))
}

// Revisar reabre uma resposta já dada; o que depende dela é refeito.
func (s *TarefaService) Revisar(perguntaID string) Resposta {
	return s.responder(s.g.Revisar(perguntaID))
}

// Retomar repete a etapa parada por erro recuperável.
func (s *TarefaService) Retomar() Resposta { return s.responder(s.g.Retomar()) }

// InformarCredenciais entrega usuário, senha, órgão e número do processo ao
// login do SEI. A senha vira []byte e é zerada depois do uso (melhor esforço:
// a cópia em string recebida da interface não pode ser apagada pelo Go).
func (s *TarefaService) InformarCredenciais(pedidoID, usuario, senha, orgao, processo string) Resposta {
	return s.responder(s.g.Fornecer(pedidoID, workflow.RespostaPedido{
		Campos:  map[string]string{"usuario": strings.TrimSpace(usuario), "orgao": orgao, "processo": strings.TrimSpace(processo)},
		Segredo: session.NovoSegredo([]byte(senha)),
	}))
}

// InformarSegredo entrega o código de dois fatores ou a senha de
// reautenticação pedidos pelo SEI.
func (s *TarefaService) InformarSegredo(pedidoID, valor string) Resposta {
	return s.responder(s.g.Fornecer(pedidoID, workflow.RespostaPedido{Segredo: session.NovoSegredo([]byte(strings.TrimSpace(valor)))}))
}

// SolicitarAjuste envia o pedido de ajuste da minuta em revisão (V4 §31.3).
func (s *TarefaService) SolicitarAjuste(texto string) Resposta {
	return s.responder(s.g.SolicitarAjuste(texto))
}

// Aprovar finaliza a tarefa: descarta os autos, a análise e a minuta. Os
// arquivos já salvos pelo usuário permanecem (V4 §29.1).
func (s *TarefaService) Aprovar() Resposta { return s.responder(s.g.Aprovar()) }

// SalvarMinuta gera o DOCX ou o PDF da minuta em revisão e grava no local
// que o usuário escolher. O caminho escolhido não é guardado.
func (s *TarefaService) SalvarMinuta(formato string) RespostaArquivo {
	falha := func(err error) RespostaArquivo {
		return RespostaArquivo{Visao: s.g.Visao(), Erro: workflow.VisaoErro(erros.De(err)), Formato: formato}
	}
	dados, pasta, ok := s.g.EmRevisao()
	if !ok || s.Documento == nil || s.SelecionarDestino == nil {
		return falha(erros.Novo(erros.TarefaOrdemInvalida))
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()
	conteudo, nome, err := s.Documento(ctx, dados, pasta, formato)
	if err != nil {
		return falha(err)
	}
	caminho, err := s.SelecionarDestino(nome, formato)
	if err != nil {
		return falha(erros.Com(erros.ArquivoEscrita, err))
	}
	if caminho == "" { // usuário fechou o diálogo
		return RespostaArquivo{Visao: s.g.Visao(), Formato: formato}
	}
	if !strings.HasSuffix(strings.ToLower(caminho), "."+formato) {
		caminho += "." + formato
	}
	if err := s.gravarArquivo(caminho, conteudo); err != nil {
		// Sem o caminho na causa (pode conter nome de pessoa ou de processo).
		causa := errors.Unwrap(err)
		if causa == nil {
			causa = err
		}
		return falha(erros.Com(erros.ArquivoEscrita, causa))
	}
	return RespostaArquivo{Visao: s.g.Visao(), Salvo: true, Formato: formato}
}

// Cancelar interrompe a tarefa e descarta os temporários.
func (s *TarefaService) Cancelar() Resposta { return s.responder(s.g.Cancelar()) }

// Descartar fecha a tarefa terminal e volta à tela inicial.
func (s *TarefaService) Descartar() Resposta { return s.responder(nil, s.g.Descartar()) }
