// Pacote pads: invariantes de escopo do julgamento de PADS (V4 §§21–22;
// plano §8 "Invariantes verificados em teste e em runtime").
//
// O Memorando Disciplinar delimita a acusação: as imputações são extraídas,
// confirmadas pelo usuário e congeladas em um ImputacaoSet versionado e
// auditável. O julgamento só aceita resultado para imputação desse conjunto;
// conduta diversa descoberta na instrução não entra (PADS-ESC-002/004/007).
package pads

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
)

var (
	ErrCongelado        = errors.New("conjunto de imputações congelado: reabra com motivo registrado para alterar")
	ErrNaoCongelado     = errors.New("conjunto de imputações ainda não confirmado e congelado")
	ErrVazio            = errors.New("nenhuma imputação delimitada no memorando")
	ErrIDInvalido       = errors.New("imputação sem identificador")
	ErrIDRepetido       = errors.New("identificador de imputação repetido")
	ErrForaDoEscopo     = errors.New("imputação fora do memorando disciplinar: não pode ser julgada no PADS")
	ErrConjuntoAlterado = errors.New("o conjunto de imputações mudou depois do início do julgamento")
	ErrPendente         = errors.New("há imputação sem resultado confirmado pelo usuário")
	ErrMotivo           = errors.New("reabertura exige motivo")
)

// ErrAcusadoUnico: PADS tem exatamente um acusado (PADS-ESC-006). O fluxo
// mostra o alerta e pergunta ao usuário (PADS-1AC-01).
type ErrAcusadoUnico struct{ Quantidade int }

func (e ErrAcusadoUnico) Error() string {
	return fmt.Sprintf("o PADS admite um acusado; os autos indicam %d — confirme quem é o acusado deste processo", e.Quantidade)
}

// Imputacao é uma acusação do Memorando Disciplinar.
type Imputacao struct {
	ID          string
	Texto       string
	Dispositivo string
	Natureza    engine.Natureza
	Fato        civil.Instante
	Evid        []string
}

// EventoAuditoria registra cada mudança do conjunto.
type EventoAuditoria struct {
	Versao  int
	Acao    string
	Detalhe string
}

// ImputacaoSet é o conjunto delimitado pelo memorando.
type ImputacaoSet struct {
	versao    int
	congelado bool
	imps      []Imputacao
	auditoria []EventoAuditoria
}

// NovoImputacaoSet cria um conjunto aberto, versão 1.
func NovoImputacaoSet() *ImputacaoSet {
	return &ImputacaoSet{versao: 1, auditoria: []EventoAuditoria{{1, "criado", ""}}}
}

func (s *ImputacaoSet) registrar(acao, detalhe string) {
	s.auditoria = append(s.auditoria, EventoAuditoria{s.versao, acao, detalhe})
}

// Adicionar inclui uma imputação extraída/confirmada (só com o conjunto aberto).
func (s *ImputacaoSet) Adicionar(im Imputacao) error {
	if s.congelado {
		return ErrCongelado
	}
	if strings.TrimSpace(im.ID) == "" {
		return ErrIDInvalido
	}
	if s.indice(im.ID) >= 0 {
		return fmt.Errorf("%w: %s", ErrIDRepetido, im.ID)
	}
	im.Evid = append([]string(nil), im.Evid...)
	s.imps = append(s.imps, im)
	s.registrar("adicionada", im.ID)
	return nil
}

// Remover retira uma imputação (só com o conjunto aberto).
func (s *ImputacaoSet) Remover(id string) error {
	if s.congelado {
		return ErrCongelado
	}
	i := s.indice(id)
	if i < 0 {
		return fmt.Errorf("%w: %s", ErrForaDoEscopo, id)
	}
	s.imps = append(s.imps[:i], s.imps[i+1:]...)
	s.registrar("removida", id)
	return nil
}

// Congelar fixa o conjunto confirmado pelo usuário.
func (s *ImputacaoSet) Congelar() error {
	if s.congelado {
		return nil
	}
	if len(s.imps) == 0 {
		return ErrVazio
	}
	s.congelado = true
	s.registrar("congelado", fmt.Sprintf("%d imputação(ões)", len(s.imps)))
	return nil
}

// Reabrir volta a delimitação (ex.: o usuário corrigiu a extração). A versão
// sobe; julgamentos iniciados na versão anterior deixam de aceitar resultados.
func (s *ImputacaoSet) Reabrir(motivo string) error {
	if strings.TrimSpace(motivo) == "" {
		return ErrMotivo
	}
	if !s.congelado {
		return nil
	}
	s.congelado = false
	s.versao++
	s.registrar("reaberto", motivo)
	return nil
}

func (s *ImputacaoSet) indice(id string) int {
	for i, im := range s.imps {
		if im.ID == id {
			return i
		}
	}
	return -1
}

// Contem diz se o ID pertence ao conjunto.
func (s *ImputacaoSet) Contem(id string) bool { return s.indice(id) >= 0 }

// Imputacao devolve uma cópia da imputação.
func (s *ImputacaoSet) Imputacao(id string) (Imputacao, bool) {
	i := s.indice(id)
	if i < 0 {
		return Imputacao{}, false
	}
	im := s.imps[i]
	im.Evid = append([]string(nil), im.Evid...)
	return im, true
}

// Imputacoes devolve cópias, na ordem do memorando.
func (s *ImputacaoSet) Imputacoes() []Imputacao {
	out := make([]Imputacao, len(s.imps))
	for i := range s.imps {
		out[i], _ = s.Imputacao(s.imps[i].ID)
	}
	return out
}

// IDs devolve os identificadores, na ordem do memorando (enum dos schemas).
func (s *ImputacaoSet) IDs() []string {
	out := make([]string, len(s.imps))
	for i, im := range s.imps {
		out[i] = im.ID
	}
	return out
}

func (s *ImputacaoSet) Versao() int     { return s.versao }
func (s *ImputacaoSet) Congelado() bool { return s.congelado }
func (s *ImputacaoSet) Auditoria() []EventoAuditoria {
	return append([]EventoAuditoria(nil), s.auditoria...)
}

// Resultado de uma imputação.
type Resultado string

const (
	Procedente   Resultado = "PROCEDENTE"
	Improcedente Resultado = "IMPROCEDENTE"
)

// ResultadoImputacao é o resultado de uma imputação do conjunto.
type ResultadoImputacao struct {
	ImputacaoID          string
	Resultado            Resultado
	ConfirmadoPorUsuario bool
}

// Global é o resultado do PADS (PADS-ESC-003).
type Global string

const (
	Procedencia        Global = "PROCEDENCIA"
	ProcedenciaParcial Global = "PROCEDENCIA_PARCIAL"
	Improcedencia      Global = "IMPROCEDENCIA"
)

// Julgamento liga um acusado a uma versão congelada do conjunto.
type Julgamento struct {
	Acusado    string
	conjunto   *ImputacaoSet
	versao     int
	resultados map[string]ResultadoImputacao
}

// NovoJulgamento exige um acusado e o conjunto congelado.
func NovoJulgamento(acusados []string, s *ImputacaoSet) (*Julgamento, error) {
	if len(acusados) != 1 || strings.TrimSpace(acusados[0]) == "" {
		return nil, ErrAcusadoUnico{Quantidade: len(acusados)}
	}
	if s == nil || !s.congelado {
		return nil, ErrNaoCongelado
	}
	return &Julgamento{Acusado: acusados[0], conjunto: s, versao: s.versao, resultados: map[string]ResultadoImputacao{}}, nil
}

func (j *Julgamento) vigente() error {
	if !j.conjunto.congelado || j.conjunto.versao != j.versao {
		return ErrConjuntoAlterado
	}
	return nil
}

// Registrar grava o resultado de uma imputação do conjunto. ID fora do
// memorando é recusado: o PADS não inova a acusação.
func (j *Julgamento) Registrar(r ResultadoImputacao) error {
	if err := j.vigente(); err != nil {
		return err
	}
	if !j.conjunto.Contem(r.ImputacaoID) {
		return fmt.Errorf("%w: %s", ErrForaDoEscopo, r.ImputacaoID)
	}
	if r.Resultado != Procedente && r.Resultado != Improcedente {
		return fmt.Errorf("resultado inválido %q", r.Resultado)
	}
	j.resultados[r.ImputacaoID] = r
	return nil
}

// Resultados devolve os resultados na ordem do memorando.
func (j *Julgamento) Resultados() []ResultadoImputacao {
	var out []ResultadoImputacao
	for _, id := range j.conjunto.IDs() {
		if r, ok := j.resultados[id]; ok {
			out = append(out, r)
		}
	}
	return out
}

// Pendentes lista imputações sem resultado confirmado.
func (j *Julgamento) Pendentes() []string {
	var out []string
	for _, id := range j.conjunto.IDs() {
		if r, ok := j.resultados[id]; !ok || !r.ConfirmadoPorUsuario {
			out = append(out, id)
		}
	}
	return out
}

// ResultadoGlobal: todas procedentes → procedência; algumas → procedência
// parcial; nenhuma → improcedência. Exige um resultado confirmado por
// imputação do conjunto.
func (j *Julgamento) ResultadoGlobal() (Global, error) {
	if err := j.vigente(); err != nil {
		return "", err
	}
	if p := j.Pendentes(); len(p) > 0 {
		return "", fmt.Errorf("%w: %s", ErrPendente, strings.Join(p, ", "))
	}
	n := 0
	for _, r := range j.resultados {
		if r.Resultado == Procedente {
			n++
		}
	}
	switch {
	case n == 0:
		return Improcedencia, nil
	case n == len(j.resultados):
		return Procedencia, nil
	}
	return ProcedenciaParcial, nil
}

// Procedentes converte as imputações procedentes para o motor; grupos
// associa ID → grupo de conexão confirmado (vazio = independente).
func (j *Julgamento) Procedentes(grupos map[string]string) []engine.ImputacaoProcedente {
	var out []engine.ImputacaoProcedente
	for _, r := range j.Resultados() {
		if r.Resultado != Procedente || !r.ConfirmadoPorUsuario {
			continue
		}
		im, _ := j.conjunto.Imputacao(r.ImputacaoID)
		out = append(out, engine.ImputacaoProcedente{ID: im.ID, Natureza: im.Natureza, Fato: im.Fato, Grupo: grupos[im.ID]})
	}
	return out
}

// Violacao é a quebra de uma invariante do julgamento.
type Violacao struct {
	Regra    string // ID do catálogo
	Ref      string
	Mensagem string
}

// VerificarCalculo confere, em runtime, as invariantes entre o julgamento e
// o cálculo oficial (plano §8):
//   - sanção ⇒ imputação do conjunto julgada procedente e confirmada;
//   - no máximo uma sanção por imputação (nas conexas, só a principal);
//   - regressão especial ⇒ nenhum desconto ordinário;
//   - desconto só para imputação sancionada.
func (j *Julgamento) VerificarCalculo(sancoes []engine.SancaoAplicada, p engine.ResultadoPontuacao) []Violacao {
	var vs []Violacao
	sancionadas := map[string]bool{}
	for _, s := range sancoes {
		r, ok := j.resultados[s.ImputacaoID]
		switch {
		case !j.conjunto.Contem(s.ImputacaoID):
			vs = append(vs, Violacao{"PADS-ESC-002", s.ImputacaoID, "sanção para imputação fora do memorando"})
		case !ok || r.Resultado != Procedente || !r.ConfirmadoPorUsuario:
			vs = append(vs, Violacao{"PADS-ESC-003", s.ImputacaoID, "sanção sem procedência confirmada"})
		}
		if sancionadas[s.ImputacaoID] {
			vs = append(vs, Violacao{"PADS-CNX-002", s.ImputacaoID, "mais de uma sanção para a mesma imputação"})
		}
		sancionadas[s.ImputacaoID] = true
	}
	if p.Regressao != engine.SemRegressao && len(p.Descontos) > 0 {
		vs = append(vs, Violacao{"PADS-CMP-006", string(p.Regressao), "regressão especial acumulada com descontos ordinários"})
	}
	for _, d := range p.Descontos {
		if !sancionadas[d.ImputacaoID] {
			vs = append(vs, Violacao{"PADS-CMP-004", d.ImputacaoID, "desconto sem sanção correspondente"})
		}
	}
	return vs
}
