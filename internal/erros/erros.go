// Package erros é o catálogo de erros do sys-executor (V4 §33, plano §13).
//
// Cada falha exibida ao usuário tem um código estável, um título, uma mensagem
// amigável em pt-BR e uma ação sugerida. A causa técnica (Causa) só existe em
// memória, para diagnóstico e testes: nunca é mostrada na interface, nunca é
// gravada em disco e nunca deve conter segredos ou dados do processo.
package erros

import (
	"errors"
	"fmt"
)

// Codigo identifica uma entrada do catálogo (ex.: "PDF-002").
type Codigo string

// Categoria agrupa os códigos por área (V4 §33: SEI, credenciais, rede,
// download, PDF, OCR, modelo, memória, indexação, arquivos, pesquisa,
// geração/validação).
type Categoria string

const (
	CatSEI        Categoria = "SEI"
	CatRede       Categoria = "REDE"
	CatPDF        Categoria = "PDF"
	CatOCR        Categoria = "OCR"
	CatArquivo    Categoria = "ARQUIVO"
	CatModelo     Categoria = "MODELO"
	CatBiblioteca Categoria = "BIBLIOTECA"
	CatReferencia Categoria = "REFERENCIA"
	CatValidacao  Categoria = "VALIDACAO"
	CatExportacao Categoria = "EXPORTACAO"
	CatSessao     Categoria = "SESSAO"
	CatTarefa     Categoria = "TAREFA"
	CatPesquisa   Categoria = "PESQUISA"
	CatInterno    Categoria = "INTERNO"
)

// Definicao é uma entrada imutável do catálogo.
type Definicao struct {
	Codigo    Codigo
	Categoria Categoria
	Titulo    string
	Mensagem  string
	Acao      string
}

// Erro é a falha concreta: uma definição do catálogo e, opcionalmente, a causa técnica.
type Erro struct {
	Definicao
	Causa error // somente memória; nunca exibida
}

func (e *Erro) Error() string {
	if e.Causa != nil {
		return fmt.Sprintf("%s: %s: %v", e.Codigo, e.Titulo, e.Causa)
	}
	return fmt.Sprintf("%s: %s", e.Codigo, e.Titulo)
}

func (e *Erro) Unwrap() error { return e.Causa }

// Is permite errors.Is(err, erros.Novo(cod)) comparar só pelo código.
func (e *Erro) Is(alvo error) bool {
	var o *Erro
	return errors.As(alvo, &o) && o.Codigo == e.Codigo
}

// Novo cria o erro do código informado. Código desconhecido é defeito de
// programação e vira INT-001 (nunca entra em pânico na mão do usuário).
func Novo(c Codigo) *Erro {
	d, ok := catalogo[c]
	if !ok {
		d = catalogo[InternoInesperado]
	}
	return &Erro{Definicao: d}
}

// Com cria o erro do código informado com a causa técnica anexada.
func Com(c Codigo, causa error) *Erro {
	e := Novo(c)
	e.Causa = causa
	return e
}

// De converte qualquer erro em *Erro: preserva um *Erro existente na cadeia ou
// embrulha o resto como INT-001, sem expor a mensagem original.
func De(err error) *Erro {
	if err == nil {
		return nil
	}
	var e *Erro
	if errors.As(err, &e) {
		return e
	}
	return Com(InternoInesperado, err)
}

// Definicoes devolve uma cópia do catálogo, para testes e documentação.
func Definicoes() []Definicao {
	out := make([]Definicao, 0, len(ordem))
	for _, c := range ordem {
		out = append(out, catalogo[c])
	}
	return out
}
