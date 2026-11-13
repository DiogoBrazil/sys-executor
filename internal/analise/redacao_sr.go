package analise

import (
	"context"
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
)

// Redação da SR (V4 §§23.5 e 24; §49): a IA redige só a síntese dos fatos
// (seção II) e a análise das provas de cada fato (seção III). A seção I e a
// SOLUÇÃO são compostas pelo aplicativo.

// RedigirSinteseSR redige a SÍNTESE DOS FATOS EM APURAÇÃO a partir da
// Portaria e dos documentos de origem, sem antecipar conclusão (V4 §23.5).
func RedigirSinteseSR(ctx context.Context, x *Executor, objSR policy.Secao, pecas []PecaResumo, pontos []Ponto, permitidos []string, re *Reescrita) (Trecho, error) {
	if len(pontos) == 0 {
		return Trecho{Pendencia: "Sem pontos de prova extraídos, a síntese dos fatos não pôde ser redigida."}, nil
	}
	limite := len(pontos)
	for {
		evs, docs, fontes, validos, _ := prepararPontos(pontos, limite)
		fontes = append(fontes, fmt.Sprint(objSR.Dados))
		fontes = append(fontes, permitidos...)
		var res Trecho
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaRedacao, Alvo: "síntese dos fatos",
			Secoes: append([]policy.Secao{objSR, {Nome: "PECAS_DOS_AUTOS", Dados: pecas}}, re.secoes()...), Evidencias: docs,
			Tarefa: "redija a SÍNTESE DOS FATOS EM APURAÇÃO: o que a Portaria de instauração manda apurar e o que os documentos de origem relatam, " +
				"em ordem cronológica, de modo objetivo. Não avalie a prova, não antecipe conclusão sobre indícios e não trate de enquadramento. " +
				"Cite cada peça pelo nome e pelo número SEI, se houver; nunca cite página. Não escreva identificadores (EV-…, F-…, S-…) no texto." + re.tarefa(),
			NomeSchema: "sintese_sr", Reserva: 2000,
			Schema: objeto(map[string]any{"paragrafos": lista(paragrafoSchema(evs, 1), 1, 6), "lacunas": lista(str(300), 0, 4)}),
		}, func(js []byte) error {
			var s saidaTrecho
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			ps, err := conferirParagrafos("parágrafo", s.Paragrafos, validos, fontes, true)
			if err != nil {
				return err
			}
			res = Trecho{Paragrafos: ps, Lacunas: limparLacunas(s.Lacunas, fontes)}
			return nil
		})
		if policy.IsFatiar(err) && limite > 4 {
			limite = limite * 2 / 3
			continue
		}
		if err != nil {
			if IsSemSeguranca(err) {
				return Trecho{Pendencia: "Não foi possível redigir a síntese dos fatos com segurança; o trecho fica a preencher."}, nil
			}
			return Trecho{}, err
		}
		return res, nil
	}
}

// ConclusaoEmRedacao é o que a análise de um fato recebe sobre cada
// sindicado: a conclusão confirmada pelo usuário e a análise conferida.
type ConclusaoEmRedacao struct {
	SindicadoID    string       `json:"sindicado_id"`
	Sindicado      string       `json:"sindicado"`
	Conclusao      string       `json:"conclusao_confirmada"` // INDICIOS ou AUSENCIA
	Enquadramentos []string     `json:"enquadramentos_confirmados,omitempty"`
	Fundamentos    []Fundamento `json:"fundamentos_da_analise"`
	Lacunas        []string     `json:"lacunas,omitempty"`
}

// DadosAnaliseFatoSR é o fato em redação.
type DadosAnaliseFatoSR struct {
	FatoID     string               `json:"fato_id"`
	Fato       string               `json:"fato"`
	Conclusoes []ConclusaoEmRedacao `json:"conclusoes"`
}

// RedigirAnaliseFatoSR redige a ANÁLISE DAS PROVAS de um fato (V4 §24): a
// prova da ocorrência e, depois, a conduta de cada sindicado,
// individualmente, coerente com a conclusão confirmada.
func RedigirAnaliseFatoSR(ctx context.Context, x *Executor, objSR policy.Secao, dados DadosAnaliseFatoSR, pontos []Ponto, permitidos []string, re *Reescrita) (Trecho, error) {
	if len(pontos) == 0 {
		return Trecho{Pendencia: fmt.Sprintf("Sem pontos de prova sobre o fato %s, a análise das provas fica a preencher.", dados.FatoID)}, nil
	}
	limite := len(pontos)
	for {
		evs, docs, fontes, validos, _ := prepararPontos(pontos, limite)
		fontes = append(fontes, dados.Fato, fmt.Sprint(objSR.Dados))
		for _, c := range dados.Conclusoes {
			fontes = append(fontes, c.Enquadramentos...)
		}
		fontes = append(fontes, permitidos...)
		var res Trecho
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaRedacao, Alvo: dados.FatoID,
			Secoes: append([]policy.Secao{objSR, {Nome: "FATO_EM_REDACAO", Dados: dados}}, re.secoes()...), Evidencias: docs,
			Tarefa: fmt.Sprintf("redija a análise das provas somente do fato %s. Primeiro, examine o que a prova mostra sobre a ocorrência do fato; "+
				"depois, examine a conduta de cada sindicado de conclusoes, um por vez, mesmo quando o resultado for igual. "+
				"A conclusao_confirmada de cada um foi decidida pelo usuário: fundamente-a, sem alterá-la. "+
				"se a conclusão for INDICIOS, mostre os elementos concretos que sustentam razoavelmente a possível infração e o vínculo do sindicado com a conduta, "+
				"e confronte-os com os requisitos dos enquadramentos_confirmados, sem juízo de condenação e sem tratar de sanção. "+
				"se a conclusão for AUSENCIA, explique por que os elementos não dão suporte razoável; se faltaram elementos ou persistiu dúvida, diga o que permanece sem esclarecimento, "+
				"sem afirmar que o fato não ocorreu. Não há resultado inconclusivo. Não sugira providências. "+
				"Cite cada peça pelo nome e pelo número SEI, se houver; nunca cite página. Não escreva identificadores (EV-…, F-…, S-…) no texto.", dados.FatoID) + re.tarefa(),
			NomeSchema: "analise_fato_sr", Reserva: 3500,
			Schema: objeto(map[string]any{"fato_id": enum(dados.FatoID),
				"paragrafos": lista(paragrafoSchema(evs, 1), 1, 14), "lacunas": lista(str(300), 0, 6)}),
		}, func(js []byte) error {
			var s struct {
				Fato       string      `json:"fato_id"`
				Paragrafos []saidaFund `json:"paragrafos"`
				Lacunas    []string    `json:"lacunas"`
			}
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			if s.Fato != dados.FatoID {
				return fmt.Errorf("fato_id deve ser %s", dados.FatoID)
			}
			ps, err := conferirParagrafos("parágrafo", s.Paragrafos, validos, fontes, true)
			if err != nil {
				return err
			}
			res = Trecho{Paragrafos: ps, Lacunas: limparLacunas(s.Lacunas, fontes)}
			return nil
		})
		if policy.IsFatiar(err) && limite > 4 {
			limite = limite * 2 / 3
			continue
		}
		if err != nil {
			if IsSemSeguranca(err) {
				return Trecho{Pendencia: fmt.Sprintf("Não foi possível redigir a análise das provas do fato %s com segurança; o trecho fica a preencher.", dados.FatoID)}, nil
			}
			return Trecho{}, err
		}
		return res, nil
	}
}
