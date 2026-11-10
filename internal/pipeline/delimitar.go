package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

const (
	perguntaDelimitacao = "confirmar_delimitacao"
	perguntaObjeto      = "confirmar_objeto"
	perguntaSemCentral  = "sem_peca_central"
	quemUsuario         = "Usuário (descrição de mídia)"
	maxExtras           = 1 // imputação/fato não identificado pela IA, informado pelo usuário
)

var errSemIA = erros.Com(erros.IAChaveAusente, errors.New("IA não configurada"))

var opcoesNatureza = []workflow.OpcaoVisao{{ID: "LEVE", Rotulo: "Leve"}, {ID: "MEDIA", Rotulo: "Média"}, {ID: "GRAVE", Rotulo: "Grave"}}

func fonteDe(p analise.Ponto, rotulo string) workflow.Fonte {
	return workflow.Fonte{Rotulo: rotulo, Pagina: p.Pagina, Inicio: p.Inicio, Fim: p.Fim, Trecho: p.Trecho}
}

func dataTexto(d civil.Data) string {
	if d.Zero() {
		return ""
	}
	return d.String()
}

func lerData(s string) civil.Data {
	t, err := time.Parse("02/01/2006", strings.TrimSpace(s))
	if err != nil {
		return civil.Data{}
	}
	return civil.D(t.Day(), int(t.Month()), t.Year())
}

// delimitar é a etapa 4: a IA extrai, o usuário confirma, o Go congela.
func (p *Pipeline) delimitar(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	if aguardar, err := p.autorizarEnvio(c, d); err != nil {
		return workflow.Delimitando, err
	} else if aguardar {
		return workflow.AguardandoEsclarecimento, nil
	}
	if err := p.garantirMotor(ctx, c, d); err != nil {
		d.IA.motorFalhou(err)
		return workflow.Delimitando, err
	}
	e := d.IA
	fluxo := policy.FluxoPADS
	if c.Tipo == workflow.TipoSR {
		fluxo = policy.FluxoSR
	}
	x := e.executor(fluxo)
	if !e.classificou {
		c.Atividade("Identificando a função das peças sem título reconhecido")
		pecas, pend, err := analise.ClassificarResiduais(ctx, x, d.Autos, d.Pecas, &e.reg)
		if err != nil {
			e.motorFalhou(err)
			return workflow.Delimitando, err
		}
		d.Pecas, e.classificou = pecas, true
		e.Pendencias = append(e.Pendencias, pend...)
		c.Publicar(Resumir(d))
	}
	if c.Tipo == workflow.TipoSR {
		return p.delimitarSR(ctx, c, d, x)
	}
	return p.delimitarPADS(ctx, c, d, x)
}

func semPecaCentral(c *workflow.Contexto, peca string) (workflow.Estado, error) {
	c.Perguntar(workflow.Pergunta{ID: perguntaSemCentral, Titulo: "Peça central não encontrada",
		Contexto: fmt.Sprintf("Sem %s não é possível delimitar o objeto do procedimento. Envie o PDF completo do processo.", peca),
		Opcoes: []workflow.Opcao{
			{ID: "outro", Rotulo: "Enviar outro PDF", Destino: workflow.RecebendoUpload},
			{ID: "cancelar", Rotulo: "Cancelar a tarefa", Destino: workflow.Cancelada},
		}})
	return workflow.AguardandoEsclarecimento, nil
}

// ---- PADS ----

func (p *Pipeline) delimitarPADS(ctx context.Context, c *workflow.Contexto, d *Dados, x *analise.Executor) (workflow.Estado, error) {
	e := d.IA
	if e.DelimPADS == nil {
		c.Atividade("Lendo o Memorando Disciplinar")
		r, err := analise.DelimitarPADS(ctx, x, d.Autos, d.Pecas, &e.reg)
		if err != nil {
			if strings.Contains(err.Error(), "não identificado") {
				return semPecaCentral(c, "o Memorando Disciplinar")
			}
			e.motorFalhou(err)
			return workflow.Delimitando, err
		}
		e.DelimPADS = &r
	}
	resp, ok := c.RespostaCompleta(perguntaDelimitacao)
	if !ok {
		ant, _ := c.Anterior(perguntaDelimitacao)
		c.Perguntar(perguntaImputacoes(*e.DelimPADS, ant, nil))
		publicarPainel(c, d)
		return workflow.AguardandoEsclarecimento, nil
	}
	// Enquadramento diferente do citado no Memorando (ou sem citação
	// correspondente): volta ao formulário com o alerta e a confirmação
	// expressa (V4 §§22.1, 48.5); nada é congelado antes disso.
	if div, pendentes := divergencias(*e.DelimPADS, resp); pendentes > 0 {
		c.Esquecer(perguntaDelimitacao)
		c.Perguntar(perguntaImputacoes(*e.DelimPADS, resp, div))
		publicarPainel(c, d)
		return workflow.AguardandoEsclarecimento, nil
	}
	set, acusado, err := congelarImputacoes(*e.DelimPADS, resp)
	if err != nil {
		return "", erros.Com(erros.InternoInesperado, err)
	}
	j, err := pads.NovoJulgamento([]string{acusado}, set)
	if err != nil {
		return "", erros.Com(erros.InternoInesperado, err)
	}
	e.Conjunto, e.Julgamento, e.Acusado = set, j, acusado
	publicarPainel(c, d)
	return workflow.Analisando, nil
}

// perguntaImputacoes monta o formulário de confirmação (PADS-ESC-001).
func perguntaImputacoes(r analise.PropostaDelimitacaoPADS, ant workflow.Resposta, div map[string]string) workflow.Pergunta {
	val := func(campo, padrao string) string {
		if v, ok := ant.Valores[campo]; ok {
			return v
		}
		return padrao
	}
	pg := workflow.Pergunta{
		ID: perguntaDelimitacao, Titulo: "Confirme as imputações do Memorando Disciplinar", Revisavel: true,
		Contexto: "A IA leu o Memorando Disciplinar e propôs as condutas e datas abaixo. Confira cada uma no trecho original e corrija o que for preciso. " +
			"O enquadramento é seu: escolha o diploma e o dispositivo de cada imputação (o dispositivo citado no Memorando aparece só para conferência). " +
			"Depois de confirmadas, só estas imputações serão julgadas; conduta diversa encontrada na instrução não vira acusação.",
		Opcoes: []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar imputações", Validar: true},
			{ID: "cancelar", Rotulo: "Cancelar a tarefa", Destino: workflow.Cancelada}},
	}
	// Acusado (PADS-ESC-006; PADS-1AC-01).
	grupo := workflow.GrupoCampos{ID: "acusado", Titulo: "Acusado"}
	campo := workflow.Campo{ID: "acusado", Rotulo: "Acusado", Tipo: workflow.CampoTexto, Obrigatorio: true, Grupo: "acusado"}
	switch len(r.Acusados) {
	case 0:
		grupo.Alertas = append(grupo.Alertas, "O acusado não foi identificado com segurança no memorando; informe-o.")
	case 1:
		campo.Valor = r.Acusados[0].Nome
		campo.Fontes = []workflow.Fonte{fonteDe(r.Acusados[0].Ponto, "Memorando")}
	default:
		grupo.Alertas = append(grupo.Alertas, fmt.Sprintf("O memorando parece indicar %d acusados, mas o PADS admite um único acusado. Escolha o acusado deste processo.", len(r.Acusados)))
		campo.Tipo = workflow.CampoEscolha
		for _, a := range r.Acusados {
			campo.Opcoes = append(campo.Opcoes, workflow.OpcaoVisao{ID: a.Nome, Rotulo: strings.TrimSpace(a.PostoGraduacao + " " + a.Nome)})
			campo.Fontes = append(campo.Fontes, fonteDe(a.Ponto, "Memorando"))
		}
	}
	campo.Valor = val("acusado", campo.Valor)
	pg.Grupos = append(pg.Grupos, grupo)
	pg.Campos = append(pg.Campos, campo)

	add := func(id, titulo string, desc []string, fontes []workflow.Fonte, alertas []string, conduta, data, incluir string) {
		if msg, ok := div[id]; ok {
			alertas = append(alertas, msg)
		}
		pg.Grupos = append(pg.Grupos, workflow.GrupoCampos{ID: id, Titulo: titulo, Descricao: desc, Fontes: fontes, Alertas: alertas})
		sim := &workflow.Condicao{Campo: "incluir_" + id, Valor: "sim"}
		pg.Campos = append(pg.Campos,
			workflow.Campo{ID: "incluir_" + id, Rotulo: "Incluir esta imputação", Tipo: workflow.CampoSimNao, Obrigatorio: true, Grupo: id, Valor: val("incluir_"+id, incluir)},
			workflow.Campo{ID: "conduta_" + id, Rotulo: "Conduta imputada", Tipo: workflow.CampoTextoLongo, Obrigatorio: true, Grupo: id, Valor: val("conduta_"+id, conduta), MostrarSe: sim})
		pg.Campos = append(pg.Campos, camposEnquadramento(id, val, sim)...)
		pg.Campos = append(pg.Campos,
			workflow.Campo{ID: "data_" + id, Rotulo: "Data do fato", Tipo: workflow.CampoData, Obrigatorio: true, Grupo: id, Valor: val("data_"+id, data), MostrarSe: sim,
				Ajuda: "DD/MM/AAAA"})
		if _, ok := div[id]; ok {
			pg.Campos = append(pg.Campos, workflow.Campo{ID: campoConfirmaDiv + id, Grupo: id, Tipo: workflow.CampoSimNao, Obrigatorio: true, MostrarSe: sim,
				Rotulo: "Confirmo este enquadramento para a conduta acusada no Memorando",
				Opcoes: []workflow.OpcaoVisao{{ID: "sim", Rotulo: "Confirmo"}, {ID: "nao", Rotulo: "Vou corrigir"}},
				Valor:  val(campoConfirmaDiv+id, "")})
		}
	}
	for _, im := range r.Imputacoes {
		desc := []string{im.Conduta, descricaoCitado(im)}
		if im.Natureza != "" {
			desc = append(desc, "Natureza declarada no Memorando: "+rotuloNatureza[string(im.Natureza)]+".")
		}
		add(im.ID, im.ID, desc, []workflow.Fonte{fonteDe(im.Ponto, "Memorando")}, im.Pendencias,
			im.Conduta, dataTexto(im.Data), "sim")
	}
	for k := 1; k <= maxExtras; k++ {
		id := fmt.Sprintf("IMP-X%d", k)
		add(id, "Imputação não identificada pela IA", []string{"Use só se o memorando trouxer outra imputação que não apareceu acima."},
			nil, nil, "", "", "nao")
	}
	for _, pnd := range r.Pendencias {
		pg.Grupos[0].Alertas = append(pg.Grupos[0].Alertas, pnd)
	}
	pg.Resumir = func(rs workflow.Resposta) string {
		return fmt.Sprintf("%d imputação(ões); acusado: %s", contarSim(rs.Valores, "incluir_"), rs.Valor("acusado"))
	}
	pg.Conferir = func(vs map[string]string) error {
		if contarSim(vs, "incluir_") == 0 {
			return errors.New("inclua ao menos uma imputação")
		}
		return nil
	}
	return pg
}

func contarSim(vs map[string]string, prefixo string) int {
	n := 0
	for k, v := range vs {
		if strings.HasPrefix(k, prefixo) && v == "sim" {
			n++
		}
	}
	return n
}

// congelarImputacoes cria o ImputacaoSet a partir da confirmação.
func congelarImputacoes(r analise.PropostaDelimitacaoPADS, resp workflow.Resposta) (*pads.ImputacaoSet, string, error) {
	set := pads.NovoImputacaoSet()
	ids := []string{}
	evid := map[string][]string{}
	for _, im := range r.Imputacoes {
		ids = append(ids, im.ID)
		evid[im.ID] = []string{fmt.Sprintf("pág. %d", im.Ponto.Pagina)}
	}
	for k := 1; k <= maxExtras; k++ {
		ids = append(ids, fmt.Sprintf("IMP-X%d", k))
	}
	n := 0
	for _, id := range ids {
		if resp.Valor("incluir_"+id) != "sim" {
			continue
		}
		n++
		disp, nat, ok := escolhido(resp, id)
		if !ok {
			return nil, "", fmt.Errorf("enquadramento de %s incompleto", id)
		}
		im := pads.Imputacao{
			ID: fmt.Sprintf("IMP-%d", n), Texto: resp.Valor("conduta_" + id), Dispositivo: disp.Referencia(),
			Natureza: nat, Fato: civil.Instante{Data: lerData(resp.Valor("data_" + id))},
			Evid: evid[id],
		}
		if err := set.Adicionar(im); err != nil {
			return nil, "", err
		}
	}
	if err := set.Congelar(); err != nil {
		return nil, "", err
	}
	return set, resp.Valor("acusado"), nil
}

// ---- Sindicância Regular ----

func (p *Pipeline) delimitarSR(ctx context.Context, c *workflow.Contexto, d *Dados, x *analise.Executor) (workflow.Estado, error) {
	e := d.IA
	if e.DelimSR == nil {
		c.Atividade("Lendo a Portaria de instauração")
		r, err := analise.DelimitarSR(ctx, x, d.Autos, d.Pecas, &e.reg)
		if err != nil {
			if strings.Contains(err.Error(), "não identificada") {
				return semPecaCentral(c, "a Portaria de instauração")
			}
			e.motorFalhou(err)
			return workflow.Delimitando, err
		}
		e.DelimSR = &r
	}
	resp, ok := c.RespostaCompleta(perguntaObjeto)
	if !ok {
		ant, _ := c.Anterior(perguntaObjeto)
		c.Perguntar(perguntaObjetoSR(*e.DelimSR, ant))
		publicarPainel(c, d)
		return workflow.AguardandoEsclarecimento, nil
	}
	obj, nomes, err := montarObjeto(*e.DelimSR, resp)
	if err != nil {
		return "", erros.Com(erros.InternoInesperado, err)
	}
	e.Objeto, e.nomes = obj, nomes
	publicarPainel(c, d)
	return workflow.Analisando, nil
}

func perguntaObjetoSR(r analise.PropostaObjetoSR, ant workflow.Resposta) workflow.Pergunta {
	val := func(campo, padrao string) string {
		if v, ok := ant.Valores[campo]; ok {
			return v
		}
		return padrao
	}
	pg := workflow.Pergunta{
		ID: perguntaObjeto, Titulo: "Confirme o objeto da Sindicância", Revisavel: true,
		Contexto: "A IA leu a Portaria de instauração e propôs os fatos e os sindicados abaixo. Confira no trecho original e corrija o que for preciso. " +
			"Fatos ou militares novos encontrados na instrução só entram depois da sua autorização.",
		Opcoes: []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar objeto", Validar: true},
			{ID: "cancelar", Rotulo: "Cancelar a tarefa", Destino: workflow.Cancelada}},
	}
	port := workflow.GrupoCampos{ID: "portaria", Titulo: "Portaria", Alertas: r.Pendencias}
	pg.Grupos = append(pg.Grupos, port)
	pg.Campos = append(pg.Campos, workflow.Campo{ID: "portaria", Rotulo: "Identificação da portaria", Tipo: workflow.CampoTexto, Obrigatorio: true, Grupo: "portaria",
		Valor: val("portaria", strings.TrimSpace(strings.Join([]string{r.Numero, r.Autoridade, dataTexto(r.Data)}, " — "))), Ajuda: "Número, autoridade e data"})
	for _, f := range r.Fatos {
		id := f.ID
		pg.Grupos = append(pg.Grupos, workflow.GrupoCampos{ID: id, Titulo: "Fato " + id, Fontes: []workflow.Fonte{fonteDe(f.Ponto, "Portaria")}})
		pg.Campos = append(pg.Campos,
			workflow.Campo{ID: "incluir_" + id, Rotulo: "Incluir este fato", Tipo: workflow.CampoSimNao, Obrigatorio: true, Grupo: id, Valor: val("incluir_"+id, "sim")},
			workflow.Campo{ID: "fato_" + id, Rotulo: "Descrição do fato", Tipo: workflow.CampoTextoLongo, Obrigatorio: true, Grupo: id,
				Valor: val("fato_"+id, f.Descricao), MostrarSe: &workflow.Condicao{Campo: "incluir_" + id, Valor: "sim"}})
	}
	for _, s := range r.Sindicados {
		id := s.ID
		pg.Grupos = append(pg.Grupos, workflow.GrupoCampos{ID: id, Titulo: "Sindicado " + id, Fontes: []workflow.Fonte{fonteDe(s.Ponto, "Portaria")}})
		pg.Campos = append(pg.Campos,
			workflow.Campo{ID: "incluir_" + id, Rotulo: "Incluir como sindicado", Tipo: workflow.CampoSimNao, Obrigatorio: true, Grupo: id, Valor: val("incluir_"+id, "sim")},
			workflow.Campo{ID: "nome_" + id, Rotulo: "Posto/graduação e nome", Tipo: workflow.CampoTexto, Obrigatorio: true, Grupo: id,
				Valor: val("nome_"+id, strings.TrimSpace(s.PostoGraduacao+" "+s.Nome)), MostrarSe: &workflow.Condicao{Campo: "incluir_" + id, Valor: "sim"}})
	}
	pg.Resumir = func(rs workflow.Resposta) string {
		return fmt.Sprintf("%d fato(s), %d sindicado(s)", contarSim(rs.Valores, "incluir_F"), contarSim(rs.Valores, "incluir_S"))
	}
	pg.Conferir = func(vs map[string]string) error {
		if contarSim(vs, "incluir_F") == 0 || contarSim(vs, "incluir_S") == 0 {
			return errors.New("inclua ao menos um fato e um sindicado")
		}
		return nil
	}
	return pg
}

func montarObjeto(r analise.PropostaObjetoSR, resp workflow.Resposta) (*sr.ObjetoSR, map[string]string, error) {
	var fatos []sr.Fato
	for _, f := range r.Fatos {
		if resp.Valor("incluir_"+f.ID) == "sim" {
			fatos = append(fatos, sr.Fato{ID: f.ID, Descricao: resp.Valor("fato_" + f.ID)})
		}
	}
	var pessoas []sr.Pessoa
	var sind []string
	nomes := map[string]string{}
	for _, s := range r.Sindicados {
		if resp.Valor("incluir_"+s.ID) == "sim" {
			pessoas = append(pessoas, sr.Pessoa{ID: s.ID, Tipo: sr.Militar})
			sind = append(sind, s.ID)
			nomes[s.ID] = resp.Valor("nome_" + s.ID)
		}
	}
	o, err := sr.NovoObjeto(resp.Valor("portaria"), fatos, pessoas, sind)
	return o, nomes, err
}
