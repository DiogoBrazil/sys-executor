package app

import (
	"errors"
	"os"
	"sort"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/rules/enquadramento"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Cadastro de peças (V4 §48.4). Os arquivos são escolhidos no diálogo
// nativo e lidos pelo Go; a interface vê só o nome, as páginas e o
// resumo do hash. Nada é gravado fora da sessão cifrada.

// SeletorArquivos abre o diálogo nativo de vários PDFs (nil se cancelado).
type SeletorArquivos func() ([]string, error)

// SalvarCadastro grava os metadados editados (pessoas, fatos, peças).
func (s *TarefaService) SalvarCadastro(c workflow.Cadastro) Resposta {
	return s.responder(s.g.SalvarCadastro(c))
}

// AnexarArquivos abre o diálogo e acrescenta os PDFs escolhidos à peça.
func (s *TarefaService) AnexarArquivos(pecaID string) Resposta {
	if v := s.g.Visao(); v == nil || v.Estado != workflow.RecebendoUpload || v.Cadastro == nil {
		return s.responder(nil, erros.Novo(erros.TarefaOrdemInvalida))
	}
	if s.SelecionarArquivos == nil {
		return s.responder(nil, erros.Com(erros.InternoInesperado, errors.New("diálogo de arquivos indisponível")))
	}
	caminhos, err := s.SelecionarArquivos()
	if err != nil {
		return s.responder(nil, erros.Com(erros.ArquivoLeitura, err))
	}
	if len(caminhos) == 0 {
		return Resposta{Visao: s.g.Visao()}
	}
	sort.Strings(caminhos) // o diálogo não garante a ordem; o usuário reordena
	var as []workflow.ArquivoEntrada
	total := 0
	for _, c := range caminhos {
		if st, err := os.Stat(c); err == nil && st.Size() > workflow.TamanhoMaximoPDF {
			return s.responder(nil, erros.Novo(erros.PDFGrande))
		}
		b, err := s.lerArquivo(c)
		if err != nil {
			causa := errors.Unwrap(err) // sem o caminho (pode conter o nº do processo)
			if causa == nil {
				causa = err
			}
			return s.responder(nil, erros.Com(erros.ArquivoLeitura, causa))
		}
		if total += len(b); total > workflow.MaxBytesCadastro {
			return s.responder(nil, erros.Novo(erros.CadastroLimite))
		}
		as = append(as, workflow.ArquivoEntrada{Nome: nomeDoArquivo(c), Dados: b, Origem: "UPLOAD"})
	}
	return s.responder(s.g.AnexarArquivos(pecaID, as))
}

// RemoverArquivo tira o arquivo da peça (e da sessão).
func (s *TarefaService) RemoverArquivo(pecaID, arquivoID string) Resposta {
	return s.responder(s.g.RemoverArquivo(pecaID, arquivoID))
}

// MoverArquivo muda a ordem do arquivo na peça (-1 sobe, +1 desce).
func (s *TarefaService) MoverArquivo(pecaID, arquivoID string, delta int) Resposta {
	return s.responder(s.g.MoverArquivo(pecaID, arquivoID, delta))
}

// ConcluirCadastro confere as pendências e inicia o processamento.
func (s *TarefaService) ConcluirCadastro() Resposta { return s.responder(s.g.ConcluirCadastro()) }

// DispositivoVisao é um item do catálogo de enquadramento para a interface.
type DispositivoVisao struct {
	ID       string `json:"id"`
	Diploma  string `json:"diploma"`
	Rotulo   string `json:"rotulo"`
	Natureza string `json:"natureza"` // fixa no RDPM; vazia no Estatuto
}

// CatalogoEnquadramento devolve os dispositivos do catálogo curado (RDPM
// arts. 15–17; Estatuto arts. 29 e 32), para o assistente.
func (s *TarefaService) CatalogoEnquadramento() []DispositivoVisao {
	ds, err := enquadramento.Todos()
	if err != nil {
		return nil
	}
	out := make([]DispositivoVisao, 0, len(ds))
	for _, d := range ds {
		out = append(out, DispositivoVisao{ID: d.ID, Diploma: d.Diploma, Rotulo: d.Rotulo(), Natureza: d.Natureza})
	}
	return out
}

// nomeDoArquivo: só o nome, sem a pasta (separadores do Windows e do Unix).
func nomeDoArquivo(caminho string) string {
	if i := strings.LastIndexAny(caminho, `\/`); i >= 0 {
		return caminho[i+1:]
	}
	return caminho
}
