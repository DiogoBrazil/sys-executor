package pipeline

import (
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// publicarPainel resume, para o usuário, o que a análise produziu até agora.
// Condutas diversas do PADS não aparecem (I-05: só registro interno).
func publicarPainel(c *workflow.Contexto, d *Dados) {
	e := d.IA
	if e == nil {
		return
	}
	p := &workflow.PainelAnalise{}
	// Objeto.
	obj := workflow.SecaoPainel{Titulo: "Objeto do procedimento"}
	switch {
	case e.Conjunto != nil:
		obj.Itens = append(obj.Itens, workflow.ItemPainel{Titulo: "Acusado", Situacao: "confirmado", Linhas: []string{e.Acusado}})
		for _, im := range e.Conjunto.Imputacoes() {
			obj.Itens = append(obj.Itens, workflow.ItemPainel{Titulo: im.ID, Situacao: "congelada",
				Linhas: []string{im.Texto, fmt.Sprintf("%s — natureza %s — fato em %s", im.Dispositivo, rotuloNatureza[string(im.Natureza)], dataTexto(im.Fato.Data))}})
		}
	case e.DelimPADS != nil:
		obj.Itens = append(obj.Itens, workflow.ItemPainel{Titulo: "Imputações propostas pela IA", Situacao: "aguardando confirmação",
			Linhas: []string{fmt.Sprintf("%d imputação(ões) encontradas no Memorando Disciplinar.", len(e.DelimPADS.Imputacoes))}})
	case e.Objeto != nil:
		for _, f := range e.Objeto.FatosAdmitidos() {
			fato, _ := e.Objeto.Fato(f)
			obj.Itens = append(obj.Itens, workflow.ItemPainel{Titulo: "Fato " + f, Situacao: "admitido", Linhas: []string{fato.Descricao}})
		}
		for _, s := range e.Objeto.Sindicados() {
			obj.Itens = append(obj.Itens, workflow.ItemPainel{Titulo: "Sindicado " + s, Situacao: "admitido", Linhas: []string{e.nomes[s]}})
		}
	case e.DelimSR != nil:
		obj.Itens = append(obj.Itens, workflow.ItemPainel{Titulo: "Objeto proposto pela IA", Situacao: "aguardando confirmação",
			Linhas: []string{fmt.Sprintf("%d fato(s) e %d sindicado(s) na Portaria.", len(e.DelimSR.Fatos), len(e.DelimSR.Sindicados))}})
	}
	if len(obj.Itens) > 0 {
		p.Secoes = append(p.Secoes, obj)
	}
	// Leitura das peças.
	if len(e.Extracoes) > 0 {
		lei := workflow.SecaoPainel{Titulo: "Leitura das peças"}
		for _, pc := range d.Pecas {
			ex, ok := e.Extracoes[pc.ID]
			if !ok {
				continue
			}
			it := workflow.ItemPainel{Titulo: pc.Titulo, Situacao: fmt.Sprintf("%d ponto(s) de prova", len(ex.Pontos))}
			for _, pt := range ex.Pontos {
				it.Fontes = append(it.Fontes, fonteDe(pt, pt.ID+" — "+pt.Alvo))
			}
			it.Linhas = append(it.Linhas, ex.Pendencias...)
			lei.Itens = append(lei.Itens, it)
		}
		p.Secoes = append(p.Secoes, lei)
	}
	// Propostas.
	prop := workflow.SecaoPainel{Titulo: "Propostas da IA (a confirmar)"}
	if e.Conjunto != nil {
		for _, im := range e.Conjunto.Imputacoes() {
			an, ok := e.Analises[im.ID]
			if !ok {
				continue
			}
			sit := "sem proposta"
			if an.Resultado != "" {
				sit = "proposta: " + strings.ToLower(an.Resultado)
			}
			it := workflow.ItemPainel{Titulo: im.ID, Situacao: sit, Linhas: append(linhasFundamentos("Fundamento", an.Fundamentos), an.Pendencias...)}
			it.Fontes = fontesDosFundamentos(e, an.Fundamentos)
			prop.Itens = append(prop.Itens, it)
		}
	}
	for k, an := range e.AnalisesSR {
		sit := "sem proposta"
		if an.Conclusao != "" {
			sit = "proposta: " + rotuloConclusao[an.Conclusao]
		}
		f, s, _ := strings.Cut(k, ":")
		prop.Itens = append(prop.Itens, workflow.ItemPainel{Titulo: fmt.Sprintf("Fato %s — %s", f, e.nomes[s]), Situacao: sit,
			Linhas: append(linhasFundamentos("Fundamento", an.Fundamentos), an.Pendencias...), Fontes: fontesDosFundamentos(e, an.Fundamentos)})
	}
	if len(prop.Itens) > 0 {
		p.Secoes = append(p.Secoes, prop)
	}
	if len(e.Pendencias) > 0 {
		pend := workflow.SecaoPainel{Titulo: "Limitações e pendências"}
		for _, l := range e.Pendencias {
			pend.Itens = append(pend.Itens, workflow.ItemPainel{Titulo: l})
		}
		p.Secoes = append(p.Secoes, pend)
	}
	c.PublicarPainel(p)
}
