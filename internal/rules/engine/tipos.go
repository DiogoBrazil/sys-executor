// Package engine é o motor determinístico dos cálculos disciplinares do PADS
// (plano §8; V4 §§5–8, 13, 15; decisões D-05, D-06, D-07; interpretações I-01…I-04).
//
// A IA não decide nada aqui: o motor recebe dados confirmados (natureza,
// datas, procedência, histórico reconhecido) e devolve resultados, os fatos
// usados, os descartados com motivo e as PENDÊNCIAS que exigem pergunta ao
// usuário. Nunca inventa dado ausente.
package engine

import (
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
)

// Natureza da transgressão (RDPM arts. 15–17; V4 §4).
type Natureza string

const (
	Leve  Natureza = "LEVE"
	Media Natureza = "MEDIA"
	Grave Natureza = "GRAVE"
)

// Unidades para equivalência (I-01): leve 1, média 2, grave 4.
func (n Natureza) Unidades() int {
	switch n {
	case Leve:
		return 1
	case Media:
		return 2
	case Grave:
		return 4
	}
	return 0
}

// PrazoReincidencia em anos: leve 1, média 2, grave 4 (V4 §7.1).
func (n Natureza) PrazoReincidencia() int {
	switch n {
	case Leve:
		return 1
	case Media:
		return 2
	case Grave:
		return 4
	}
	return 0
}

// Sancao correspondente à natureza (V4 §4).
func (n Natureza) Sancao() Sancao {
	switch n {
	case Leve:
		return Repreensao
	case Media:
		return Detencao
	case Grave:
		return Prisao
	}
	return ""
}

// Valida informa se a natureza é conhecida.
func (n Natureza) Valida() bool { return n == Leve || n == Media || n == Grave }

// Sancao disciplinar.
type Sancao string

const (
	Repreensao Sancao = "REPREENSAO"
	Detencao   Sancao = "DETENCAO"
	Prisao     Sancao = "PRISAO"
)

// Desconto ordinário em pontos (RDPM art. 71; V4 §8.2).
func (s Sancao) Desconto() int {
	switch s {
	case Repreensao:
		return 1
	case Detencao:
		return 2
	case Prisao:
		return 4
	}
	return 0
}

// Comportamento disciplinar (RDPM art. 70).
type Comportamento string

const (
	Excepcional  Comportamento = "EXCEPCIONAL"
	Otimo        Comportamento = "OTIMO"
	Bom          Comportamento = "BOM"
	Insuficiente Comportamento = "INSUFICIENTE"
)

// ClassificarSaldo aplica as faixas do art. 70 (V4 §8.1): mais de 40 →
// excepcional; 30 a 40 → ótimo; 10 a 29 → bom; menos de 10 → insuficiente.
func ClassificarSaldo(pontos int) Comportamento {
	switch {
	case pontos > 40:
		return Excepcional
	case pontos >= 30:
		return Otimo
	case pontos >= 10:
		return Bom
	}
	return Insuficiente
}

// Origem de uma transgressão no cálculo.
type Origem string

const (
	// Atual: imputação do PADS em julgamento, considerada PROCEDENTE.
	Atual Origem = "ATUAL"
	// Historico: transgressão anterior RECONHECIDA em decisão disciplinar.
	Historico Origem = "HISTORICO"
	// Pendente: acusação ainda não decidida — nunca entra em cálculo (V4 §8.7).
	Pendente Origem = "PENDENTE"
)

// Transgressao é um fato considerado nos gatilhos de comportamento.
type Transgressao struct {
	ID       string
	Natureza Natureza
	Data     civil.Data // data de COMETIMENTO
	Origem   Origem
	// Só para o excepcional (V4 §15.3): confirmada pelo usuário como do
	// período atual em excepcional, e não consumida em regressão anterior.
	PeriodoExcepcionalAtual bool
	ConsumidaEmRegressao    bool
}

// TipoPendencia identifica o que precisa ser perguntado ou confirmado.
type TipoPendencia string

const (
	PendDataTerminoDesconhecida  TipoPendencia = "DATA_TERMINO_DESCONHECIDA" // reincidência (R3, A26)
	PendHorarios                 TipoPendencia = "HORARIOS_NECESSARIOS"      // mesma data, sem horas (R7)
	PendCaso29Fev                TipoPendencia = "CASO_29_FEV"               // P-03
	PendDadosComportamento       TipoPendencia = "DADOS_COMPORTAMENTO"       // pontos/classificação ausentes
	PendDivergenciaComportamento TipoPendencia = "DIVERGENCIA_COMPORTAMENTO" // pontos × classificação (§8.1)
	PendHistoricoSozinho         TipoPendencia = "HISTORICO_SOZINHO_GATILHO" // D-06
	PendExcepcionalInconsistente TipoPendencia = "EXCEPCIONAL_INCONSISTENTE" // I-02
	PendSaldoNegativo            TipoPendencia = "SALDO_NEGATIVO"            // P-04
	PendDataFatoAtual            TipoPendencia = "DATA_FATO_ATUAL"           // fato atual sem data
	PendPrincipalConexao         TipoPendencia = "PRINCIPAL_CONEXAO"         // I-04
	PendFundamentacao            TipoPendencia = "FUNDAMENTACAO_AUSENTE"     // circunstância sem fundamento
	PendArt36                    TipoPendencia = "ART36_INCOMPLETO"          // fatores I–VII
)

// Pendencia é algo que o usuário precisa responder; o cálculo afetado fica
// marcado como não concluído.
type Pendencia struct {
	Tipo     TipoPendencia
	Ref      string // ID do fato, punição ou grupo envolvido
	Mensagem string
}

func pend(t TipoPendencia, ref, formato string, args ...any) Pendencia {
	return Pendencia{Tipo: t, Ref: ref, Mensagem: fmt.Sprintf(formato, args...)}
}
