package minuta

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
)

// providência que a solução não traz por padrão (V4 §25.4).
var reProvidenciaAuto = regexp.MustCompile(`(?i)(sugiro (o )?arquivamento|sugere-se (o )?arquivamento|instaura[çc][ãa]o de pads|instaurar pads|encaminh(ar|e-se|amento) (à|a|ao) (corregedoria|autoridade|minist[ée]rio))`)

// ValidarSR confere o relatório composto contra as conclusões e os
// enquadramentos confirmados (V4 §42.1; §§23 a 25; §49). obj é o objeto da
// sindicância (fatos e sindicados admitidos); evidencias, os identificadores
// que existem nos autos. Não substitui a revisão humana.
func ValidarSR(d *Documento, e EntradaSR, obj *sr.ObjetoSR, evidencias map[string]bool) []Achado {
	var out []Achado
	falha := func(regra, f string, a ...any) {
		out = append(out, Achado{Regra: regra, Mensagem: fmt.Sprintf(f, a...)})
	}

	// 1. Título, as quatro seções e as seis alíneas, na ordem (V4 §23.1).
	ordem := []string{TituloSR, SecaoDadosSR, SecaoSinteseSR, SecaoAnaliseSR, SecaoSolucaoSR}
	pos, inicioSolucao, alinea := 0, -1, 0
	for i, b := range d.Blocos {
		if (b.Tipo == Titulo || b.Tipo == Secao) && pos < len(ordem) && b.Texto == ordem[pos] {
			pos++
			if pos == len(ordem) {
				inicioSolucao = i
			}
			continue
		}
		if pos == 2 && b.Tipo == Item && alinea < len(AlineasSR) && strings.HasPrefix(b.Texto, AlineasSR[alinea]) {
			alinea++
		}
	}
	if pos != len(ordem) {
		falha("estrutura", "falta a parte %q ou as seções estão fora de ordem", ordem[pos])
		return out
	}
	if alinea != len(AlineasSR) {
		falha("estrutura", "a seção I não traz a alínea %q na ordem", AlineasSR[alinea])
	}
	if !contemTrecho(d, SlotSinteseSR) {
		falha("estrutura", "falta a síntese dos fatos em apuração")
	}
	var sb strings.Builder
	for _, b := range d.Blocos[inicioSolucao:] {
		sb.WriteString(b.Texto + "\n")
	}
	solucao := sb.String()

	// 2. Conclusões no domínio da SR: fato admitido, sujeito sindicado, uma
	// por par, indícios com enquadramento e ausência sem (V4 §§24.3, 25; §49).
	var cs []sr.ConclusaoSR
	for _, f := range e.Fatos {
		if !contemTrecho(d, SlotAnaliseSR(f.ID)) {
			falha("estrutura", "falta a análise das provas do fato %s", f.ID)
		}
		for _, c := range f.Conclusoes {
			r := sr.Ausencia
			if c.Indicios {
				r = sr.Indicios
			}
			dc := sr.ConclusaoSR{FatoID: f.ID, PessoaID: c.SindicadoID, Resultado: r}
			for _, q := range c.Enquadramentos {
				fonte := "biblioteca"
				if q.Informado {
					fonte = "informado pelo usuário"
				}
				dc.Enquadramentos = append(dc.Enquadramentos, sr.Enquadramento{Tipo: q.Tipo, Diploma: q.Referencia, Artigo: q.Referencia, LegalPassageID: fonte})
			}
			cs = append(cs, dc)

			// 3. A solução diz o que foi confirmado, sobre quem foi confirmado.
			if !strings.Contains(solucao, c.Sindicado) {
				falha("solucao", "a solução não menciona o sindicado %s", c.Sindicado)
			}
			if !c.Indicios && !strings.Contains(strings.ToLower(solucao), "não foram identificados indícios suficientes") {
				falha("solucao", "a solução não registra a ausência de indícios confirmada para %s", c.Sindicado)
			}
			for _, q := range c.Enquadramentos {
				if !strings.Contains(solucao, q.Referencia) {
					falha("solucao", "a solução não traz o enquadramento %s confirmado para %s", q.Referencia, c.Sindicado)
				}
			}
		}
	}
	if obj != nil {
		for _, v := range obj.ValidarConclusoes(cs) {
			falha("solucao", "%s (%s)", v.Mensagem, v.Regra)
		}
		if n := len(obj.FatosAdmitidos()) * len(obj.Sindicados()); n != len(cs) {
			falha("solucao", "a solução tem %d conclusão(ões), e o objeto confirmado pede %d (um por fato e sindicado)", len(cs), n)
		}
	}
	if strings.Contains(strings.ToLower(solucao), "inconclusiv") {
		falha("solucao", "a solução usa resultado inconclusivo, que a SR não admite (V4 §49)")
	}
	if m := reProvidenciaAuto.FindString(solucao); m != "" {
		falha("solucao", "a solução traz providência que não foi pedida (%q); a V4 §25.4 não a admite por padrão", m)
	}

	// 4. Proibições de forma e evidências (iguais às do PADS).
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
			falha("evidencia", "o %s, redigido pelo assistente, não cita evidência dos autos", campo)
		}
		for _, id := range b.Evid {
			if !evidencias[id] {
				falha("evidencia", "o %s cita evidência que não existe nos autos", campo)
			}
		}
	}
	return out
}
