package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

const (
	perguntaMidias        = "midias"
	perguntaConexao       = "conexao"
	perguntaComportamento = "comportamento"
	perguntaHistorico     = "historico"
	perguntaServicos      = "servicos"
	maxHistorico          = 5
)

var rotuloNatureza = map[string]string{"LEVE": "leve", "MEDIA": "média", "GRAVE": "grave"}

// analisar cobre as etapas 5 a 9: extração por peça, escopo (SR),
// mídias, análise por alvo e confirmações do usuário.
func (p *Pipeline) analisar(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	d.Fontes, d.Calculo = nil, nil // refeitos depois da análise (as respostas podem ter mudado)
	// A autorização vale para o provedor e o modelo atuais: se mudaram
	// depois dela (ex.: ao corrigir uma falha), pergunta de novo.
	if aguardar, err := p.autorizarEnvio(c, d); err != nil {
		return workflow.Analisando, err
	} else if aguardar {
		return workflow.AguardandoEsclarecimento, nil
	}
	if err := p.garantirMotor(ctx, c, d); err != nil {
		if d.IA != nil {
			d.IA.motorFalhou(err)
		}
		return workflow.Analisando, err
	}
	e := d.IA
	fluxo := policy.FluxoPADS
	if c.Tipo == workflow.TipoSR {
		fluxo = policy.FluxoSR
	}
	x := e.executor(fluxo)
	if err := p.extrairPecas(ctx, c, d, x); err != nil {
		e.motorFalhou(err)
		return workflow.Analisando, err
	}
	if c.Tipo == workflow.TipoSR {
		if pergunta := autorizacoes(c, d); pergunta {
			return workflow.AguardandoEsclarecimento, nil
		}
		if err := p.extrairPecas(ctx, c, d, x); err != nil { // fatos novos autorizados
			e.motorFalhou(err)
			return workflow.Analisando, err
		}
	}
	if pergunta := midias(c, d); pergunta {
		return workflow.AguardandoEsclarecimento, nil
	}
	if err := analisarAlvos(ctx, c, d, x); err != nil {
		e.motorFalhou(err)
		return workflow.Analisando, err
	}
	publicarPainel(c, d)
	var pendente bool
	if c.Tipo == workflow.TipoSR {
		pendente = confirmacoesSR(c, d)
	} else {
		pendente = confirmacoesPADS(c, d)
	}
	publicarPainel(c, d)
	if pendente {
		return workflow.AguardandoEsclarecimento, nil
	}
	return workflow.ConsultandoFontes, nil
}

// alvos devolve as imputações congeladas (PADS) ou os fatos admitidos (SR).
func alvos(d *Dados) ([]analise.AlvoExtracao, policy.Secao) {
	e := d.IA
	var as []analise.AlvoExtracao
	if e.Conjunto != nil {
		for _, im := range e.Conjunto.Imputacoes() {
			as = append(as, analise.AlvoExtracao{ID: im.ID, Descricao: strings.TrimSpace(im.Texto + " (" + im.Dispositivo + ")")})
		}
		return as, policy.Secao{Nome: "IMPUTACOES_CONGELADAS", Dados: as}
	}
	for _, id := range e.Objeto.FatosAdmitidos() {
		f, _ := e.Objeto.Fato(id)
		as = append(as, analise.AlvoExtracao{ID: id, Descricao: f.Descricao})
	}
	var sind []analise.AlvoExtracao
	for _, s := range e.Objeto.Sindicados() {
		sind = append(sind, analise.AlvoExtracao{ID: s, Descricao: e.nomes[s]})
	}
	return as, policy.Secao{Nome: "OBJETO_SR", Dados: map[string]any{"fatos": as, "sindicados": sind}}
}

func (p *Pipeline) extrairPecas(ctx context.Context, c *workflow.Contexto, d *Dados, x *analise.Executor) error {
	e := d.IA
	as, secao := alvos(d)
	var fila []int
	for i, pc := range d.Pecas {
		if _, feito := e.Extracoes[pc.ID]; !feito && analise.PecaComProva(pc) {
			fila = append(fila, i)
		}
	}
	for k, i := range fila {
		pc := d.Pecas[i]
		c.Progresso(k, len(fila), "peças lidas")
		c.Atividade("Lendo " + pc.Titulo)
		flags := map[string]bool{}
		for _, m := range d.Midias {
			if m.Peca == pc.ID {
				flags[catalog.CondMidiaAusente] = true
			}
		}
		ex, err := analise.ExtrairPeca(ctx, x, d.Autos, pc, secao, as, flags, &e.reg)
		if err != nil {
			return err
		}
		e.Extracoes[pc.ID] = ex
		e.Pontos = append(e.Pontos, ex.Pontos...)
		for _, a := range ex.Achados {
			if a.Tipo == analise.AchadoCondutaDiversa {
				e.CondutasDiversas = append(e.CondutasDiversas, a) // I-05: registro interno
			}
		}
		publicarPainel(c, d)
	}
	if len(fila) > 0 {
		c.Progresso(len(fila), len(fila), "peças lidas")
	}
	return nil
}

// autorizacoes: fatos e militares novos achados na instrução viram
// Autorizacao PENDENTE e pergunta (SR-ESC-001/002; SR19, SR20, SR35).
func autorizacoes(c *workflow.Contexto, d *Dados) bool {
	e := d.IA
	nf, nm := 0, 0
	for _, a := range e.Objeto.Autorizacoes() {
		if a.Tipo == sr.FatoNovo {
			nf++
		} else {
			nm++
		}
	}
	for _, pc := range d.Pecas {
		ex, ok := e.Extracoes[pc.ID]
		if !ok {
			continue
		}
		for _, a := range ex.Achados {
			if a.Tipo != analise.AchadoFatoNovo && a.Tipo != analise.AchadoMilitarNovo {
				continue
			}
			chave := fmt.Sprintf("%s|%d|%d", a.Tipo, a.Ponto.Pagina, a.Ponto.Inicio)
			if _, feito := e.autorizacoesFeitas[chave]; feito {
				continue
			}
			evid := []string{fmt.Sprintf("pág. %d", a.Ponto.Pagina)}
			var aut sr.Autorizacao
			var err error
			if a.Tipo == analise.AchadoFatoNovo {
				nf++
				aut, err = e.Objeto.ProporFatoNovo(sr.Fato{ID: fmt.Sprintf("F-N%d", nf), Descricao: a.Descricao}, a.Descricao, evid)
			} else {
				nm++
				id := fmt.Sprintf("S-N%d", nm)
				if err = e.Objeto.RegistrarPessoa(sr.Pessoa{ID: id, Tipo: sr.Militar}); err == nil {
					e.nomes[id] = a.Descricao
					aut, err = e.Objeto.ProporMilitarNovo(id, a.Descricao, evid)
				}
			}
			if err != nil {
				e.Pendencias = append(e.Pendencias, "Não foi possível registrar um item fora do objeto: "+a.Descricao)
				continue
			}
			e.autorizacoesFeitas[chave] = aut.ID
			e.dag.Depende("analise", "autorizacao:"+aut.ID)
		}
	}
	pontoDe := map[string]analise.Achado{}
	for _, ex := range e.Extracoes {
		for _, a := range ex.Achados {
			if id, ok := e.autorizacoesFeitas[fmt.Sprintf("%s|%d|%d", a.Tipo, a.Ponto.Pagina, a.Ponto.Inicio)]; ok {
				pontoDe[id] = a
			}
		}
	}
	for _, a := range e.Objeto.Autorizacoes() {
		if a.Estado != sr.Pendente {
			continue
		}
		id := "autorizacao:" + a.ID
		if r, ok := c.Resposta(id); ok {
			if err := e.Objeto.Decidir(a.ID, r == "autorizar"); err == nil && r == "autorizar" && a.Tipo == sr.FatoNovo {
				// Fato novo autorizado: a peça onde apareceu é relida com ele no objeto.
				delete(e.Extracoes, pontoDe[a.ID].Ponto.Peca)
				e.Pontos = semPontosDaPeca(e.Pontos, pontoDe[a.ID].Ponto.Peca)
			}
			continue
		}
		titulo, ctxt := "Fato fora da Portaria", "Os autos trazem um fato que não consta da Portaria de instauração. Ele só entra na análise (III) e na solução (IV) se você autorizar; a Portaria não é alterada."
		if a.Tipo == sr.MilitarNovo {
			titulo, ctxt = "Militar não indicado na Portaria", "Os autos indicam a participação de um militar que não é sindicado (pode ser de outra OPM). Ele só passa a sindicado se você autorizar; a Portaria não é alterada."
		}
		ach := pontoDe[a.ID]
		c.Perguntar(workflow.Pergunta{ID: id, Titulo: titulo, Contexto: ctxt,
			Grupos: []workflow.GrupoCampos{{ID: a.ID, Titulo: a.Resumo, Fontes: []workflow.Fonte{fonteDe(ach.Ponto, ach.Ponto.Titulo)}}},
			Opcoes: []workflow.Opcao{{ID: "autorizar", Rotulo: "Autorizar a inclusão"}, {ID: "recusar", Rotulo: "Não incluir"}}})
		return true
	}
	return false
}

func semPontosDaPeca(ps []analise.Ponto, peca string) []analise.Ponto {
	var out []analise.Ponto
	for _, p := range ps {
		if p.Peca != peca || p.Quem == quemUsuario {
			out = append(out, p)
		}
	}
	return out
}

type midiaRef struct {
	Peca, Titulo, Termo string
	Pagina, Inicio, Fim int
}

func listarMidias(d *Dados) []midiaRef {
	vistos := map[string]bool{}
	var out []midiaRef
	titulo := map[string]string{}
	for _, pc := range d.Pecas {
		titulo[pc.ID] = pc.Titulo
	}
	add := func(m midiaRef) {
		k := fmt.Sprintf("%d|%d", m.Pagina, m.Inicio/200)
		if !vistos[k] {
			vistos[k] = true
			out = append(out, m)
		}
	}
	for _, m := range d.Midias {
		add(midiaRef{Peca: m.Peca, Titulo: titulo[m.Peca], Termo: m.Termo, Pagina: m.Pagina, Inicio: m.Inicio, Fim: m.Fim})
	}
	for _, ex := range d.IA.Extracoes {
		for _, a := range ex.Achados {
			if a.Tipo == analise.AchadoMidia {
				add(midiaRef{Peca: a.Ponto.Peca, Titulo: a.Ponto.Titulo, Termo: a.Descricao, Pagina: a.Ponto.Pagina, Inicio: a.Ponto.Inicio, Fim: a.Ponto.Fim})
			}
		}
	}
	return out
}

// midias: mídia citada e não juntada → descrição/transcrição do usuário ou
// prosseguir com ressalva (A08–A11; TRV-PRV-004). Devolve true se perguntou.
func midias(c *workflow.Contexto, d *Dados) bool {
	e := d.IA
	ms := listarMidias(d)
	if len(ms) == 0 {
		return false
	}
	resp, ok := c.RespostaCompleta(perguntaMidias)
	if !ok {
		ant, _ := c.Anterior(perguntaMidias)
		pg := workflow.Pergunta{ID: perguntaMidias, Titulo: "Mídias citadas e não incluídas no PDF", Revisavel: true,
			Contexto: "Os autos mencionam vídeo, áudio ou outro arquivo que não está no PDF. O aplicativo não acessa nem baixa mídia. " +
				"Se você conhece o conteúdo, descreva ou transcreva; sem descrição, a análise usa só a prova restante e a limitação é registrada.",
			Opcoes: []workflow.Opcao{{ID: "continuar", Rotulo: "Continuar a análise", Validar: true}}}
		for k, m := range ms {
			id := fmt.Sprintf("midia_%d", k+1)
			pg.Grupos = append(pg.Grupos, workflow.GrupoCampos{ID: id, Titulo: fmt.Sprintf("%s — pág. %d", m.Titulo, m.Pagina),
				Descricao: []string{"Menção: " + m.Termo},
				Fontes:    []workflow.Fonte{{Rotulo: m.Titulo, Pagina: m.Pagina, Inicio: m.Inicio, Fim: m.Fim}}})
			pg.Campos = append(pg.Campos, workflow.Campo{ID: id, Rotulo: "Descrição ou transcrição (opcional)", Tipo: workflow.CampoTextoLongo,
				Grupo: id, Valor: ant.Valor(id)})
		}
		pg.Resumir = func(r workflow.Resposta) string {
			n := 0
			for _, v := range r.Valores {
				if strings.TrimSpace(v) != "" {
					n++
				}
			}
			return fmt.Sprintf("%d de %d mídia(s) descrita(s)", n, len(ms))
		}
		c.Perguntar(pg)
		return true
	}
	if !e.midiasAplicadas {
		e.midiasAplicadas = true
		for k, m := range ms {
			desc := strings.TrimSpace(resp.Valor(fmt.Sprintf("midia_%d", k+1)))
			if desc == "" {
				e.Pendencias = append(e.Pendencias, fmt.Sprintf(prefixoPendMidia+"%s (pág. %d) sem descrição: a análise considera só a prova restante.", m.Titulo, m.Pagina))
				continue
			}
			e.Pontos = append(e.Pontos, analise.Ponto{ID: e.reg.Novo(), Peca: m.Peca, Titulo: m.Titulo, Pagina: m.Pagina, Inicio: m.Inicio, Fim: m.Fim,
				Quem: quemUsuario, Trecho: "Descrição fornecida pelo usuário: " + desc, Sentido: "NEUTRO"})
		}
	}
	return false
}

func pontosDe(e *EstadoIA, alvo string) []analise.Ponto {
	var out []analise.Ponto
	for _, p := range e.Pontos {
		if p.Alvo == alvo || p.Quem == quemUsuario {
			out = append(out, p)
		}
	}
	return out
}

func analisarAlvos(ctx context.Context, c *workflow.Contexto, d *Dados, x *analise.Executor) error {
	e := d.IA
	as, secao := alvos(d)
	flags := map[string]bool{catalog.CondMidiaAusente: len(listarMidias(d)) > 0}
	if e.Conjunto != nil {
		for k, a := range as {
			if _, feito := e.Analises[a.ID]; feito {
				continue
			}
			c.Progresso(k, len(as), "imputações analisadas")
			c.Atividade("Analisando a imputação " + a.ID)
			r, err := analise.AnalisarImputacao(ctx, x, a, secao, pontosDe(e, a.ID), flags)
			if err != nil {
				return err
			}
			e.Analises[a.ID] = r
			publicarPainel(c, d)
		}
		return nil
	}
	total, k := len(as)*len(e.Objeto.Sindicados()), 0
	for _, f := range as {
		for _, s := range e.Objeto.Sindicados() {
			k++
			chave := f.ID + ":" + s
			if _, feito := e.AnalisesSR[chave]; feito {
				continue
			}
			c.Progresso(k-1, total, "análises de fato e sindicado")
			c.Atividade(fmt.Sprintf("Analisando o fato %s quanto a %s", f.ID, e.nomes[s]))
			r, err := analise.AnalisarFatoPessoa(ctx, x, f, analise.AlvoExtracao{ID: s, Descricao: e.nomes[s]}, secao, pontosDe(e, f.ID))
			if err != nil {
				return err
			}
			e.AnalisesSR[chave] = r
			publicarPainel(c, d)
		}
	}
	return nil
}

func fontesDosFundamentos(e *EstadoIA, fs []analise.Fundamento) []workflow.Fonte {
	por := map[string]analise.Ponto{}
	for _, p := range e.Pontos {
		por[p.ID] = p
	}
	var out []workflow.Fonte
	vistos := map[string]bool{}
	for _, f := range fs {
		for _, id := range f.Evid {
			if p, ok := por[id]; ok && !vistos[id] && p.Quem != quemUsuario {
				vistos[id] = true
				out = append(out, fonteDe(p, fmt.Sprintf("%s (%s)", p.Titulo, id)))
			}
		}
	}
	return out
}

func linhasFundamentos(rotulo string, fs []analise.Fundamento) []string {
	var out []string
	for _, f := range fs {
		out = append(out, fmt.Sprintf("%s: %s [%s]", rotulo, f.Texto, strings.Join(f.Evid, ", ")))
	}
	return out
}

// ---- Etapa 9: confirmações do PADS ----

func confirmacoesPADS(c *workflow.Contexto, d *Dados) bool {
	e := d.IA
	for _, im := range e.Conjunto.Imputacoes() {
		id := "resultado:" + im.ID
		resp, ok := c.RespostaCompleta(id)
		if !ok {
			c.Perguntar(perguntaResultado(e, im, e.Analises[im.ID], c))
			return true
		}
		_ = e.Julgamento.Registrar(pads.ResultadoImputacao{ImputacaoID: im.ID, Resultado: pads.Resultado(resp.Valor("resultado")), ConfirmadoPorUsuario: true})
	}
	procedentes := e.Julgamento.Procedentes(nil)
	if len(procedentes) == 0 {
		return false // improcedência: sem cálculo de comportamento nem sanção
	}
	if len(procedentes) >= 2 {
		if _, ok := c.RespostaCompleta(perguntaConexao); !ok {
			c.Perguntar(perguntaConexaoPADS(procedentes))
			return true
		}
	}
	if _, ok := c.RespostaCompleta(perguntaComportamento); !ok {
		c.Perguntar(perguntaComportamentoPADS(c))
		return true
	}
	comp, _ := c.RespostaCompleta(perguntaComportamento)
	if comp.Valor("comportamento") != string(engine.Insuficiente) {
		if _, ok := c.RespostaCompleta(perguntaHistorico); !ok {
			c.Perguntar(perguntaHistoricoPADS(c))
			return true
		}
	}
	for _, nat := range naturezasPrincipais(c, e, procedentes) {
		id := "reincidencia:" + string(nat)
		if _, ok := c.RespostaCompleta(id); !ok {
			n := nat
			if engine.ProximaPerguntaReincidencia(&n, nil, false) == engine.PerguntarSeHaAnterior {
				c.Perguntar(perguntaReincidencia(nat, c))
				return true
			}
		}
	}
	if _, ok := c.RespostaCompleta(perguntaServicos); !ok {
		c.Perguntar(perguntaServicosPADS(c))
		return true
	}
	return false
}

// naturezasPrincipais: nas conexas só a natureza da principal conta para a
// reincidência (REI-CNX-01).
func naturezasPrincipais(c *workflow.Contexto, e *EstadoIA, procedentes []engine.ImputacaoProcedente) []engine.Natureza {
	grupos := map[string]string{}
	conf := map[string]string{}
	if r, ok := c.RespostaCompleta(perguntaConexao); ok && r.Valor("conexas") == "sim" {
		for _, p := range procedentes {
			grupos[p.ID] = "g"
		}
		conf["g"] = r.Valor("principal")
	}
	conx := engine.AplicarConexao(e.Julgamento.Procedentes(grupos), conf)
	vistos := map[engine.Natureza]bool{}
	var out []engine.Natureza
	for _, p := range conx.Principais {
		if !vistos[p.Natureza] {
			vistos[p.Natureza] = true
			out = append(out, p.Natureza)
		}
	}
	return out
}

func valorAnterior(c *workflow.Contexto, pergunta, campo, padrao string) string {
	if a, ok := c.Anterior(pergunta); ok {
		if v, ok := a.Valores[campo]; ok {
			return v
		}
	}
	return padrao
}

func perguntaResultado(e *EstadoIA, im pads.Imputacao, an analise.AnaliseImputacao, c *workflow.Contexto) workflow.Pergunta {
	id := "resultado:" + im.ID
	g := workflow.GrupoCampos{ID: im.ID, Titulo: fmt.Sprintf("%s — %s", im.ID, im.Dispositivo),
		Descricao: []string{im.Texto}, Alertas: an.Pendencias}
	if an.Resultado != "" {
		g.Descricao = append(g.Descricao, "Proposta da IA: "+strings.ToLower(an.Resultado)+".")
	}
	g.Descricao = append(g.Descricao, linhasFundamentos("Fundamento", an.Fundamentos)...)
	g.Descricao = append(g.Descricao, linhasFundamentos("Tese da defesa", an.TesesDefesa)...)
	for _, l := range an.Lacunas {
		g.Alertas = append(g.Alertas, "Lacuna: "+l)
	}
	g.Fontes = fontesDosFundamentos(e, append(append([]analise.Fundamento{}, an.Fundamentos...), an.TesesDefesa...))
	pg := workflow.Pergunta{ID: id, Titulo: "Resultado da imputação " + im.ID, Revisavel: true,
		Contexto: "Confira a proposta e as fontes. O resultado é seu: a IA apenas sugere.",
		Grupos:   []workflow.GrupoCampos{g},
		Opcoes:   []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}}}
	pg.Campos = append(pg.Campos, workflow.Campo{ID: "resultado", Rotulo: "Resultado", Tipo: workflow.CampoEscolha, Obrigatorio: true, Grupo: im.ID,
		Opcoes: []workflow.OpcaoVisao{{ID: string(pads.Procedente), Rotulo: "Procedente"}, {ID: string(pads.Improcedente), Rotulo: "Improcedente"}},
		Valor:  valorAnterior(c, id, "resultado", an.Resultado)})
	k := 0
	for _, grupo := range []struct {
		nome string
		cs   []analise.CircunstanciaProposta
	}{{"Atenuante", an.Atenuantes}, {"Agravante", an.Agravantes}, {"Causa de justificação", an.Justificacoes}} {
		for _, cr := range grupo.cs {
			k++
			cid := fmt.Sprintf("circ_%d", k)
			disp := fmt.Sprintf("art. %d, %s", cr.Artigo, romano(cr.Inciso))
			if cr.Alinea != "" {
				disp += ", alínea " + cr.Alinea
			}
			pg.Campos = append(pg.Campos, workflow.Campo{ID: cid, Rotulo: fmt.Sprintf("%s sugerida (%s): reconhecer?", grupo.nome, disp),
				Tipo: workflow.CampoSimNao, Grupo: im.ID, Ajuda: cr.Fundamento, Valor: valorAnterior(c, id, cid, "nao"),
				MostrarSe: &workflow.Condicao{Campo: "resultado", Valor: string(pads.Procedente)}})
		}
	}
	pg.Resumir = func(r workflow.Resposta) string { return strings.ToLower(r.Valor("resultado")) }
	return pg
}

func romano(n int) string {
	vals := []int{10, 9, 5, 4, 1}
	sim := []string{"X", "IX", "V", "IV", "I"}
	s := ""
	for i, v := range vals {
		for n >= v {
			s += sim[i]
			n -= v
		}
	}
	return s
}

func perguntaConexaoPADS(ps []engine.ImputacaoProcedente) workflow.Pergunta {
	sug, empate := engine.SugerirPrincipal(ps)
	var ops []workflow.OpcaoVisao
	for _, p := range ps {
		ops = append(ops, workflow.OpcaoVisao{ID: p.ID, Rotulo: fmt.Sprintf("%s (%s)", p.ID, rotuloNatureza[string(p.Natureza)])})
	}
	ajuda := "Sugestão: a de maior gravidade."
	if empate {
		ajuda = "Há empate de gravidade: escolha a principal."
	}
	return workflow.Pergunta{ID: perguntaConexao, Titulo: "Conexão entre as imputações procedentes", Revisavel: true,
		Contexto: "Imputações conexas recebem uma única sanção (a da principal); as demais entram como agravante, sem punição própria.",
		Opcoes:   []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}},
		Campos: []workflow.Campo{
			{ID: "conexas", Rotulo: "As imputações procedentes são conexas?", Tipo: workflow.CampoSimNao, Obrigatorio: true},
			{ID: "principal", Rotulo: "Imputação principal", Tipo: workflow.CampoEscolha, Opcoes: ops, Obrigatorio: true, Valor: sug, Ajuda: ajuda,
				MostrarSe: &workflow.Condicao{Campo: "conexas", Valor: "sim"}},
		},
		Resumir: func(r workflow.Resposta) string {
			if r.Valor("conexas") == "sim" {
				return "conexas; principal " + r.Valor("principal")
			}
			return "não conexas"
		}}
}

// T08: pontuação e comportamento atuais são essenciais e quase nunca estão
// nos autos: pergunta objetiva (PADS-CMP-002).
func perguntaComportamentoPADS(c *workflow.Contexto) workflow.Pergunta {
	return workflow.Pergunta{ID: perguntaComportamento, Titulo: "Comportamento e pontuação atuais do acusado", Revisavel: true,
		Contexto: "Informe a situação atual, conforme a ficha do militar. O cálculo do comportamento proposto é feito pelo aplicativo, não pela IA.",
		Opcoes:   []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}},
		Campos: []workflow.Campo{
			{ID: "comportamento", Rotulo: "Comportamento atual", Tipo: workflow.CampoEscolha, Obrigatorio: true,
				Opcoes: []workflow.OpcaoVisao{{ID: string(engine.Excepcional), Rotulo: "Excepcional"}, {ID: string(engine.Otimo), Rotulo: "Ótimo"},
					{ID: string(engine.Bom), Rotulo: "Bom"}, {ID: string(engine.Insuficiente), Rotulo: "Insuficiente"}},
				Valor: valorAnterior(c, perguntaComportamento, "comportamento", "")},
			{ID: "pontos", Rotulo: "Pontuação atual", Tipo: workflow.CampoNumero, Obrigatorio: true, Min: -50, Max: 100,
				Valor: valorAnterior(c, perguntaComportamento, "pontos", "")},
		},
		Resumir: func(r workflow.Resposta) string {
			return strings.ToLower(r.Valor("comportamento")) + ", " + r.Valor("pontos") + " pontos"
		}}
}

func perguntaHistoricoPADS(c *workflow.Contexto) workflow.Pergunta {
	pg := workflow.Pergunta{ID: perguntaHistorico, Titulo: "Histórico disciplinar recente", Revisavel: true,
		Contexto: "Para conferir a regressão especial de comportamento (RDPM, art. 70), informe as transgressões anteriores reconhecidas em decisão disciplinar " +
			"nos 12 meses anteriores ao fato (ou no período atual de comportamento excepcional). Não inclua acusações pendentes.",
		Opcoes: []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}},
		Campos: []workflow.Campo{{ID: "tem", Rotulo: "Há transgressões anteriores nesse período?", Tipo: workflow.CampoSimNao, Obrigatorio: true,
			Valor: valorAnterior(c, perguntaHistorico, "tem", "")}}}
	sim := &workflow.Condicao{Campo: "tem", Valor: "sim"}
	for k := 1; k <= maxHistorico; k++ {
		g := fmt.Sprintf("h%d", k)
		pg.Grupos = append(pg.Grupos, workflow.GrupoCampos{ID: g, Titulo: fmt.Sprintf("Transgressão anterior %d", k)})
		pg.Campos = append(pg.Campos,
			workflow.Campo{ID: g + "_natureza", Rotulo: "Natureza", Tipo: workflow.CampoEscolha, Opcoes: opcoesNatureza, Grupo: g, Obrigatorio: k == 1,
				MostrarSe: sim, Valor: valorAnterior(c, perguntaHistorico, g+"_natureza", "")},
			workflow.Campo{ID: g + "_data", Rotulo: "Data do fato", Tipo: workflow.CampoData, Grupo: g, Obrigatorio: k == 1,
				MostrarSe: sim, Valor: valorAnterior(c, perguntaHistorico, g+"_data", "")})
	}
	pg.Resumir = func(r workflow.Resposta) string {
		if r.Valor("tem") != "sim" {
			return "nenhuma"
		}
		n := 0
		for k := 1; k <= maxHistorico; k++ {
			if r.Valor(fmt.Sprintf("h%d_natureza", k)) != "" {
				n++
			}
		}
		return fmt.Sprintf("%d transgressão(ões)", n)
	}
	return pg
}

// perguntaReincidencia (PADS-REI-009): só depois de fixada a natureza.
func perguntaReincidencia(nat engine.Natureza, c *workflow.Contexto) workflow.Pergunta {
	id := "reincidencia:" + string(nat)
	r := rotuloNatureza[string(nat)]
	return workflow.Pergunta{ID: id, Titulo: "Punição anterior de natureza " + r, Revisavel: true,
		Contexto: fmt.Sprintf("O acusado possui punição anterior de natureza %s? A reincidência depende da data de TÉRMINO do cumprimento da punição anterior.", r),
		Opcoes:   []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}},
		Campos: []workflow.Campo{
			{ID: "possui", Rotulo: "Possui punição anterior de natureza " + r + "?", Tipo: workflow.CampoSimNao, Obrigatorio: true,
				Valor: valorAnterior(c, id, "possui", "")},
			{ID: "termino", Rotulo: "Data de término do cumprimento", Tipo: workflow.CampoData, Ajuda: "DD/MM/AAAA. Sem a data, a reincidência não é reconhecida.",
				MostrarSe: &workflow.Condicao{Campo: "possui", Valor: "sim"}, Valor: valorAnterior(c, id, "termino", "")},
			{ID: "hora", Rotulo: "Hora do término (só se for no mesmo dia do fato)", Tipo: workflow.CampoTexto, Ajuda: "HH:MM",
				MostrarSe: &workflow.Condicao{Campo: "possui", Valor: "sim"}, Valor: valorAnterior(c, id, "hora", "")},
		},
		Resumir: func(rr workflow.Resposta) string {
			if rr.Valor("possui") != "sim" {
				return "não possui"
			}
			if rr.Valor("termino") == "" {
				return "possui; término não informado"
			}
			return "possui; término em " + rr.Valor("termino")
		}}
}

// perguntaServicosPADS (PADS-DOS-005; art. 38, II).
func perguntaServicosPADS(c *workflow.Contexto) workflow.Pergunta {
	return workflow.Pergunta{ID: perguntaServicos, Titulo: "Relevantes serviços prestados", Revisavel: true,
		Contexto: "O acusado prestou relevantes serviços (atenuante do art. 38, II)? Responda só com base em fato concreto.",
		Opcoes:   []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}},
		Campos: []workflow.Campo{
			{ID: "relevantes", Rotulo: "Prestou relevantes serviços?", Tipo: workflow.CampoSimNao, Obrigatorio: true, Valor: valorAnterior(c, perguntaServicos, "relevantes", "")},
			{ID: "justificativa", Rotulo: "Justificativa sucinta e factual", Tipo: workflow.CampoTextoLongo, Obrigatorio: true,
				MostrarSe: &workflow.Condicao{Campo: "relevantes", Valor: "sim"}, Valor: valorAnterior(c, perguntaServicos, "justificativa", "")},
		},
		Resumir: func(r workflow.Resposta) string {
			if r.Valor("relevantes") == "sim" {
				return "sim"
			}
			return "não"
		}}
}

// ---- Etapa 9: confirmações da SR ----

func confirmacoesSR(c *workflow.Contexto, d *Dados) bool {
	e := d.IA
	for _, f := range e.Objeto.FatosAdmitidos() {
		fato, _ := e.Objeto.Fato(f)
		for _, s := range e.Objeto.Sindicados() {
			id := "conclusao:" + f + ":" + s
			if _, ok := c.RespostaCompleta(id); ok {
				continue
			}
			an := e.AnalisesSR[f+":"+s]
			g := workflow.GrupoCampos{ID: f + s, Titulo: fmt.Sprintf("Fato %s — %s", f, e.nomes[s]), Descricao: []string{fato.Descricao}, Alertas: an.Pendencias}
			if an.Conclusao != "" {
				g.Descricao = append(g.Descricao, "Proposta da IA: "+rotuloConclusao[an.Conclusao]+".")
			}
			g.Descricao = append(g.Descricao, linhasFundamentos("Fundamento", an.Fundamentos)...)
			for _, l := range an.Lacunas {
				g.Alertas = append(g.Alertas, "Lacuna: "+l)
			}
			for _, l := range an.Diligencias {
				g.Alertas = append(g.Alertas, "Diligência sugerida: "+l)
			}
			g.Fontes = fontesDosFundamentos(e, an.Fundamentos)
			var ops []workflow.OpcaoVisao
			for _, k := range []string{"INDICIOS", "AUSENCIA", "INCONCLUSIVO"} {
				ops = append(ops, workflow.OpcaoVisao{ID: k, Rotulo: rotuloConclusao[k]})
			}
			// O título leva o sindicado: com dois sindicados no mesmo fato,
			// "Suas respostas" distingue as duas conclusões.
			titulo := "Conclusão sobre o fato " + f
			if nome := e.nomes[s]; nome != "" {
				titulo += ": " + nome
			}
			c.Perguntar(workflow.Pergunta{ID: id, Titulo: titulo, Revisavel: true,
				Contexto: "Confira a proposta e as fontes. A conclusão é sua: a IA apenas sugere.",
				Grupos:   []workflow.GrupoCampos{g},
				Opcoes:   []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}},
				Campos: []workflow.Campo{{ID: "conclusao", Rotulo: "Conclusão", Tipo: workflow.CampoEscolha, Obrigatorio: true, Opcoes: ops,
					Grupo: f + s, Valor: valorAnterior(c, id, "conclusao", an.Conclusao)}},
				Resumir: func(r workflow.Resposta) string { return rotuloConclusao[r.Valor("conclusao")] }})
			return true
		}
	}
	return false
}

var rotuloConclusao = map[string]string{"INDICIOS": "indícios suficientes", "AUSENCIA": "ausência de indícios", "INCONCLUSIVO": "inconclusivo"}
