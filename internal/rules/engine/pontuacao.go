package engine

import "github.com/DiogoBrazil/sys-executor/internal/rules/civil"

// SancaoAplicada é a punição proposta para uma imputação procedente (nas
// conexas, só a principal recebe sanção — V4 §§5.1, 8.7).
type SancaoAplicada struct {
	ImputacaoID string
	Natureza    Natureza
	Sancao      Sancao
}

// EntradaPontuacao reúne os dados confirmados para o cálculo do comportamento.
type EntradaPontuacao struct {
	Comportamento *Comportamento // informado/confirmado (nil = ausente)
	Pontos        *int           // informado/confirmado (nil = ausente)
	Sancoes       []SancaoAplicada
	// Fatos: histórico reconhecido + procedentes do PADS atual (nas conexas,
	// só a principal). Processos distintos são sempre histórico (V4 §15.4).
	Fatos      []Transgressao
	Regra29Fev civil.Regra29Fev
}

// TipoRegressao identifica a regressão especial aplicada.
type TipoRegressao string

const (
	SemRegressao         TipoRegressao = ""
	RegressaoBomOtimo    TipoRegressao = "BOM_OTIMO"
	RegressaoExcepcional TipoRegressao = "EXCEPCIONAL"
)

// DescontoAplicado é um desconto ordinário (art. 71).
type DescontoAplicado struct {
	ImputacaoID string
	Sancao      Sancao
	Pontos      int
}

// ResultadoPontuacao é o comportamento proposto após o julgamento.
type ResultadoPontuacao struct {
	Concluido             bool
	ComportamentoAnterior Comportamento
	PontosAnteriores      int
	Regressao             TipoRegressao
	BomOtimo              *ResultadoBomOtimo
	Excepcional           *ResultadoExcepcional
	Descontos             []DescontoAplicado
	PontosFinais          int
	ComportamentoFinal    Comportamento
	SaldoNegativo         bool
	Pendencias            []Pendencia
}

// Pontuacao calcula o comportamento proposto (V4 §§8, 15.4):
//   - dados ausentes → não concluído ([A PREENCHER]); divergência entre pontos
//     e classificação → pergunta, sem calcular;
//   - EXCEPCIONAL: 3ª grave equivalente do período atual → ÓTIMO, 30 pontos;
//   - BOM/ÓTIMO: 2 graves equivalentes em menos de 1 ano com fato atual →
//     INSUFICIENTE 9 / BOM 20;
//   - havendo regressão especial, a pontuação é o valor fixo, SEM nenhum
//     desconto ordinário das sanções deste PADS (§15.4);
//   - senão, saldo − Σ descontos (repreensão 1, detenção 2, prisão 4);
//     INSUFICIENTE nunca volta a 9 (§8.3).
func Pontuacao(e EntradaPontuacao) ResultadoPontuacao {
	var r ResultadoPontuacao
	if e.Comportamento == nil || e.Pontos == nil {
		r.Pendencias = append(r.Pendencias, pend(PendDadosComportamento, "",
			"Informe a pontuação e o comportamento atuais da ficha individual; sem eles, o cálculo não é concluído e a minuta usa [A PREENCHER]."))
		return r
	}
	r.ComportamentoAnterior, r.PontosAnteriores = *e.Comportamento, *e.Pontos
	if c := ClassificarSaldo(*e.Pontos); c != *e.Comportamento {
		r.Pendencias = append(r.Pendencias, pend(PendDivergenciaComportamento, "",
			"O comportamento informado (%s) não corresponde à pontuação informada (%d pontos, faixa %s). Esclareça antes do cálculo.",
			*e.Comportamento, *e.Pontos, c))
		return r
	}

	switch *e.Comportamento {
	case Excepcional:
		g := GatilhoExcepcional(e.Fatos)
		r.Excepcional = &g
		r.Pendencias = append(r.Pendencias, g.Pendencias...)
		if len(g.Pendencias) > 0 {
			return r
		}
		if g.Acionado {
			return fixar(r, RegressaoExcepcional, 30, Otimo)
		}
	case Otimo, Bom:
		g := GatilhoBomOtimo(e.Fatos, e.Regra29Fev)
		r.BomOtimo = &g
		r.Pendencias = append(r.Pendencias, g.Pendencias...)
		if len(g.Pendencias) > 0 {
			return r
		}
		if g.Acionado {
			if *e.Comportamento == Bom {
				return fixar(r, RegressaoBomOtimo, 9, Insuficiente)
			}
			return fixar(r, RegressaoBomOtimo, 20, Bom)
		}
	}

	// Desconto ordinário (inclusive INSUFICIENTE, sem reposição a 9).
	saldo := *e.Pontos
	for _, s := range e.Sancoes {
		d := s.Sancao.Desconto()
		r.Descontos = append(r.Descontos, DescontoAplicado{s.ImputacaoID, s.Sancao, d})
		saldo -= d
	}
	r.PontosFinais, r.ComportamentoFinal = saldo, ClassificarSaldo(saldo)
	r.Concluido = true
	if saldo < 0 {
		r.SaldoNegativo = true
		r.Pendencias = append(r.Pendencias, pend(PendSaldoNegativo, "",
			"O saldo calculado ficou negativo (%d pontos). O cálculo foi feito, mas este caso não está previsto: confirme como deve constar na minuta.", saldo))
	}
	return r
}

func fixar(r ResultadoPontuacao, t TipoRegressao, pontos int, c Comportamento) ResultadoPontuacao {
	r.Regressao, r.PontosFinais, r.ComportamentoFinal, r.Concluido = t, pontos, c, true
	r.Descontos = nil // nenhum desconto ordinário do mesmo PADS (§15.4)
	return r
}
