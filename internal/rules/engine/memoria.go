package engine

import (
	"fmt"
	"sort"
)

// ItemMemoria é um dos sete itens da memória de cálculo (V4 §8.8).
type ItemMemoria struct {
	Titulo string
	Linhas []string
}

// Memoria é a memória de cálculo do comportamento, mostrada ao usuário antes
// da aprovação da minuta (PADS-CMP-015). Não afirma alteração administrativa.
type Memoria struct {
	Itens [7]ItemMemoria
}

var rotulosComp = map[Comportamento]string{Excepcional: "excepcional", Otimo: "ótimo", Bom: "bom", Insuficiente: "insuficiente"}

// MontarMemoria descreve, em pt-BR, o cálculo feito por Pontuacao.
func MontarMemoria(e EntradaPontuacao, r ResultadoPontuacao, conx ResultadoConexao) Memoria {
	var m Memoria
	// 1. Atuais informados.
	m.Itens[0].Titulo = "Pontuação e comportamento atuais"
	if e.Pontos == nil || e.Comportamento == nil {
		m.Itens[0].Linhas = []string{"[A PREENCHER] — não informados; cálculo não concluído."}
	} else {
		m.Itens[0].Linhas = []string{fmt.Sprintf("%d pontos, comportamento %s (informados/confirmados pelo usuário).", *e.Pontos, rotulosComp[*e.Comportamento])}
	}
	// 2. Imputações procedentes.
	m.Itens[1].Titulo = "Imputações procedentes consideradas"
	for _, p := range conx.Principais {
		l := fmt.Sprintf("%s: natureza %s, fato em %s, sanção %s", p.ID, rotuloNat(p.Natureza), p.Fato.Data, rotuloSancao(p.Natureza.Sancao()))
		if sec := conx.Secundarias[p.ID]; len(sec) > 0 {
			l += fmt.Sprintf("; principal das conexas %v (absorvidas como agravante, sem sanção própria)", sec)
		}
		m.Itens[1].Linhas = append(m.Itens[1].Linhas, l+".")
	}
	// 3. Histórico.
	m.Itens[2].Titulo = "Histórico disciplinar anterior utilizado"
	for _, f := range e.Fatos {
		if f.Origem != Historico {
			continue
		}
		m.Itens[2].Linhas = append(m.Itens[2].Linhas, fmt.Sprintf("%s: natureza %s, cometida em %s, reconhecida em decisão disciplinar.", f.ID, rotuloNat(f.Natureza), dataOu(f)))
	}
	if len(m.Itens[2].Linhas) == 0 {
		m.Itens[2].Linhas = []string{"Nenhum histórico reconhecido informado."}
	}
	// 4. Equivalências e janela.
	m.Itens[3].Titulo = "Equivalências e janela temporal"
	switch {
	case r.BomOtimo != nil && r.BomOtimo.Janela != nil:
		j := r.BomOtimo.Janela
		m.Itens[3].Linhas = append(m.Itens[3].Linhas,
			fmt.Sprintf("Fatos de %s a %s (anterior ao aniversário %s, intervalo inferior a um ano): %v.", j.Inicio, j.Fatos[len(j.Fatos)-1].Data, j.Limite, ids(j.Fatos)),
			fmt.Sprintf("Equivalência: %d unidade(s) = %s (2 leves = 1 média; 2 médias = 1 grave).", j.Equivalente.Unidades, j.Equivalente))
	case r.Excepcional != nil:
		x := r.Excepcional
		m.Itens[3].Linhas = append(m.Itens[3].Linhas,
			fmt.Sprintf("Período atual em excepcional, sem janela de um ano: anteriores %v (%d unidade(s)) + atuais %v = %d unidade(s); a terceira grave equivale a 12.", x.Previas, x.UnidadesPrevias, x.Atuais, x.UnidadesTotais))
		for _, d := range x.Desconsideradas {
			m.Itens[3].Linhas = append(m.Itens[3].Linhas, fmt.Sprintf("Desconsiderada %s: %s.", d.ID, d.Motivo))
		}
	default:
		m.Itens[3].Linhas = []string{"Gatilho de regressão especial não atingido (ou não aplicável ao comportamento atual)."}
	}
	// 5. Regressão ou desconto.
	m.Itens[4].Titulo = "Regressão especial ou desconto ordinário"
	switch r.Regressao {
	case RegressaoBomOtimo, RegressaoExcepcional:
		m.Itens[4].Linhas = []string{fmt.Sprintf("Regressão especial (art. 70): pontuação fixa de %d pontos; não se acumulam os descontos ordinários das punições deste julgamento.", r.PontosFinais)}
	default:
		for _, d := range r.Descontos {
			m.Itens[4].Linhas = append(m.Itens[4].Linhas, fmt.Sprintf("%s: %s, −%d ponto(s) (art. 71).", d.ImputacaoID, rotuloSancao(d.Sancao), d.Pontos))
		}
		if len(m.Itens[4].Linhas) == 0 {
			m.Itens[4].Linhas = []string{"Sem descontos."}
		}
	}
	// 6. Proposto.
	m.Itens[5].Titulo = "Pontuação e comportamento propostos"
	if r.Concluido {
		m.Itens[5].Linhas = []string{fmt.Sprintf("Propõe-se %d pontos, comportamento %s. A alteração depende do ato administrativo competente.", r.PontosFinais, rotulosComp[r.ComportamentoFinal])}
	} else {
		m.Itens[5].Linhas = []string{"[A PREENCHER] — cálculo não concluído (ver pendências)."}
	}
	// 7. Fundamento e ressalvas.
	m.Itens[6].Titulo = "Fundamento legal e ressalvas"
	m.Itens[6].Linhas = []string{"RDPM, arts. 70 e 71. Equivalências e gatilhos conforme interpretação funcional aprovada pelo usuário, sujeita à conferência jurídica."}
	ps := append([]Pendencia(nil), r.Pendencias...)
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].Tipo < ps[j].Tipo })
	for _, p := range ps {
		m.Itens[6].Linhas = append(m.Itens[6].Linhas, "Pendência: "+p.Mensagem)
	}
	return m
}

func ids(fs []Transgressao) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.ID
	}
	return out
}

func dataOu(f Transgressao) string {
	if f.Data.Zero() {
		return "data não informada"
	}
	return f.Data.String()
}

func rotuloSancao(s Sancao) string {
	switch s {
	case Repreensao:
		return "repreensão"
	case Detencao:
		return "detenção"
	case Prisao:
		return "prisão"
	}
	return "—"
}
