package engine

import (
	"errors"
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
)

// PunicaoAnterior é uma punição já aplicada ao acusado.
type PunicaoAnterior struct {
	ID       string
	Natureza Natureza
	// Termino do cumprimento; nil = desconhecido (V4 §15.5-2).
	Termino *civil.Instante
}

// Descarte explica por que uma punição anterior não foi considerada.
type Descarte struct {
	ID     string
	Motivo string
}

// ResultadoReincidencia é a conclusão sobre a agravante do art. 39, III.
type ResultadoReincidencia struct {
	// Reconhecida: no máximo UMA agravante, por mais punições válidas que
	// existam (V4 §15.5; R9; A25).
	Reconhecida bool
	Validas     []string
	Descartadas []Descarte
	Pendencias  []Pendencia
}

// Reincidencia aplica a regra funcional final (V4 §§7.1, 15.5; D-07), para UMA
// transgressão atual procedente (a principal, se conexas):
//  1. mesma natureza (leve/leve, média/média, grave/grave);
//  2. data de término do cumprimento conhecida;
//  3. término ESTRITAMENTE anterior ao fato atual (na mesma data, só com
//     horários que provem a anterioridade);
//  4. fato atual até n anos depois do término, contado o aniversário (D-07).
func Reincidencia(natureza Natureza, fato civil.Instante, anteriores []PunicaoAnterior, regra civil.Regra29Fev) ResultadoReincidencia {
	var r ResultadoReincidencia
	anos := natureza.PrazoReincidencia()
	for _, a := range anteriores {
		if a.Natureza != natureza {
			r.Descartadas = append(r.Descartadas, Descarte{a.ID, fmt.Sprintf("natureza diferente (%s × %s): só se compara mesma natureza", rotuloNat(a.Natureza), rotuloNat(natureza))})
			continue
		}
		if a.Termino == nil || a.Termino.Data.Zero() {
			r.Descartadas = append(r.Descartadas, Descarte{a.ID, "data de término do cumprimento não comprovada"})
			r.Pendencias = append(r.Pendencias, pend(PendDataTerminoDesconhecida, a.ID,
				"Informe a data de término do cumprimento da punição anterior (DD/MM/AAAA); sem ela, a reincidência não é reconhecida."))
			continue
		}
		switch civil.Comparar(*a.Termino, fato) {
		case civil.Depois:
			r.Descartadas = append(r.Descartadas, Descarte{a.ID, "o cumprimento terminou depois do fato atual"})
			continue
		case civil.Simultaneo:
			r.Descartadas = append(r.Descartadas, Descarte{a.ID, "término e fato no mesmo horário: não há anterioridade"})
			continue
		case civil.Indeterminado:
			r.Descartadas = append(r.Descartadas, Descarte{a.ID, "mesma data sem horários que comprovem a anterioridade"})
			r.Pendencias = append(r.Pendencias, pend(PendHorarios, a.ID,
				"Término do cumprimento e fato atual na mesma data (%s): informe os horários; sem eles, a reincidência não é reconhecida.", fato.Data))
			continue
		}
		ok, err := civil.DentroPrazoReincidencia(a.Termino.Data, fato.Data, anos, regra)
		if errors.Is(err, civil.ErrCasoLimite29Fev) {
			r.Descartadas = append(r.Descartadas, Descarte{a.ID, "prazo termina em aniversário de 29/02: aguarda decisão do usuário"})
			r.Pendencias = append(r.Pendencias, pend(PendCaso29Fev, a.ID,
				"O término do cumprimento foi em 29/02 e o prazo de %d ano(s) termina em ano não bissexto: considerar 28/02 ou 01/03?", anos))
			continue
		}
		if !ok {
			r.Descartadas = append(r.Descartadas, Descarte{a.ID, fmt.Sprintf("decorrido tempo superior a %d ano(s) entre o término e o fato", anos)})
			continue
		}
		r.Validas = append(r.Validas, a.ID)
	}
	r.Reconhecida = len(r.Validas) > 0
	return r
}

func rotuloNat(n Natureza) string {
	switch n {
	case Leve:
		return "leve"
	case Media:
		return "média"
	case Grave:
		return "grave"
	}
	return "desconhecida"
}

// PerguntaReincidencia é a próxima pergunta necessária (V4 §7.2; PADS-REI-009).
type PerguntaReincidencia string

const (
	SemPergunta           PerguntaReincidencia = ""
	PerguntarSeHaAnterior PerguntaReincidencia = "POSSUI_ANTERIOR_MESMA_NATUREZA"
	PerguntarDataTermino  PerguntaReincidencia = "DATA_TERMINO_CUMPRIMENTO"
)

// ProximaPerguntaReincidencia decide se e o que perguntar. Só depois de
// fixada a natureza atual; resposta "não" encerra sem pedir datas; data já
// conhecida (documentos) não é perguntada de novo.
func ProximaPerguntaReincidencia(natureza *Natureza, possuiAnterior *bool, terminoConhecido bool) PerguntaReincidencia {
	switch {
	case natureza == nil || !natureza.Valida():
		return SemPergunta
	case possuiAnterior == nil:
		return PerguntarSeHaAnterior
	case !*possuiAnterior:
		return SemPergunta
	case !terminoConhecido:
		return PerguntarDataTermino
	}
	return SemPergunta
}
