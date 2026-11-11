package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/legal"
	"github.com/DiogoBrazil/sys-executor/internal/minuta"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Redacao guarda os trechos já redigidos pela IA nesta tarefa. Um trecho
// pronto não é pedido de novo quando a etapa recomeça (falha de rede,
// revisão de dados pessoais). Rever uma decisão da análise descarta tudo.
type Redacao struct {
	Relatorio     *analise.Trecho
	Fundamentacao map[string]analise.Trecho       // por imputação
	Fatores       map[string][]analise.FatorArt36 // por imputação principal
	PendFatores   map[string]string
	// Validação (por trecho): assinatura do texto já conferido, ciclos de
	// correção automática feitos e o que ficou para o usuário conferir.
	Verificado    map[string]string
	Ciclos        map[string]int
	PendValidacao map[string][]string
}

func novaRedacao() *Redacao {
	return &Redacao{Fundamentacao: map[string]analise.Trecho{}, Fatores: map[string][]analise.FatorArt36{}, PendFatores: map[string]string{},
		Verificado: map[string]string{}, Ciclos: map[string]int{}, PendValidacao: map[string][]string{}}
}

// citacoesPermitidas: o que a redação pode citar de norma, além do que está
// nas evidências — só o que veio da biblioteca (TRV-LIB-004) e o texto fixo
// aprovado pelo usuário (art. 5º, LV, da Constituição).
func citacoesPermitidas(f *FontesJuridicas) []string {
	out := []string{"art. 5º, inciso LV, da Constituição Federal"}
	if f == nil {
		return out
	}
	for _, p := range f.Enquadramento {
		out = append(out, p.Referencia, "artigo "+p.Artigo+" do RDPM")
	}
	for _, p := range f.Base {
		out = append(out, p.Referencia, "artigo "+p.Artigo+" do RDPM")
	}
	return out
}

const (
	slotRelatorio      = "relatorio"
	prefixoSlotFund    = "fundamentacao:"
	prefixoSlotArt36   = "art36:"
	unidadeDeProgresso = "trecho(s)"
)

// redator redige (ou reescreve) um trecho por vez, sempre com as mesmas
// fontes: é usado pela redação, pela correção automática e pelos ajustes.
type redator struct {
	d          *Dados
	x          *analise.Executor
	congeladas policy.Secao
	permitidos []string
	rotulos    [7]string
	imps       map[string]pads.Imputacao
	resultados map[string]pads.Resultado
	sancoes    map[string]SancaoCalculada
	ordem      []string // trechos, na ordem da minuta
}

func (p *Pipeline) novoRedator(d *Dados) *redator {
	e := d.IA
	_, congeladas := alvos(d)
	r := &redator{d: d, x: e.executor(policy.FluxoPADS), congeladas: congeladas, permitidos: citacoesPermitidas(d.Fontes), rotulos: rotulosArt36(p),
		imps: map[string]pads.Imputacao{}, resultados: map[string]pads.Resultado{}, sancoes: map[string]SancaoCalculada{}, ordem: []string{slotRelatorio}}
	for _, rr := range e.Julgamento.Resultados() {
		r.resultados[rr.ImputacaoID] = rr.Resultado
	}
	for _, im := range e.Conjunto.Imputacoes() {
		r.imps[im.ID] = im
		r.ordem = append(r.ordem, prefixoSlotFund+im.ID)
	}
	if d.Calculo != nil {
		for _, s := range d.Calculo.Sancoes {
			r.sancoes[s.ImputacaoID] = s
			r.ordem = append(r.ordem, prefixoSlotArt36+s.ImputacaoID)
		}
	}
	return r
}

// descricao: como o trecho é chamado para o usuário e para a IA.
func (r *redator) descricao(slot string) string {
	switch {
	case slot == slotRelatorio:
		return "relatório (relato dos fatos, da defesa e das provas)"
	case strings.HasPrefix(slot, prefixoSlotFund):
		return "fundamentação da imputação do " + r.imps[strings.TrimPrefix(slot, prefixoSlotFund)].Dispositivo
	}
	return "exame do art. 36 quanto à transgressão do " + r.imps[strings.TrimPrefix(slot, prefixoSlotArt36)].Dispositivo
}

// feito informa se o trecho já foi redigido.
func (r *redator) feito(slot string) bool {
	rd := r.d.Redacao
	switch {
	case slot == slotRelatorio:
		return rd.Relatorio != nil
	case strings.HasPrefix(slot, prefixoSlotFund):
		_, ok := rd.Fundamentacao[strings.TrimPrefix(slot, prefixoSlotFund)]
		return ok
	}
	_, ok := rd.Fatores[strings.TrimPrefix(slot, prefixoSlotArt36)]
	return ok
}

// paragrafos devolve o texto atual do trecho que cita evidências (o que a
// verificação de suporte confere).
func (r *redator) paragrafos(slot string) []analise.Paragrafo {
	rd := r.d.Redacao
	switch {
	case slot == slotRelatorio:
		if rd.Relatorio == nil {
			return nil
		}
		return rd.Relatorio.Paragrafos
	case strings.HasPrefix(slot, prefixoSlotFund):
		return rd.Fundamentacao[strings.TrimPrefix(slot, prefixoSlotFund)].Paragrafos
	}
	var out []analise.Paragrafo
	for _, f := range rd.Fatores[strings.TrimPrefix(slot, prefixoSlotArt36)] {
		if len(f.Evid) > 0 && f.Texto != analise.AusenciaDeElementos {
			out = append(out, analise.Paragrafo{Texto: f.Texto, Evid: f.Evid})
		}
	}
	return out
}

// textoAtual: o trecho como está, para a reescrita.
func (r *redator) textoAtual(slot string) []string {
	var out []string
	if strings.HasPrefix(slot, prefixoSlotArt36) {
		for _, f := range r.d.Redacao.Fatores[strings.TrimPrefix(slot, prefixoSlotArt36)] {
			out = append(out, fmt.Sprintf("inciso %d: %s", f.Inciso, f.Texto))
		}
		return out
	}
	for _, p := range r.paragrafos(slot) {
		out = append(out, p.Texto)
	}
	return out
}

func (r *redator) pontos(slot string) []analise.Ponto {
	if slot == slotRelatorio {
		return r.d.IA.Pontos
	}
	_, id, _ := strings.Cut(slot, ":")
	return pontosDoAlvo(r.d.IA.Pontos, id)
}

// resultado confirmado da imputação do trecho ("" fora da fundamentação).
func (r *redator) resultado(slot string) string {
	if strings.HasPrefix(slot, prefixoSlotFund) {
		return string(r.resultados[strings.TrimPrefix(slot, prefixoSlotFund)])
	}
	return ""
}

// redigir pede o trecho à IA e o guarda. Numa reescrita (re != nil) que a IA
// não consiga fazer com segurança, o texto anterior é mantido (mantido=true).
func (r *redator) redigir(ctx context.Context, slot string, re *analise.Reescrita) (mantido bool, err error) {
	d, e, rd := r.d, r.d.IA, r.d.Redacao
	switch {
	case slot == slotRelatorio:
		var pecas []analise.PecaResumo
		for _, pc := range d.Pecas {
			pecas = append(pecas, analise.PecaResumo{Titulo: pc.Titulo, SEI: pc.SEI, Funcao: string(pc.Funcao)})
		}
		t, err := analise.RedigirRelatorio(ctx, r.x, r.congeladas, pecas, e.Pontos, r.permitidos, re)
		if err != nil {
			return false, err
		}
		if re != nil && t.Pendencia != "" {
			return true, nil
		}
		rd.Relatorio = &t
	case strings.HasPrefix(slot, prefixoSlotFund):
		id := strings.TrimPrefix(slot, prefixoSlotFund)
		im, an := r.imps[id], e.Analises[id]
		t, err := analise.RedigirFundamentacao(ctx, r.x, r.congeladas, analise.DadosFundamentacao{
			Imputacao: im.ID, Conduta: im.Texto, Dispositivo: im.Dispositivo, Resultado: string(r.resultados[id]),
			Fundamentos: an.Fundamentos, Teses: an.TesesDefesa, Lacunas: an.Lacunas,
		}, r.pontos(slot), r.permitidos, re)
		if err != nil {
			return false, err
		}
		if re != nil && t.Pendencia != "" {
			return true, nil
		}
		rd.Fundamentacao[id] = t
	default:
		id := strings.TrimPrefix(slot, prefixoSlotArt36)
		fs, pend, err := analise.RedigirArt36(ctx, r.x, r.congeladas, secaoCalculoIA(d.Calculo, r.sancoes[id]), id, r.rotulos[:], r.pontos(slot), r.permitidos, re)
		if err != nil {
			return false, err
		}
		if re != nil && pend != "" {
			return true, nil
		}
		rd.Fatores[id], rd.PendFatores[id] = fs, pend
	}
	return false, nil
}

// prepararIA confere a autorização de envio e abre o motor, como toda etapa
// que chama a IA. aguardar=true: há pergunta pendente ao usuário.
func (p *Pipeline) prepararIA(ctx context.Context, c *workflow.Contexto, d *Dados) (aguardar bool, err error) {
	if aguardar, err := p.autorizarEnvio(c, d); err != nil || aguardar {
		return aguardar, err
	}
	if err := p.garantirMotor(ctx, c, d); err != nil {
		d.IA.motorFalhou(err)
		return false, err
	}
	return false, nil
}

// redigir é a etapa 12/13: a IA redige os trechos e o aplicativo compõe a
// minuta. Só o PADS, por enquanto. A minuta só é mostrada depois da
// validação (V4 §42.1).
func (p *Pipeline) redigir(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	if c.Tipo != workflow.TipoPADS || d.IA == nil || d.IA.Conjunto == nil || d.IA.Julgamento == nil {
		// A redação da SR ainda não existe: pára como etapa futura.
		return workflow.Redigindo, erros.Novo(erros.TarefaEtapaIndisponiv)
	}
	if aguardar, err := p.prepararIA(ctx, c, d); err != nil {
		return workflow.Redigindo, err
	} else if aguardar {
		return workflow.AguardandoEsclarecimento, nil
	}
	if d.Redacao == nil {
		d.Redacao = novaRedacao()
	}
	r := p.novoRedator(d)
	for i, slot := range r.ordem {
		if r.feito(slot) {
			continue
		}
		c.Atividade("Redigindo: " + r.descricao(slot) + ".")
		c.Progresso(i, len(r.ordem), unidadeDeProgresso)
		if _, err := r.redigir(ctx, slot, nil); err != nil {
			d.IA.motorFalhou(err)
			return workflow.Redigindo, err
		}
	}
	c.Progresso(len(r.ordem), len(r.ordem), unidadeDeProgresso)
	d.Minuta, d.entradaMinuta = comporMinutaPADS(d, r.rotulos)
	return workflow.Validando, nil
}

func pontosDoAlvo(ps []analise.Ponto, alvo string) []analise.Ponto {
	var out []analise.Ponto
	for _, p := range ps {
		if p.Alvo == alvo {
			out = append(out, p)
		}
	}
	return out
}

// rotulosArt36 lê da biblioteca o enunciado dos incisos I a VII do art. 36.
func rotulosArt36(p *Pipeline) [7]string {
	var out [7]string
	abrir := p.Biblioteca
	if abrir == nil {
		abrir = legal.Padrao
	}
	b, err := abrir()
	if err != nil {
		return out
	}
	for i, inc := range []string{"I", "II", "III", "IV", "V", "VI", "VII"} {
		if ps, ok := b.Dispositivo(legal.Referencia{Documento: documentoRDPM, Artigo: "36", Inciso: inc}); ok {
			out[i] = ps.Texto
		}
	}
	return out
}

// secaoCalculoIA: os valores definitivos que a IA recebe para examinar o
// art. 36 (antecedentes e comportamento). Ela não recalcula nada.
func secaoCalculoIA(calc *CalculoOficial, s SancaoCalculada) policy.Secao {
	dados := map[string]any{
		"imputacao_id": s.ImputacaoID,
		"natureza":     rotuloNatureza[string(s.Natureza)],
		"sancao":       rotuloSancao[s.Sancao],
		"reincidencia": s.Reincidencia.Reconhecida,
	}
	if calc != nil {
		dados["comportamento_e_pontos_atuais"] = calc.Memoria.Itens[0].Linhas
		dados["historico_disciplinar_informado"] = calc.Memoria.Itens[2].Linhas
	}
	return policy.Secao{Nome: "CALCULO_OFICIAL", Dados: dados}
}

// comporMinutaPADS junta as decisões, o cálculo e os trechos redigidos.
func comporMinutaPADS(d *Dados, rotulos [7]string) (*minuta.Documento, *minuta.EntradaPADS) {
	e, r := d.IA, d.Redacao
	in := minuta.EntradaPADS{Acusado: e.Acusado, RotulosArt36: rotulos}
	if e.DelimPADS != nil {
		if len(e.DelimPADS.Acusados) == 1 {
			ac := e.DelimPADS.Acusados[0]
			in.Matricula = ac.Matricula
			// O acusado confirmado pode vir sem o posto (autos sem cadastro).
			if pg := strings.TrimSpace(ac.PostoGraduacao); pg != "" && !strings.Contains(strings.ToUpper(in.Acusado), strings.ToUpper(pg)) &&
				strings.EqualFold(strings.TrimSpace(ac.Nome), strings.TrimSpace(in.Acusado)) {
				in.Acusado = pg + " " + in.Acusado
			}
		}
		in.Memorando = "Memorando Disciplinar incluído nos autos"
		if sei := strings.TrimSpace(e.DelimPADS.Peca.SEI); sei != "" {
			in.Memorando = "Memorando Disciplinar (SEI nº " + sei + ")"
		}
	}
	if g, err := e.Julgamento.ResultadoGlobal(); err == nil {
		in.Global = g
	}
	if r.Relatorio != nil {
		in.Relatorio = *r.Relatorio
	}
	procedente := map[string]bool{}
	for _, rr := range e.Julgamento.Resultados() {
		procedente[rr.ImputacaoID] = rr.Resultado == pads.Procedente
	}
	disp := map[string]string{}
	for _, im := range e.Conjunto.Imputacoes() {
		disp[im.ID] = im.Dispositivo
		mi := minuta.ImputacaoPADS{ID: im.ID, Conduta: im.Texto, Dispositivo: im.Dispositivo, Natureza: im.Natureza,
			Procedente: procedente[im.ID], Fundamentacao: r.Fundamentacao[im.ID]}
		if d.Fontes != nil {
			mi.TextoDispositivo = d.Fontes.Enquadramento[im.ID].Texto
		}
		in.Imputacoes = append(in.Imputacoes, mi)
	}
	if calc := d.Calculo; calc != nil {
		in.Pontuacao, in.Servicos, in.PendenciasCalculo = calc.Pontuacao, calc.Servicos, calc.Pendencias
		for _, s := range calc.Sancoes {
			ms := minuta.SancaoPADS{ImputacaoID: s.ImputacaoID, Dispositivo: disp[s.ImputacaoID], Natureza: s.Natureza, Sancao: s.Sancao, Dias: s.Dias,
				AtenuantesContadas: s.Dosimetria.AtenuantesContadas, AgravantesContadas: s.Dosimetria.AgravantesContadas,
				Atenuantes: s.Atenuantes, Agravantes: s.Agravantes, Justificacoes: s.Justificacoes,
				Fatores: r.Fatores[s.ImputacaoID], PendenciaFatores: r.PendFatores[s.ImputacaoID]}
			for _, sec := range s.Secundarias {
				ms.Secundarias = append(ms.Secundarias, disp[sec])
			}
			in.Sancoes = append(in.Sancoes, ms)
		}
	}
	doc := minuta.ComporPADS(in)
	for _, slot := range append([]string{slotRelatorio}, slotsDe(in)...) {
		for _, pend := range r.PendValidacao[slot] {
			doc.Pendencia(pend)
		}
	}
	return doc, &in
}

func slotsDe(in minuta.EntradaPADS) []string {
	var out []string
	for _, im := range in.Imputacoes {
		out = append(out, prefixoSlotFund+im.ID)
	}
	for _, s := range in.Sancoes {
		out = append(out, prefixoSlotArt36+s.ImputacaoID)
	}
	return out
}

// visaoMinuta converte a minuta para a interface, com as fontes dos autos de
// cada bloco redigido pela IA.
func visaoMinuta(d *Dados, validada bool) *workflow.MinutaVisao {
	doc := d.Minuta
	if doc == nil {
		return nil
	}
	porID := map[string]analise.Ponto{}
	if d.IA != nil {
		for _, p := range d.IA.Pontos {
			porID[p.ID] = p
		}
	}
	v := &workflow.MinutaVisao{Pendencias: doc.Pendencias, CamposPendentes: doc.CamposPendentes(), Validada: validada, Versao: d.versaoMinuta}
	for _, a := range d.Achados {
		v.Achados = append(v.Achados, a.Mensagem)
	}
	for _, a := range d.Ajustes {
		v.Ajustes = append(v.Ajustes, workflow.AjusteVisao{Pedido: a.Pedido, Resultado: a.Resultado, Aplicado: a.Aplicado})
	}
	for _, b := range doc.Blocos {
		bv := workflow.BlocoMinuta{Tipo: string(b.Tipo), Texto: b.Texto, Origem: string(b.Origem), Pendente: b.Pendente}
		for _, id := range b.Evid {
			if p, ok := porID[id]; ok {
				bv.Fontes = append(bv.Fontes, fonteDe(p, fmt.Sprintf("%s, pág. %d", p.Titulo, p.Pagina)))
			}
		}
		v.Blocos = append(v.Blocos, bv)
	}
	return v
}
