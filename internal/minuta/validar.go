package minuta

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
)

// Achado é uma falha encontrada pela validação determinística (V4 §42.1).
type Achado struct {
	// Regra: identificador curto do validador ("estrutura", "resultado"…).
	Regra    string
	Mensagem string
}

// assinatura ou ato oficial simulado (V4 §32.1).
var reAtoSimulado = regexp.MustCompile(`(?i)(assinado eletronicamente|documento assinado|assinatura eletr[ôo]nica de|publicado no (boletim|bi)\b|c[óo]digo verificador|c[óo]digo crc)`)

// ValidarPADS confere a minuta composta contra as decisões confirmadas e o
// cálculo oficial. evidencias são os identificadores que existem nos autos.
// Não substitui a revisão humana (V4 §42.1).
func ValidarPADS(d *Documento, e EntradaPADS, evidencias map[string]bool) []Achado {
	var out []Achado
	falha := func(regra, f string, a ...any) {
		out = append(out, Achado{Regra: regra, Mensagem: fmt.Sprintf(f, a...)})
	}

	// 1. Seções obrigatórias, na ordem (item 6).
	ordem := []string{"JULGAMENTO DE PADS", "1. RELATÓRIO", "2. FUNDAMENTAÇÃO", "3. DECISÃO"}
	pos, inicioDecisao := 0, -1
	for i, b := range d.Blocos {
		if (b.Tipo == Titulo || b.Tipo == Secao) && pos < len(ordem) && b.Texto == ordem[pos] {
			pos++
			if pos == len(ordem) {
				inicioDecisao = i
			}
		}
	}
	if pos != len(ordem) {
		falha("estrutura", "falta a parte %q ou as seções estão fora de ordem", ordem[pos])
		return out
	}
	decisao := d.Blocos[inicioDecisao:]
	textoDecisao := func() string {
		var sb strings.Builder
		for _, b := range decisao {
			sb.WriteString(b.Texto + "\n")
		}
		return sb.String()
	}()

	// 2. Resultado confirmado de cada imputação (itens 3 e 7): todas as do
	// memorando, e só elas.
	itens := 0
	for _, b := range decisao {
		if b.Tipo == Item && strings.Contains(b.Texto, "– imputação do ") {
			itens++
		}
	}
	if itens != len(e.Imputacoes) {
		falha("resultado", "a decisão lista %d imputação(ões), e o memorando tem %d", itens, len(e.Imputacoes))
	}
	procedentes := 0
	for _, im := range e.Imputacoes {
		r := "IMPROCEDENTE"
		if im.Procedente {
			r = "PROCEDENTE"
			procedentes++
		}
		if !contemBloco(decisao, Item, "imputação do "+im.Dispositivo+": "+r) {
			falha("resultado", "a decisão não traz a imputação do %s como %s", im.Dispositivo, r)
		}
		if !contemTrecho(d, "fundamentacao:"+im.ID) {
			falha("estrutura", "falta a fundamentação da imputação do %s", im.Dispositivo)
		}
	}
	global := pads.Improcedencia
	switch {
	case procedentes == len(e.Imputacoes) && procedentes > 0:
		global = pads.Procedencia
	case procedentes > 0:
		global = pads.ProcedenciaParcial
	}
	if e.Global != global || !strings.Contains(textoDecisao, "Julgar "+nomeGlobal[global]+" a acusação") {
		falha("resultado", "o resultado global da decisão não corresponde aos resultados confirmados (%s)", nomeGlobal[global])
	}

	// 3. Punição, dias e comportamento iguais aos do cálculo oficial (item 5).
	impostas := strings.Count(textoDecisao, "imponho a ")
	if impostas != len(e.Sancoes) {
		falha("calculo", "a decisão impõe %d punição(ões), e o cálculo oficial tem %d", impostas, len(e.Sancoes))
	}
	if len(e.Sancoes) == 0 {
		if procedentes > 0 {
			falha("calculo", "há imputação procedente sem punição calculada")
		}
		if procedentes == 0 && !strings.Contains(textoDecisao, "ABSOLVER ") {
			falha("resultado", "falta a absolvição na decisão")
		}
	}
	privativa := false
	for _, s := range e.Sancoes {
		esperado := "punição disciplinar de REPREENSÃO"
		if s.Sancao != engine.Repreensao {
			privativa = true
			esperado = fmt.Sprintf("punição disciplinar de %s %s de %s", numeroExtenso(s.Dias), plural(s.Dias, "dia", "dias"), nomeSancao[s.Sancao])
			if s.Dias <= 0 {
				esperado = "punição disciplinar de " + APreencher
			}
		}
		if !strings.Contains(textoDecisao, esperado+", por ter praticado a transgressão disciplinar do "+s.Dispositivo) {
			falha("calculo", "a punição escrita para o %s difere do cálculo oficial (%s)", s.Dispositivo, esperado)
		}
		if !strings.Contains(textoDecisao, "natureza "+nomeNatureza[s.Natureza]) {
			falha("calculo", "a natureza da transgressão do %s difere da confirmada", s.Dispositivo)
		}
		if !contemTrecho(d, "art36:"+s.ImputacaoID) {
			falha("estrutura", "falta o exame do art. 36 para o %s", s.Dispositivo)
		}
	}
	if len(e.Sancoes) > 0 && e.Pontuacao.Concluido {
		achou := false
		for _, b := range decisao {
			if b.Trecho == "comportamento" && strings.Contains(b.Texto, itoa(e.Pontuacao.PontosFinais)) &&
				strings.Contains(b.Texto, nomeComportamento[e.Pontuacao.ComportamentoFinal]) {
				achou = true
			}
		}
		if !achou {
			falha("calculo", "a pontuação ou o comportamento escritos diferem do cálculo oficial")
		}
	}

	// 4. Textos condicionais: cláusula de residência só com detenção ou
	// prisão (V4 §10); nota de punição só com punição (V4 §3.4).
	if tem := contemTrecho(d, "clausula_residencial"); tem != privativa {
		falha("texto_fixo", "a cláusula de cumprimento em residência só cabe com detenção ou prisão")
	}
	if tem := strings.Contains(textoDecisao, "nota de punição somente depois"); tem != (len(e.Sancoes) > 0) {
		falha("texto_fixo", "a providência da nota de punição só cabe quando há punição")
	}

	// 5. Proibições de forma e evidências (itens 1 e 2).
	for i, b := range d.Blocos {
		campo := fmt.Sprintf("bloco %d", i+1)
		if err := analise.ConferirFormaMinuta(campo, b.Texto, b.Origem == IA); err != nil {
			falha("forma", "%s", err.Error())
		}
		if reAtoSimulado.MatchString(b.Texto) && !strings.Contains(b.Texto, "data conforme a assinatura eletrônica") {
			falha("forma", "o %s sugere assinatura ou publicação já realizada", campo)
		}
		if b.Origem != IA {
			continue
		}
		if b.Tipo == Paragrafo && len(b.Evid) == 0 {
			falha("evidencia", "o %s, redigido pela IA, não cita evidência dos autos", campo)
		}
		for _, id := range b.Evid {
			if !evidencias[id] {
				falha("evidencia", "o %s cita evidência que não existe nos autos", campo)
			}
		}
	}
	return out
}

func contemBloco(bs []Bloco, t Tipo, trecho string) bool {
	for _, b := range bs {
		if b.Tipo == t && strings.Contains(b.Texto, trecho) {
			return true
		}
	}
	return false
}

func contemTrecho(d *Documento, slot string) bool {
	for _, b := range d.Blocos {
		if b.Trecho == slot {
			return true
		}
	}
	return false
}
