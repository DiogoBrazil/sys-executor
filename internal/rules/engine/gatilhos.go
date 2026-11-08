package engine

import (
	"errors"
	"sort"

	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
)

// Janela é um conjunto de fatos com datas em intervalo estritamente inferior
// a um ano (V4 §15.2), a partir do fato mais antigo.
type Janela struct {
	Inicio      civil.Data
	Limite      civil.Data // aniversário do início (exclusivo)
	Fatos       []Transgressao
	Equivalente Equivalente
	ContemAtual bool
}

// ResultadoBomOtimo é o gatilho de regressão especial para BOM/ÓTIMO (V4 §§8.5,
// 15.1, 15.2; D-06): equivalente a 2 graves (8 unidades) com datas de
// cometimento em menos de 1 ano, contendo ao menos uma procedente do PADS atual.
type ResultadoBomOtimo struct {
	Acionado bool
	Janela   *Janela // janela usada (subconjunto demonstrado, DT-05)
	// HistoricoSozinho: o histórico reconhecido, sem fato atual, já fecha o
	// gatilho — possível inconsistência com a classificação atual (D-06).
	HistoricoSozinho *Janela
	Pendencias       []Pendencia
}

const unidadesDuasGraves = 8

// GatilhoBomOtimo procura a janela válida. Fatos pendentes são ignorados;
// fatos sem data geram pendência (a janela exige datas individuais, V4 §8.5).
func GatilhoBomOtimo(fatos []Transgressao, regra civil.Regra29Fev) ResultadoBomOtimo {
	var res ResultadoBomOtimo
	var validos []Transgressao
	vistos := map[string]bool{}
	for _, f := range fatos {
		if f.Origem == Pendente || !f.Natureza.Valida() || vistos[f.ID] {
			continue
		}
		vistos[f.ID] = true
		if f.Data.Zero() {
			res.Pendencias = append(res.Pendencias, pend(PendDataFatoAtual, f.ID, "Informe a data de cometimento da transgressão %s para verificar a janela de um ano.", f.ID))
			continue
		}
		validos = append(validos, f)
	}
	sort.SliceStable(validos, func(i, j int) bool {
		if c := validos[i].Data.Cmp(validos[j].Data); c != 0 {
			return c < 0
		}
		return validos[i].ID < validos[j].ID
	})

	r, err := buscarJanelas(validos, regra)
	if errors.Is(err, civil.ErrCasoLimite29Fev) {
		// Caso-limite de 29/02: só pergunta se a escolha mudar o resultado.
		a, _ := buscarJanelas(validos, civil.Usar28Fev)
		b, _ := buscarJanelas(validos, civil.Usar01Mar)
		if (a.Acionado == b.Acionado) && ((a.HistoricoSozinho == nil) == (b.HistoricoSozinho == nil)) {
			r = a
		} else {
			res.Pendencias = append(res.Pendencias, pend(PendCaso29Fev, "", "Há fato em 29/02 e a janela de um ano termina em ano não bissexto; o resultado muda conforme o critério: considerar 28/02 ou 01/03?"))
			return res
		}
	}
	res.Acionado, res.Janela, res.HistoricoSozinho = r.Acionado, r.Janela, r.HistoricoSozinho
	if res.HistoricoSozinho != nil && !res.Acionado {
		res.Pendencias = append(res.Pendencias, pend(PendHistoricoSozinho, "",
			"O histórico reconhecido, sozinho, já equivale a duas graves em menos de um ano (%s a %s), o que pode ser inconsistente com o comportamento atual informado. Confirme os dados.",
			res.HistoricoSozinho.Inicio, res.HistoricoSozinho.Fatos[len(res.HistoricoSozinho.Fatos)-1].Data))
	}
	return res
}

func buscarJanelas(fatos []Transgressao, regra civil.Regra29Fev) (ResultadoBomOtimo, error) {
	var res ResultadoBomOtimo
	for i := 0; i < len(fatos); i++ {
		if i > 0 && fatos[i].Data == fatos[i-1].Data {
			continue // mesma data de início já avaliada
		}
		lim, err := civil.Aniversario(fatos[i].Data, 1, regra)
		if err != nil {
			return res, err
		}
		j := &Janela{Inicio: fatos[i].Data, Limite: lim}
		for k := i; k < len(fatos) && fatos[k].Data.Cmp(lim) < 0; k++ {
			j.Fatos = append(j.Fatos, fatos[k])
			if fatos[k].Origem == Atual {
				j.ContemAtual = true
			}
		}
		j.Equivalente = Equivaler(j.Fatos)
		if j.Equivalente.Unidades < unidadesDuasGraves {
			continue
		}
		if j.ContemAtual {
			res.Acionado, res.Janela = true, j
			return res, nil
		}
		if res.HistoricoSozinho == nil {
			res.HistoricoSozinho = j
		}
	}
	return res, nil
}

// ResultadoExcepcional é o gatilho da 3ª grave equivalente para o EXCEPCIONAL
// (V4 §§8.6, 15.3; I-02), sem janela de um ano.
type ResultadoExcepcional struct {
	Acionado        bool
	UnidadesPrevias int
	UnidadesTotais  int
	Previas, Atuais []string
	Desconsideradas []Descarte
	Pendencias      []Pendencia
}

const unidadesTresGraves = 12

// GatilhoExcepcional considera só o histórico reconhecido CONFIRMADO como do
// período atual em excepcional e não consumido em regressão anterior (EX2),
// mais as procedentes do PADS atual. Prévias < 12 unidades e prévias + atuais
// ≥ 12 → aciona. Prévias já ≥ 12 → inconsistência: pergunta (I-02).
func GatilhoExcepcional(fatos []Transgressao) ResultadoExcepcional {
	var r ResultadoExcepcional
	var previas, atuais []Transgressao
	for _, f := range fatos {
		switch {
		case f.Origem == Pendente:
			r.Desconsideradas = append(r.Desconsideradas, Descarte{f.ID, "acusação pendente, sem decisão disciplinar"})
		case f.Origem == Atual:
			atuais = append(atuais, f)
		case f.ConsumidaEmRegressao:
			r.Desconsideradas = append(r.Desconsideradas, Descarte{f.ID, "já usada em regressão anterior de excepcional para ótimo"})
		case !f.PeriodoExcepcionalAtual:
			r.Desconsideradas = append(r.Desconsideradas, Descarte{f.ID, "não pertence ao período atual em excepcional"})
		default:
			previas = append(previas, f)
		}
	}
	ep, ea := Equivaler(previas), Equivaler(atuais)
	r.UnidadesPrevias, r.UnidadesTotais = ep.Unidades, ep.Unidades+ea.Unidades
	r.Previas, r.Atuais = ep.Usadas, ea.Usadas
	if r.UnidadesPrevias >= unidadesTresGraves {
		r.Pendencias = append(r.Pendencias, pend(PendExcepcionalInconsistente, "",
			"O histórico do período atual já equivale a três ou mais graves, o que é incompatível com o comportamento excepcional informado. Confirme quais transgressões pertencem ao período atual."))
		return r
	}
	r.Acionado = r.UnidadesTotais >= unidadesTresGraves
	return r
}
