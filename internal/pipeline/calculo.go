package pipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

const perguntaCalculo = "calculo"

// SancaoCalculada é a punição proposta para uma imputação procedente
// principal, com a dosimetria e o que a sustenta.
type SancaoCalculada struct {
	ImputacaoID string
	Natureza    engine.Natureza
	Sancao      engine.Sancao
	// DiasPropostos pelo motor (V4 §6.3); Dias é o valor confirmado pelo
	// usuário (0 na repreensão e antes da confirmação).
	DiasPropostos int
	Dias          int
	Dosimetria    engine.ResultadoDosimetria
	Reincidencia  engine.ResultadoReincidencia
	// Circunstâncias reconhecidas pelo usuário na confirmação do resultado.
	Atenuantes, Agravantes, Justificacoes []engine.Circunstancia
	// Secundarias: conexas absorvidas como agravante, sem sanção própria.
	Secundarias []string
}

// CalculoOficial é o resultado do motor determinístico para o PADS (etapa 10
// do plano §5.4). A IA nunca calcula: recebe estes valores como definitivos.
type CalculoOficial struct {
	Conexao   engine.ResultadoConexao
	Sancoes   []SancaoCalculada
	Entrada   engine.EntradaPontuacao
	Pontuacao engine.ResultadoPontuacao
	Memoria   engine.Memoria
	// Servicos: justificativa dos relevantes serviços (art. 38, II), se houver.
	Servicos string
	// Pendencias do motor, em texto (mostradas ao usuário e levadas à minuta
	// como ressalva ou [A PREENCHER]).
	Pendencias []string
	// Confirmado: o usuário conferiu a memória e os dias.
	Confirmado bool
}

// Sancao devolve a sanção calculada de uma imputação principal.
func (c *CalculoOficial) Sancao(imputacaoID string) (SancaoCalculada, bool) {
	if c == nil {
		return SancaoCalculada{}, false
	}
	for _, s := range c.Sancoes {
		if s.ImputacaoID == imputacaoID {
			return s, true
		}
	}
	return SancaoCalculada{}, false
}

var rotuloSancao = map[engine.Sancao]string{engine.Repreensao: "repreensão", engine.Detencao: "detenção", engine.Prisao: "prisão"}
var rotuloComportamento = map[engine.Comportamento]string{engine.Excepcional: "excepcional", engine.Otimo: "ótimo", engine.Bom: "bom", engine.Insuficiente: "insuficiente"}

// calcular é a etapa 10: aplica o motor determinístico às decisões
// confirmadas e mostra a memória de cálculo para o usuário conferir (V4
// §§8.8, 9). Sem imputação procedente, não há o que calcular.
func (p *Pipeline) calcular(_ context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	e := d.IA
	if e == nil || e.Julgamento == nil {
		return workflow.Calculando, erros.Novo(erros.TarefaOrdemInvalida)
	}
	if len(e.Julgamento.Procedentes(nil)) == 0 {
		d.Calculo = nil
		c.Esquecer(perguntaCalculo)
		publicarPainel(c, d)
		return workflow.Redigindo, nil
	}
	calc := montarCalculo(c, e)
	if vs := e.Julgamento.VerificarCalculo(calc.Conexao.Sancoes, calc.Pontuacao); len(vs) > 0 {
		// Invariante quebrada: defeito do aplicativo, nunca seguir adiante.
		return workflow.Calculando, erros.Com(erros.InternoInesperado, fmt.Errorf("cálculo inconsistente: %s (%s)", vs[0].Mensagem, vs[0].Regra))
	}
	d.Calculo = calc
	resp, ok := c.RespostaCompleta(perguntaCalculo)
	if !ok {
		publicarPainel(c, d)
		c.Perguntar(perguntaCalculoPADS(calc, d.diasConfirmados))
		return workflow.AguardandoEsclarecimento, nil
	}
	d.diasConfirmados = map[string][2]int{}
	for i := range calc.Sancoes {
		s := &calc.Sancoes[i]
		if s.Sancao != engine.Repreensao {
			s.Dias, _ = strconv.Atoi(resp.Valor("dias_" + s.ImputacaoID))
			d.diasConfirmados[s.ImputacaoID] = [2]int{s.DiasPropostos, s.Dias}
		}
	}
	calc.Confirmado = true
	publicarPainel(c, d)
	return workflow.Redigindo, nil
}

// reconhecidas devolve as circunstâncias que o usuário marcou "sim" na
// confirmação do resultado da imputação (mesma ordem dos campos circ_N).
func reconhecidas(c *workflow.Contexto, imputacaoID string, an analise.AnaliseImputacao) (at, ag, ju []engine.Circunstancia) {
	resp, _ := c.RespostaCompleta("resultado:" + imputacaoID)
	k := 0
	for gi, grupo := range [][]analise.CircunstanciaProposta{an.Atenuantes, an.Agravantes, an.Justificacoes} {
		for _, cr := range grupo {
			k++
			if resp.Valor(fmt.Sprintf("circ_%d", k)) != "sim" {
				continue
			}
			ci := engine.Circunstancia{Artigo: cr.Artigo, Inciso: cr.Inciso, Alinea: cr.Alinea, Fundamento: cr.Fundamento}
			switch gi {
			case 0:
				at = append(at, ci)
			case 1:
				ag = append(ag, ci)
			default:
				ju = append(ju, ci)
			}
		}
	}
	return at, ag, ju
}

func lerHora(s string) civil.Hora {
	h, m, ok := strings.Cut(strings.TrimSpace(s), ":")
	hh, e1 := strconv.Atoi(h)
	mm, e2 := strconv.Atoi(m)
	if !ok || e1 != nil || e2 != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return civil.Hora{}
	}
	return civil.H(hh, mm)
}

// montarCalculo lê as respostas confirmadas e chama o motor.
func montarCalculo(c *workflow.Contexto, e *EstadoIA) *CalculoOficial {
	calc := &CalculoOficial{}
	// Conexão (V4 §5): um único grupo quando o usuário declarou conexas.
	grupos, conf := map[string]string{}, map[string]string{}
	if r, ok := c.RespostaCompleta(perguntaConexao); ok && r.Valor("conexas") == "sim" {
		for _, p := range e.Julgamento.Procedentes(nil) {
			grupos[p.ID] = "g"
		}
		conf["g"] = r.Valor("principal")
	}
	calc.Conexao = engine.AplicarConexao(e.Julgamento.Procedentes(grupos), conf)
	pend := func(ps []engine.Pendencia) {
		for _, x := range ps {
			calc.Pendencias = append(calc.Pendencias, x.Mensagem)
		}
	}
	pend(calc.Conexao.Pendencias)

	// Comportamento e pontuação atuais (anteriores à punição: V4 §6.2).
	comp, _ := c.RespostaCompleta(perguntaComportamento)
	var compAnterior *engine.Comportamento
	var pontos *int
	if v := engine.Comportamento(comp.Valor("comportamento")); v != "" {
		compAnterior = &v
	}
	if n, err := strconv.Atoi(strings.TrimSpace(comp.Valor("pontos"))); err == nil {
		pontos = &n
	}
	var servicos *bool
	if r, ok := c.RespostaCompleta(perguntaServicos); ok {
		v := r.Valor("relevantes") == "sim"
		servicos = &v
		if v {
			calc.Servicos = strings.TrimSpace(r.Valor("justificativa"))
		}
	}

	for _, pr := range calc.Conexao.Principais {
		s := SancaoCalculada{ImputacaoID: pr.ID, Natureza: pr.Natureza, Sancao: pr.Natureza.Sancao(), Secundarias: calc.Conexao.Secundarias[pr.ID]}
		s.Atenuantes, s.Agravantes, s.Justificacoes = reconhecidas(c, pr.ID, e.Analises[pr.ID])
		// Reincidência (art. 39, III): mesma natureza da principal.
		if r, ok := c.RespostaCompleta("reincidencia:" + string(pr.Natureza)); ok && r.Valor("possui") == "sim" {
			ant := engine.PunicaoAnterior{ID: "punição anterior", Natureza: pr.Natureza}
			if dt := lerData(r.Valor("termino")); !dt.Zero() {
				ant.Termino = &civil.Instante{Data: dt, Hora: lerHora(r.Valor("hora"))}
			}
			s.Reincidencia = engine.Reincidencia(pr.Natureza, pr.Fato, []engine.PunicaoAnterior{ant}, civil.SemEscolha29Fev)
			pend(s.Reincidencia.Pendencias)
		}
		in := engine.EntradaDosimetria{Sancao: s.Sancao, Atenuantes: s.Atenuantes, Agravantes: s.Agravantes, Justificacoes: s.Justificacoes,
			Reincidencia: s.Reincidencia.Reconhecida, ConexasSecundarias: len(s.Secundarias),
			ServicosRelevantes: servicos, JustificativaServicos: calc.Servicos}
		if compAnterior != nil {
			in.ComportamentoAnterior = *compAnterior
		}
		s.Dosimetria = engine.Dosimetria(in)
		s.DiasPropostos = s.Dosimetria.Dias
		pend(s.Dosimetria.Pendencias)
		if len(s.Justificacoes) > 0 {
			calc.Pendencias = append(calc.Pendencias, fmt.Sprintf("Na %s foi reconhecida causa de justificação (art. 37), que exclui a transgressão, mas o resultado confirmado é procedente. Confira.", pr.ID))
		}
		calc.Sancoes = append(calc.Sancoes, s)
	}

	// Comportamento proposto (V4 §§8, 15.4): histórico reconhecido + atuais.
	calc.Entrada = engine.EntradaPontuacao{Comportamento: compAnterior, Pontos: pontos, Sancoes: calc.Conexao.Sancoes}
	excepcional := compAnterior != nil && *compAnterior == engine.Excepcional
	if h, ok := c.RespostaCompleta(perguntaHistorico); ok && h.Valor("tem") == "sim" {
		for k := 1; k <= maxHistorico; k++ {
			nat := engine.Natureza(h.Valor(fmt.Sprintf("h%d_natureza", k)))
			if !nat.Valida() {
				continue
			}
			calc.Entrada.Fatos = append(calc.Entrada.Fatos, engine.Transgressao{ID: fmt.Sprintf("anterior %d", k), Natureza: nat,
				Data: lerData(h.Valor(fmt.Sprintf("h%d_data", k))), Origem: engine.Historico, PeriodoExcepcionalAtual: excepcional})
		}
	}
	for _, pr := range calc.Conexao.Principais {
		calc.Entrada.Fatos = append(calc.Entrada.Fatos, engine.Transgressao{ID: pr.ID, Natureza: pr.Natureza, Data: pr.Fato.Data,
			Origem: engine.Atual, PeriodoExcepcionalAtual: excepcional})
	}
	calc.Pontuacao = engine.Pontuacao(calc.Entrada)
	pend(calc.Pontuacao.Pendencias)
	calc.Memoria = engine.MontarMemoria(calc.Entrada, calc.Pontuacao, calc.Conexao)
	return calc
}

// linhasSancao descreve a dosimetria de uma sanção para o usuário.
func linhasSancao(s SancaoCalculada) []string {
	ls := []string{fmt.Sprintf("Natureza %s: %s (RDPM, art. 58, II).", rotuloNatureza[string(s.Natureza)], rotuloSancao[s.Sancao])}
	if len(s.Secundarias) > 0 {
		ls = append(ls, fmt.Sprintf("Principal das conexas %s, que entram como agravante (art. 39, II) e não recebem punição própria.", strings.Join(s.Secundarias, ", ")))
	}
	lista := func(xs []string) string {
		if len(xs) == 0 {
			return "nenhuma"
		}
		return strings.Join(xs, "; ")
	}
	ls = append(ls, "Atenuantes contadas: "+lista(s.Dosimetria.AtenuantesContadas)+".", "Agravantes contadas: "+lista(s.Dosimetria.AgravantesContadas)+".")
	switch {
	case s.Reincidencia.Reconhecida:
		ls = append(ls, "Reincidência reconhecida (art. 39, III): punição anterior da mesma natureza, com término dentro do prazo.")
	case len(s.Reincidencia.Descartadas) > 0:
		ls = append(ls, "Reincidência não reconhecida: "+s.Reincidencia.Descartadas[0].Motivo+".")
	}
	ls = append(ls, "Regra aplicada: "+s.Dosimetria.Regra+".")
	return ls
}

func maxDias(s engine.Sancao) int {
	if s == engine.Detencao {
		return engine.LimiteDiasDetencao
	}
	return 30
}

// perguntaCalculoPADS mostra a memória de cálculo e pede a confirmação dos
// dias (V4 §6.3: a regra de dias é uma proposta, sujeita à revisão).
//
// anteriores guarda, por imputação, a proposta e os dias da última
// confirmação: o ajuste do usuário só é reaproveitado se a proposta do motor
// continua a mesma; se o cálculo mudou, vale a nova proposta.
func perguntaCalculoPADS(calc *CalculoOficial, anteriores map[string][2]int) workflow.Pergunta {
	pg := workflow.Pergunta{ID: perguntaCalculo, Titulo: "Confira a dosimetria e o comportamento proposto", Revisavel: true,
		Contexto: "O cálculo é feito pelo aplicativo, não pela IA, a partir das suas respostas. Confira a memória de cálculo. " +
			"A quantidade de dias é uma proposta: ajuste, se for o caso. Para mudar um dado de entrada, use \"Alterar\" em \"Suas respostas\".",
		Opcoes: []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}}}
	for _, s := range calc.Sancoes {
		g := workflow.GrupoCampos{ID: "s_" + s.ImputacaoID, Titulo: fmt.Sprintf("%s — %s", s.ImputacaoID, rotuloSancao[s.Sancao]), Descricao: linhasSancao(s)}
		pg.Grupos = append(pg.Grupos, g)
		if s.Sancao == engine.Repreensao {
			continue
		}
		id := "dias_" + s.ImputacaoID
		valor := s.DiasPropostos
		if a, ok := anteriores[s.ImputacaoID]; ok && a[0] == s.DiasPropostos {
			valor = a[1]
		}
		pg.Campos = append(pg.Campos, workflow.Campo{ID: id, Rotulo: "Dias de " + rotuloSancao[s.Sancao], Tipo: workflow.CampoNumero, Obrigatorio: true,
			Min: 1, Max: maxDias(s.Sancao), Grupo: g.ID, Valor: strconv.Itoa(valor),
			Ajuda: fmt.Sprintf("Proposta: %d dia(s).", s.DiasPropostos)})
	}
	for i, it := range calc.Memoria.Itens {
		pg.Grupos = append(pg.Grupos, workflow.GrupoCampos{ID: fmt.Sprintf("m%d", i+1), Titulo: fmt.Sprintf("%d. %s", i+1, it.Titulo), Descricao: it.Linhas})
	}
	if len(calc.Pendencias) > 0 {
		pg.Grupos = append(pg.Grupos, workflow.GrupoCampos{ID: "pend", Titulo: "Pendências do cálculo", Alertas: calc.Pendencias,
			Descricao: []string{"O que ficar pendente aparece na minuta como [A PREENCHER] ou como ressalva."}})
	}
	sancoes := calc.Sancoes
	divergente := false
	for _, x := range calc.Pontuacao.Pendencias {
		divergente = divergente || x.Tipo == engine.PendDivergenciaComportamento
	}
	pg.Conferir = func(v map[string]string) error {
		if divergente {
			return fmt.Errorf("o comportamento e a pontuação informados não combinam; corrija em \"Suas respostas\" antes de confirmar")
		}
		for _, s := range sancoes {
			if s.Sancao == engine.Repreensao {
				continue
			}
			n, err := strconv.Atoi(strings.TrimSpace(v["dias_"+s.ImputacaoID]))
			if err != nil {
				return fmt.Errorf("informe os dias de %s da %s", rotuloSancao[s.Sancao], s.ImputacaoID)
			}
			if err := engine.ValidarDias(s.Sancao, n); err != nil {
				return fmt.Errorf("%s: %v", s.ImputacaoID, err)
			}
		}
		return nil
	}
	pont := calc.Pontuacao
	pg.Resumir = func(r workflow.Resposta) string {
		var ps []string
		for _, s := range sancoes {
			if s.Sancao == engine.Repreensao {
				ps = append(ps, s.ImputacaoID+": repreensão")
				continue
			}
			ps = append(ps, fmt.Sprintf("%s: %s dia(s) de %s", s.ImputacaoID, r.Valor("dias_"+s.ImputacaoID), rotuloSancao[s.Sancao]))
		}
		if pont.Concluido {
			ps = append(ps, fmt.Sprintf("%d pontos, %s", pont.PontosFinais, rotuloComportamento[pont.ComportamentoFinal]))
		} else {
			ps = append(ps, "comportamento a preencher")
		}
		return strings.Join(ps, "; ")
	}
	return pg
}

// secaoCalculo mostra o cálculo no painel (memória do §8.8).
func secaoCalculo(d *Dados) (workflow.SecaoPainel, bool) {
	calc := d.Calculo
	s := workflow.SecaoPainel{Titulo: "Cálculo oficial"}
	if calc == nil {
		return s, false
	}
	sit := "aguardando confirmação"
	if calc.Confirmado {
		sit = "confirmado"
	}
	for _, x := range calc.Sancoes {
		titulo := fmt.Sprintf("%s — %s", x.ImputacaoID, rotuloSancao[x.Sancao])
		if x.Sancao != engine.Repreensao {
			dias := x.DiasPropostos
			if calc.Confirmado {
				dias = x.Dias
			}
			titulo += fmt.Sprintf(", %d dia(s)", dias)
		}
		s.Itens = append(s.Itens, workflow.ItemPainel{Titulo: titulo, Situacao: sit, Linhas: linhasSancao(x)})
	}
	for i, it := range calc.Memoria.Itens {
		s.Itens = append(s.Itens, workflow.ItemPainel{Titulo: fmt.Sprintf("%d. %s", i+1, it.Titulo), Linhas: it.Linhas})
	}
	for _, l := range calc.Pendencias {
		s.Itens = append(s.Itens, workflow.ItemPainel{Titulo: l, Situacao: "pendência"})
	}
	return s, true
}
