package pipeline

import (
	"context"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
	"github.com/DiogoBrazil/sys-executor/internal/legal"
	"github.com/DiogoBrazil/sys-executor/internal/minuta"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Redação do Relatório de Sindicância Regular (V4 §§23 a 25; §49). A IA
// redige a síntese (II) e a análise das provas de cada fato (III); a seção I
// e a SOLUÇÃO são compostas pelo aplicativo (minuta.ComporSR).

// prontoParaRedigir: as decisões de que a minuta depende estão tomadas.
func prontoParaRedigir(c *workflow.Contexto, d *Dados) bool {
	if d.IA == nil {
		return false
	}
	if c.Tipo == workflow.TipoSR {
		return d.IA.Objeto != nil && d.Fontes != nil
	}
	return d.IA.Conjunto != nil && d.IA.Julgamento != nil
}

func (p *Pipeline) novoRedatorSR(d *Dados) *redator {
	e := d.IA
	_, objeto := alvos(d)
	r := &redator{d: d, x: e.executor(policy.FluxoSR), congeladas: objeto, permitidos: citacoesPermitidasSR(d.Fontes), sr: true,
		ordem: []string{minuta.SlotSinteseSR}}
	for _, f := range e.Objeto.FatosAdmitidos() {
		r.ordem = append(r.ordem, minuta.SlotAnaliseSR(f))
	}
	return r
}

// citacoesPermitidasSR: só os enquadramentos confirmados (e o art. 9º do
// CPM, quando combinado), com o nome do diploma e com a sigla.
func citacoesPermitidasSR(f *FontesJuridicas) []string {
	var out []string
	if f == nil {
		return out
	}
	for _, qs := range f.EnquadramentoSR {
		for _, q := range qs {
			out = append(out, q.ReferenciaMinuta(), q.Passagem.Referencia)
			r := legal.Referencia{Artigo: q.Passagem.Artigo, Paragrafo: q.Passagem.Paragrafo, Inciso: q.Passagem.Inciso, Alinea: q.Passagem.Alinea}
			for _, x := range diplomasPenaisSR {
				if x.doc == q.Passagem.DocumentoID {
					out = append(out, r.Texto()+" do "+x.sigla, r.Texto()+" da "+x.sigla)
				}
			}
			if q.Art9Alinea != "" {
				out = append(out, "art. 9º, inciso II, alínea "+q.Art9Alinea+", do CPM")
			}
		}
	}
	return out
}

func (rd *Redacao) trechoSR(slot string) *analise.Trecho {
	if slot == minuta.SlotSinteseSR {
		return rd.Sintese
	}
	if t, ok := rd.AnaliseFato[strings.TrimPrefix(slot, minuta.SlotAnaliseSR(""))]; ok {
		return &t
	}
	return nil
}

func (r *redator) fatoDoSlot(slot string) string {
	return strings.TrimPrefix(slot, minuta.SlotAnaliseSR(""))
}

func (r *redator) descricaoSR(slot string) string {
	if slot == minuta.SlotSinteseSR {
		return "síntese dos fatos em apuração"
	}
	return "análise das provas do fato " + r.fatoDoSlot(slot)
}

func (r *redator) pontosSR(slot string) []analise.Ponto {
	if slot == minuta.SlotSinteseSR {
		return r.d.IA.Pontos
	}
	return pontosDe(r.d.IA, r.fatoDoSlot(slot))
}

// resultadoSR: as conclusões confirmadas do fato, para a verificação de
// coerência ("" na síntese).
func (r *redator) resultadoSR(slot string) string {
	if slot == minuta.SlotSinteseSR {
		return ""
	}
	var xs []string
	for _, c := range r.conclusoesDoFato(r.fatoDoSlot(slot)) {
		t := c.Sindicado + ": " + rotuloConclusao[c.Conclusao]
		if len(c.Enquadramentos) > 0 {
			t += " (" + strings.Join(c.Enquadramentos, "; ") + ")"
		}
		xs = append(xs, t)
	}
	return strings.Join(xs, "; ")
}

// conclusoesDoFato: o que a redação recebe de cada sindicado no fato.
func (r *redator) conclusoesDoFato(fato string) []analise.ConclusaoEmRedacao {
	e := r.d.IA
	var out []analise.ConclusaoEmRedacao
	for _, s := range e.Objeto.Sindicados() {
		an := e.AnalisesSR[fato+":"+s]
		c := analise.ConclusaoEmRedacao{SindicadoID: s, Sindicado: e.nomes[s], Conclusao: r.conclusao(fato, s), Fundamentos: an.Fundamentos, Lacunas: an.Lacunas}
		if r.d.Fontes != nil {
			for _, q := range r.d.Fontes.EnquadramentoSR[fato+":"+s] {
				c.Enquadramentos = append(c.Enquadramentos, nomeTipoEnq[q.Tipo]+": "+q.ReferenciaMinuta())
			}
		}
		out = append(out, c)
	}
	return out
}

var nomeTipoEnq = map[sr.TipoEnquadramento]string{sr.Transgressao: "transgressão disciplinar", sr.CrimeMilitar: "crime militar", sr.CrimeComum: "crime comum"}

// conclusao confirmada guardada no momento da redação (a resposta está no
// contexto; a cópia evita depender dele fora da etapa).
func (r *redator) conclusao(fato, s string) string {
	if v, ok := r.d.conclusoesSR[fato+":"+s]; ok {
		return v
	}
	return string(sr.Ausencia)
}

func (r *redator) redigirSR(ctx context.Context, slot string, re *analise.Reescrita) (bool, error) {
	d, rd := r.d, r.d.Redacao
	if slot == minuta.SlotSinteseSR {
		var pecas []analise.PecaResumo
		for _, pc := range d.Pecas {
			pecas = append(pecas, analise.PecaResumo{Titulo: pc.Titulo, SEI: pc.SEI, Funcao: string(pc.Funcao)})
		}
		t, err := analise.RedigirSinteseSR(ctx, r.x, r.congeladas, pecas, r.pontosSR(slot), r.permitidos, re)
		if err != nil {
			return false, err
		}
		if re != nil && t.Pendencia != "" {
			return true, nil
		}
		rd.Sintese = &t
		return false, nil
	}
	fato := r.fatoDoSlot(slot)
	fo, _ := d.IA.Objeto.Fato(fato)
	t, err := analise.RedigirAnaliseFatoSR(ctx, r.x, r.congeladas, analise.DadosAnaliseFatoSR{FatoID: fato, Fato: fo.Descricao, Conclusoes: r.conclusoesDoFato(fato)},
		r.pontosSR(slot), r.permitidos, re)
	if err != nil {
		return false, err
	}
	if re != nil && t.Pendencia != "" {
		return true, nil
	}
	rd.AnaliseFato[fato] = t
	return false, nil
}

// compor monta a minuta com o que já foi redigido e a guarda na tarefa.
func (r *redator) compor() *minuta.Documento {
	if r.sr {
		r.d.Minuta, r.d.entradaMinutaSR = comporMinutaSR(r.d, r.ordem)
	} else {
		r.d.Minuta, r.d.entradaMinuta = comporMinutaPADS(r.d, r.rotulos)
	}
	return r.d.Minuta
}

// comporMinutaSR junta o cadastro, as conclusões, os enquadramentos e os
// trechos redigidos.
func comporMinutaSR(d *Dados, slots []string) (*minuta.Documento, *minuta.EntradaSR) {
	e, rd := d.IA, d.Redacao
	in := minuta.EntradaSR{Portaria: identificacaoPortaria(e)}
	cad := d.cadastro
	in.SemCadastro = cad == nil || len(cad.Pecas) == 0
	pessoas := map[string]workflow.PessoaCadastro{}
	if cad != nil {
		for _, p := range cad.Pessoas {
			pessoas[p.ID] = p
		}
	}
	nomeCad := func(id string) string {
		p := pessoas[id]
		if p.Instituicao {
			return strings.TrimSpace(p.Nome)
		}
		return strings.TrimSpace(strings.TrimSpace(p.PostoGrad) + " " + strings.TrimSpace(p.Nome))
	}
	// SEI do termo de cada pessoa ouvida, pelo papel do depoimento.
	ouvidos := func(papel string) []minuta.PessoaSR {
		var out []minuta.PessoaSR
		if cad == nil {
			return out
		}
		for _, pc := range cad.Pecas {
			if pc.Tipo != workflow.PecaDepoimento || pc.Papel != papel {
				continue
			}
			for _, id := range pc.PessoaIDs {
				if n := nomeCad(id); n != "" {
					out = append(out, minuta.PessoaSR{Nome: n, SEI: pc.SEI})
				}
			}
		}
		return out
	}
	termoSind := ouvidos(workflow.PapelSindicado)
	for _, s := range e.Objeto.Sindicados() {
		p := minuta.PessoaSR{Nome: e.nomes[s]}
		for _, t := range termoSind {
			if mesmoNome(t.Nome, p.Nome) {
				p.SEI = t.SEI
			}
		}
		in.Sindicados = append(in.Sindicados, p)
	}
	if cad != nil {
		for _, id := range cad.Ofendidos {
			if n := nomeCad(id); n != "" {
				in.Ofendidos = append(in.Ofendidos, n)
			}
		}
		in.SemOfendido = cad.SemOfendido
		for _, pc := range cad.Pecas {
			t := strings.ToLower(pc.Titulo + " " + pc.Descricao)
			if pc.Tipo == workflow.PecaOutro && (strings.Contains(t, "laudo") || strings.Contains(t, "perícia") || strings.Contains(t, "pericia")) {
				x := strings.TrimSpace(pc.Titulo)
				if sei := strings.TrimSpace(pc.SEI); sei != "" {
					x += " (SEI nº " + sei + ")"
				}
				in.Pericias = append(in.Pericias, x)
			}
		}
	}
	in.Testemunhas = ouvidos(workflow.PapelTestemunha)
	in.Outras = ouvidos(workflow.PapelDeclarante)
	if rd.Sintese != nil {
		in.Sintese = *rd.Sintese
	}
	temCrime := false
	for _, f := range e.Objeto.FatosAdmitidos() {
		fo, _ := e.Objeto.Fato(f)
		fs := minuta.FatoSR{ID: f, Descricao: fo.Descricao, Analise: rd.AnaliseFato[f]}
		for _, s := range e.Objeto.Sindicados() {
			cs := minuta.ConclusaoSR{SindicadoID: s, Sindicado: e.nomes[s], Indicios: d.conclusoesSR[f+":"+s] == string(sr.Indicios)}
			if cs.Indicios && d.Fontes != nil {
				for _, q := range d.Fontes.EnquadramentoSR[f+":"+s] {
					m := minuta.EnquadramentoSR{Tipo: q.Tipo, Referencia: q.ReferenciaMinuta(), Rubrica: q.Passagem.Rubrica}
					if q.Tipo == sr.Transgressao {
						m.Texto = q.Passagem.Texto
					} else {
						temCrime = true
					}
					cs.Enquadramentos = append(cs.Enquadramentos, m)
				}
			}
			fs.Conclusoes = append(fs.Conclusoes, cs)
		}
		in.Fatos = append(in.Fatos, fs)
	}
	doc := minuta.ComporSR(in)
	if strings.TrimSpace(in.Portaria) != "" {
		doc.Pendencia("Confira a identificação da Portaria na alínea a (número, data e autoridade), como consta dos autos.")
	}
	if temCrime {
		doc.Pendencia("Confira, para cada crime indicado, a redação do dispositivo vigente na data do fato: a biblioteca traz o texto atual.")
	}
	for _, slot := range slots {
		for _, pend := range rd.PendValidacao[slot] {
			doc.Pendencia(pend)
		}
	}
	return doc, &in
}

// identificacaoPortaria: a identificação confirmada pelo usuário, com o
// número SEI da peça, se houver.
func identificacaoPortaria(e *EstadoIA) string {
	t := strings.TrimSpace(e.Objeto.Portaria)
	if t == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(t), "portaria") {
		t = "Portaria nº " + t
	}
	if e.DelimSR != nil {
		if sei := strings.TrimSpace(e.DelimSR.Peca.SEI); sei != "" {
			t += " (SEI nº " + sei + ")"
		}
	}
	return t
}

// mesmoNome: o nome do cadastro está contido no nome confirmado (ou o
// contrário), sem diferença de caixa.
func mesmoNome(a, b string) bool {
	a, b = strings.ToUpper(strings.TrimSpace(a)), strings.ToUpper(strings.TrimSpace(b))
	return a != "" && b != "" && (strings.Contains(a, b) || strings.Contains(b, a))
}
