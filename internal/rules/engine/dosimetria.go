package engine

import (
	"fmt"
	"sort"
	"strings"
)

// Circunstancia reconhecida (atenuante art. 38, agravante art. 39, causa de
// justificação art. 37), sempre com fundamento específico (V4 §6.2).
type Circunstancia struct {
	Artigo     int
	Inciso     int // em algarismo arábico (VII = 7)
	Alinea     string
	Fundamento string
}

// Chave legível, ex.: "art. 39, VII, a".
func (c Circunstancia) Chave() string {
	s := fmt.Sprintf("art. %d, %s", c.Artigo, romano(c.Inciso))
	if c.Alinea != "" {
		s += ", " + c.Alinea
	}
	return s
}

func romano(n int) string {
	vals := []int{10, 9, 5, 4, 1}
	sims := []string{"X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(sims[i])
			n -= v
		}
	}
	return b.String()
}

// EntradaDosimetria reúne o que foi reconhecido para UMA sanção.
type EntradaDosimetria struct {
	Sancao        Sancao
	Atenuantes    []Circunstancia // art. 38 (exceto I e II, tratados abaixo)
	Agravantes    []Circunstancia // art. 39 (exceto I, II e III, tratados abaixo)
	Justificacoes []Circunstancia // art. 37: só exigem fundamentação aqui
	// Reincidencia: resultado do motor (art. 39, III), no máximo uma vez.
	Reincidencia bool
	// ConexasSecundarias absorvidas como agravante (inciso II, no máximo uma vez).
	ConexasSecundarias int
	// ComportamentoAnterior ao julgamento: ÓT/EXC → art. 38, I; INS → art. 39, I.
	ComportamentoAnterior Comportamento
	// Relevantes serviços (art. 38, II): nil = não perguntado; sim exige justificativa.
	ServicosRelevantes    *bool
	JustificativaServicos string
}

// ResultadoDosimetria é a proposta de dias e a memória da contagem.
type ResultadoDosimetria struct {
	Dias               int // 0 para repreensão (sem dias)
	AtenuantesContadas []string
	AgravantesContadas []string
	Regra              string
	Pendencias         []Pendencia
}

// Dosimetria aplica a heurística do usuário (V4 §6.3) com a contagem da D-05:
// nenhuma ou quantidades iguais → 2 dias; mais agravantes → 3; mais
// atenuantes → 1; a diferença não multiplica; repreensão não tem dias.
// Cada alínea do art. 39, VII conta uma; incisos II e III no máximo uma vez.
func Dosimetria(e EntradaDosimetria) ResultadoDosimetria {
	var r ResultadoDosimetria
	at, ag := map[string]bool{}, map[string]bool{}
	exigirFundamento := func(c Circunstancia, tipo string) bool {
		if strings.TrimSpace(c.Fundamento) == "" {
			r.Pendencias = append(r.Pendencias, pend(PendFundamentacao, c.Chave(),
				"A %s %s foi indicada sem fundamentação específica e não foi considerada.", tipo, c.Chave()))
			return false
		}
		return true
	}
	for _, c := range e.Justificacoes {
		exigirFundamento(c, "causa de justificação")
	}
	for _, c := range e.Atenuantes {
		if c.Artigo != 38 || c.Inciso == 1 || c.Inciso == 2 {
			continue // I e II têm regra própria abaixo
		}
		if exigirFundamento(c, "atenuante") {
			at[c.Chave()] = true
		}
	}
	for _, c := range e.Agravantes {
		if c.Artigo != 39 || c.Inciso == 1 || c.Inciso == 3 {
			continue // I (comportamento) e III (reincidência) vêm do motor
		}
		if !exigirFundamento(c, "agravante") {
			continue
		}
		if c.Inciso == 2 {
			ag["art. 39, II"] = true // conexão: no máximo uma vez
			continue
		}
		ag[c.Chave()] = true // VII: cada alínea conta
	}
	switch e.ComportamentoAnterior {
	case Otimo, Excepcional:
		at["art. 38, I"] = true
	case Insuficiente:
		ag["art. 39, I"] = true
	}
	if e.ServicosRelevantes != nil && *e.ServicosRelevantes {
		if strings.TrimSpace(e.JustificativaServicos) == "" {
			r.Pendencias = append(r.Pendencias, pend(PendFundamentacao, "art. 38, II",
				"Relevantes serviços: informe uma justificativa sucinta e factual; sem ela, a atenuante não é considerada."))
		} else {
			at["art. 38, II"] = true
		}
	}
	if e.ConexasSecundarias > 0 {
		ag["art. 39, II"] = true
	}
	if e.Reincidencia {
		ag["art. 39, III"] = true
	}
	r.AtenuantesContadas, r.AgravantesContadas = ordenar(at), ordenar(ag)
	na, ng := len(at), len(ag)
	if e.Sancao == Repreensao {
		r.Regra = "repreensão: sem cálculo de dias"
		return r
	}
	switch {
	case ng > na:
		r.Dias, r.Regra = 3, fmt.Sprintf("%d atenuante(s) × %d agravante(s): mais agravantes → 3 dias", na, ng)
	case na > ng:
		r.Dias, r.Regra = 1, fmt.Sprintf("%d atenuante(s) × %d agravante(s): mais atenuantes → 1 dia", na, ng)
	default:
		r.Dias, r.Regra = 2, fmt.Sprintf("%d atenuante(s) × %d agravante(s): quantidades iguais → 2 dias", na, ng)
	}
	return r
}

func ordenar(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LimiteDiasDetencao: detenção limitada a 10 dias (RDPM art. 58, II, b; I-03).
const LimiteDiasDetencao = 10

// ValidarDias confere a compatibilidade dos dias com a sanção (DOS-LIM-01).
func ValidarDias(s Sancao, dias int) error {
	switch {
	case s == Repreensao && dias != 0:
		return fmt.Errorf("repreensão não tem duração em dias")
	case s == Detencao && (dias < 1 || dias > LimiteDiasDetencao):
		return fmt.Errorf("detenção deve ter de 1 a %d dias", LimiteDiasDetencao)
	case s == Prisao && dias < 1:
		return fmt.Errorf("prisão deve ter ao menos 1 dia")
	}
	return nil
}

// VerificarArt36 exige os sete fatores do art. 36 (I a VII) quando houver
// punição, ainda que como "ausência de elementos" (V4 §6.1; DOS-36-01).
func VerificarArt36(fatores map[int]string) []Pendencia {
	var ps []Pendencia
	for i := 1; i <= 7; i++ {
		if strings.TrimSpace(fatores[i]) == "" {
			ps = append(ps, pend(PendArt36, romano(i), "Fator do art. 36, %s não examinado (indique a análise ou a ausência de elementos).", romano(i)))
		}
	}
	return ps
}
