package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/legal"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// FontesJuridicas é o que a biblioteca devolveu para a tarefa (etapa 11 do
// plano §5.4). A redação só pode citar norma a partir destas passagens.
type FontesJuridicas struct {
	// Enquadramento: dispositivo de cada imputação do PADS, por ID.
	Enquadramento map[string]legal.Passagem
	// Base: dispositivos que regem o julgamento, na ordem em que são usados.
	Base []legal.Passagem
	// Lacunas: o que foi procurado e não está na biblioteca (V4 §41: sinalizar
	// a limitação; nunca suprir com texto inventado).
	Lacunas []string
}

// Base do julgamento do PADS no RDPM. "Sempre" vale para qualquer resultado;
// "punicao" só entra quando há imputação procedente.
var basePADS = []struct {
	artigo  string
	punicao bool
}{
	{"4", false},  // princípios; ampla defesa e contraditório
	{"32", false}, // julgamento depois das alegações finais; publicação
	{"33", false}, // conteúdo do julgamento
	{"36", true},  // fatores a considerar
	{"37", true},  // causas de justificação
	{"38", true},  // atenuantes
	{"39", true},  // agravantes e reincidência
	{"41", true},  // punições
	{"45", true},  // cumprimento na residência
	{"50", true},  // competência
	{"58", true},  // preceitos da aplicação da punição
	{"70", true},  // comportamento
	{"71", true},  // mudança de comportamento e pontos
}

const documentoRDPM = "rdpm"

// consultarFontes recupera, na biblioteca jurídica local, os dispositivos
// que a minuta vai citar. Não usa a IA nem a rede. No PADS: o dispositivo de
// cada imputação e a base do julgamento. Na SR ainda não há conjunto
// definido: a etapa só registra que a biblioteca está disponível.
func (p *Pipeline) consultarFontes(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	abrir := p.Biblioteca
	if abrir == nil {
		abrir = legal.Padrao
	}
	c.Atividade("Consultando a biblioteca jurídica local.")
	b, err := abrir()
	if err != nil {
		return workflow.ConsultandoFontes, err
	}
	f := &FontesJuridicas{Enquadramento: map[string]legal.Passagem{}}
	if c.Tipo == workflow.TipoSR || d.IA == nil || d.IA.Conjunto == nil {
		d.Fontes = f
		publicarPainel(c, d)
		return workflow.Redigindo, nil
	}
	e := d.IA
	for _, im := range e.Conjunto.Imputacoes() {
		if err := ctx.Err(); err != nil {
			return workflow.ConsultandoFontes, err
		}
		r, ok := b.InterpretarReferencia(im.Dispositivo)
		if ok && r.Documento != "" {
			if ps, ok := b.Dispositivo(r); ok {
				f.Enquadramento[im.ID] = ps
				continue
			}
		}
		f.Lacunas = append(f.Lacunas, fmt.Sprintf("O dispositivo da %s (%s) não foi localizado na biblioteca jurídica.", im.ID, im.Dispositivo))
	}
	procedente := false
	if e.Julgamento != nil {
		for _, r := range e.Julgamento.Resultados() {
			procedente = procedente || r.Resultado == pads.Procedente
		}
	}
	for _, a := range basePADS {
		if a.punicao && !procedente {
			continue
		}
		ps, ok := b.Dispositivo(legal.Referencia{Documento: documentoRDPM, Artigo: a.artigo})
		if !ok {
			f.Lacunas = append(f.Lacunas, fmt.Sprintf("O art. %s do RDPM não foi localizado na biblioteca jurídica.", a.artigo))
			continue
		}
		f.Base = append(f.Base, ps)
	}
	d.Fontes = f
	publicarPainel(c, d)
	return workflow.Calculando, nil
}

// secaoFontes mostra ao usuário o que foi recuperado, com documento, versão
// e página, para conferência no original.
func secaoFontes(d *Dados) (workflow.SecaoPainel, bool) {
	f := d.Fontes
	s := workflow.SecaoPainel{Titulo: "Fontes jurídicas"}
	if f == nil {
		return s, false
	}
	origem := func(p legal.Passagem) string {
		o := fmt.Sprintf("%s — pág. %d — versão %s", p.Arquivo, p.Pagina, p.Versao)
		if p.OCR {
			o += " — texto lido por OCR: confira no original"
		}
		return o
	}
	if d.IA != nil && d.IA.Conjunto != nil {
		for _, im := range d.IA.Conjunto.Imputacoes() {
			p, ok := f.Enquadramento[im.ID]
			if !ok {
				continue
			}
			it := workflow.ItemPainel{Titulo: p.Referencia, Situacao: "enquadramento da " + im.ID}
			if p.Caput != "" {
				it.Linhas = append(it.Linhas, p.Caput)
			}
			it.Linhas = append(it.Linhas, p.Texto, origem(p))
			s.Itens = append(s.Itens, it)
		}
	}
	for _, p := range f.Base {
		s.Itens = append(s.Itens, workflow.ItemPainel{Titulo: p.Referencia, Situacao: "base do julgamento",
			Linhas: append(strings.Split(p.Texto, "\n"), origem(p))})
	}
	for _, l := range f.Lacunas {
		s.Itens = append(s.Itens, workflow.ItemPainel{Titulo: l, Situacao: "não localizado"})
	}
	return s, len(s.Itens) > 0
}
