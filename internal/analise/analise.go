package analise

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
)

// Fundamento é uma frase da análise com as evidências que a sustentam.
type Fundamento struct {
	Texto string   `json:"texto"`
	Evid  []string `json:"evid"`
}

// CircunstanciaProposta é uma atenuante, agravante ou causa de justificação
// sugerida pela IA, a confirmar pelo usuário (PADS-DOS-002/003).
type CircunstanciaProposta struct {
	Artigo     int      `json:"artigo"`
	Inciso     int      `json:"inciso"`
	Alinea     string   `json:"alinea"`
	Fundamento string   `json:"fundamento"`
	Evid       []string `json:"evid"`
}

// AnaliseImputacao é a proposta da etapa 8 no PADS.
type AnaliseImputacao struct {
	ImputacaoID   string                  `json:"imputacaoId"`
	Resultado     string                  `json:"resultado"` // PROCEDENTE, IMPROCEDENTE ou "" (sem análise)
	Fundamentos   []Fundamento            `json:"fundamentos"`
	TesesDefesa   []Fundamento            `json:"tesesDefesa"`
	Atenuantes    []CircunstanciaProposta `json:"atenuantes"`
	Agravantes    []CircunstanciaProposta `json:"agravantes"`
	Justificacoes []CircunstanciaProposta `json:"justificacoes"`
	Lacunas       []string                `json:"lacunas"`
	Pontos        []string                `json:"pontos"` // EVs enviados
	Selecionados  int                     `json:"selecionados"`
	Disponiveis   int                     `json:"disponiveis"`
	Pendencias    []string                `json:"pendencias"`
}

type saidaFund struct {
	Texto string   `json:"texto"`
	Evid  []string `json:"evid_refs"`
}

type saidaCirc struct {
	Inciso     int      `json:"inciso"`
	Alinea     string   `json:"alinea"`
	Fundamento string   `json:"fundamento"`
	Evid       []string `json:"evid_refs"`
}

type saidaAnalisePADS struct {
	Imputacao     string      `json:"imputacao_id"`
	Resultado     string      `json:"resultado_proposto"`
	Fundamentos   []saidaFund `json:"fundamentos"`
	TesesDefesa   []saidaFund `json:"teses_defesa"`
	Atenuantes    []saidaCirc `json:"atenuantes"`
	Agravantes    []saidaCirc `json:"agravantes"`
	Justificacoes []saidaCirc `json:"justificacoes"`
	Lacunas       []string    `json:"lacunas"`
}

func fundSchema(evs []string) map[string]any {
	return objeto(map[string]any{"texto": str(700), "evid_refs": lista(enum(evs...), 1, 8)})
}

func circSchema(evs []string, min, max int, alinea bool) map[string]any {
	p := map[string]any{"inciso": inteiro(min, max), "fundamento": str(400), "evid_refs": lista(enum(evs...), 1, 6)}
	if alinea {
		p["alinea"] = str(2)
	}
	return objeto(p)
}

// Incisos que a IA pode sugerir: os de comportamento (38 I, 39 I), relevantes
// serviços (38 II), conexão (39 II) e reincidência (39 III) vêm do motor e
// das perguntas, nunca da IA.
func schemaAnalisePADS(imp string, evs []string) map[string]any {
	return objeto(map[string]any{
		"imputacao_id":       enum(imp),
		"resultado_proposto": enum("PROCEDENTE", "IMPROCEDENTE"),
		"fundamentos":        lista(fundSchema(evs), 1, 10),
		"teses_defesa":       lista(fundSchema(evs), 0, 6),
		"atenuantes":         lista(circSchema(evs, 3, 20, false), 0, 5),
		"agravantes":         lista(circSchema(evs, 4, 20, true), 0, 6),
		"justificacoes":      lista(circSchema(evs, 1, 20, false), 0, 4),
		"lacunas":            lista(str(300), 0, 6),
	})
}

func conferirFundamentos(campo string, fs []saidaFund, validos map[string]bool, fontes []string) ([]Fundamento, error) {
	var out []Fundamento
	for k, f := range fs {
		if strings.TrimSpace(f.Texto) == "" || len(f.Evid) == 0 {
			return nil, fmt.Errorf("%s %d sem texto ou sem evid_refs", campo, k+1)
		}
		for _, e := range f.Evid {
			if !validos[e] {
				return nil, fmt.Errorf("%s %d cita evidência inexistente %q", campo, k+1, e)
			}
		}
		if err := conferirTextoLivre(campo, f.Texto, fontes); err != nil {
			return nil, err
		}
		if err := conferirCitacoesLegais(campo, f.Texto, fontes); err != nil {
			return nil, err
		}
		out = append(out, Fundamento{Texto: strings.TrimSpace(f.Texto), Evid: dedup(f.Evid)})
	}
	return out, nil
}

func conferirCirc(campo string, artigo int, cs []saidaCirc, validos map[string]bool, fontes []string) ([]CircunstanciaProposta, error) {
	var out []CircunstanciaProposta
	for k, c := range cs {
		if strings.TrimSpace(c.Fundamento) == "" || len(c.Evid) == 0 {
			return nil, fmt.Errorf("%s %d sem fundamento específico ou sem evidência (NUNCA presuma)", campo, k+1)
		}
		for _, e := range c.Evid {
			if !validos[e] {
				return nil, fmt.Errorf("%s %d cita evidência inexistente %q", campo, k+1, e)
			}
		}
		if err := conferirTextoLivre(campo, c.Fundamento, fontes); err != nil {
			return nil, err
		}
		if err := conferirCitacoesLegais(campo, c.Fundamento, fontes); err != nil {
			return nil, err
		}
		out = append(out, CircunstanciaProposta{Artigo: artigo, Inciso: c.Inciso, Alinea: strings.ToLower(strings.TrimSpace(c.Alinea)),
			Fundamento: strings.TrimSpace(c.Fundamento), Evid: dedup(c.Evid)})
	}
	return out, nil
}

func dedup(xs []string) []string {
	var out []string
	vistos := map[string]bool{}
	for _, x := range xs {
		if !vistos[x] {
			vistos[x] = true
			out = append(out, x)
		}
	}
	return out
}

// selecionarPontos ajusta a lista ao orçamento (F4-06): primeiro tudo; sem
// espaço, tira os neutros; depois reduz mantendo o equilíbrio entre
// acusação e defesa, na ordem dos autos. A seleção aparece como pendência.
func selecionarPontos(ps []Ponto, limite int) []Ponto {
	if len(ps) <= limite {
		return ps
	}
	var acus, def, neu []Ponto
	for _, p := range ps {
		switch p.Sentido {
		case "ACUSACAO":
			acus = append(acus, p)
		case "DEFESA":
			def = append(def, p)
		default:
			neu = append(neu, p)
		}
	}
	var out []Ponto
	for i := 0; len(out) < limite && (i < len(acus) || i < len(def) || i < len(neu)); i++ {
		for _, g := range [][]Ponto{def, acus} {
			if i < len(g) && len(out) < limite {
				out = append(out, g[i])
			}
		}
	}
	for i := 0; len(out) < limite && i < len(neu); i++ {
		out = append(out, neu[i])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pagina != out[j].Pagina {
			return out[i].Pagina < out[j].Pagina
		}
		return out[i].Inicio < out[j].Inicio
	})
	return out
}

// AnalisarImputacao propõe resultado e fundamentos de UMA imputação (etapa 8).
// Sem pontos de prova, não chama a IA: a ausência vira pendência.
func AnalisarImputacao(ctx context.Context, x *Executor, imp AlvoExtracao, congeladas policy.Secao, pontos []Ponto, flags map[string]bool) (AnaliseImputacao, error) {
	r := AnaliseImputacao{ImputacaoID: imp.ID, Disponiveis: len(pontos)}
	if len(pontos) == 0 {
		r.Pendencias = append(r.Pendencias, "Nenhum ponto de prova sobre esta imputação foi encontrado nos autos; a análise fica com o usuário.")
		return r, nil
	}
	const reserva = 2400
	limite := len(pontos)
	for {
		sel := selecionarPontos(pontos, limite)
		evs := make([]string, len(sel))
		docs := make([]policy.Evidencia, len(sel))
		fontes := make([]string, len(sel))
		validos := map[string]bool{}
		for i, p := range sel {
			evs[i], docs[i], fontes[i], validos[p.ID] = p.ID, p.Evidencia(), p.Trecho, true
		}
		fontes = append(fontes, imp.Descricao, fmt.Sprint(congeladas.Dados))
		var res AnaliseImputacao
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaAnalise, Alvo: imp.ID, Flags: flags,
			Secoes: []policy.Secao{congeladas, {Nome: "IMPUTACAO_EM_ANALISE", Dados: imp}}, Evidencias: docs,
			Tarefa:     fmt.Sprintf("analise somente a imputação %s com base nas evidências acima.", imp.ID),
			NomeSchema: "analise_imputacao", Schema: schemaAnalisePADS(imp.ID, evs), Reserva: reserva,
		}, func(js []byte) error {
			var s saidaAnalisePADS
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			if s.Imputacao != imp.ID {
				return fmt.Errorf("imputacao_id deve ser %s", imp.ID)
			}
			if s.Resultado != "PROCEDENTE" && s.Resultado != "IMPROCEDENTE" {
				return fmt.Errorf("resultado_proposto inválido")
			}
			var err error
			res = AnaliseImputacao{ImputacaoID: imp.ID, Resultado: s.Resultado}
			if res.Fundamentos, err = conferirFundamentos("fundamento", s.Fundamentos, validos, fontes); err != nil {
				return err
			}
			if len(res.Fundamentos) == 0 {
				return fmt.Errorf("ao menos um fundamento com evidência é obrigatório")
			}
			if res.TesesDefesa, err = conferirFundamentos("tese de defesa", s.TesesDefesa, validos, fontes); err != nil {
				return err
			}
			if res.Atenuantes, err = conferirCirc("atenuante", 38, s.Atenuantes, validos, fontes); err != nil {
				return err
			}
			if res.Agravantes, err = conferirCirc("agravante", 39, s.Agravantes, validos, fontes); err != nil {
				return err
			}
			if res.Justificacoes, err = conferirCirc("causa de justificação", 37, s.Justificacoes, validos, fontes); err != nil {
				return err
			}
			for _, l := range s.Lacunas {
				if err := conferirTextoLivre("lacuna", l, fontes); err != nil {
					return err
				}
				if err := conferirCitacoesLegais("lacuna", l, fontes); err != nil {
					return err
				}
				if t := strings.TrimSpace(l); t != "" {
					res.Lacunas = append(res.Lacunas, t)
				}
			}
			return nil
		})
		if policy.IsFatiar(err) && limite > 4 {
			limite = limite * 2 / 3
			continue
		}
		if err != nil {
			if IsSemSeguranca(err) {
				r.Pendencias = append(r.Pendencias, fmt.Sprintf("Não foi possível analisar a imputação %s com segurança; a análise fica com o usuário.", imp.ID))
				return r, nil
			}
			return r, err
		}
		res.Pontos, res.Selecionados, res.Disponiveis = evs, len(sel), len(pontos)
		if len(sel) < len(pontos) {
			res.Pendencias = append(res.Pendencias, fmt.Sprintf("Por limite de memória, a análise considerou %d de %d pontos de prova (todos os da defesa e da acusação têm prioridade).", len(sel), len(pontos)))
		}
		return res, nil
	}
}

// ---- Sindicância Regular ----

// AnaliseFatoPessoa é a proposta da etapa 8 na SR (SR-SOL-003/004).
type AnaliseFatoPessoa struct {
	FatoID      string       `json:"fatoId"`
	PessoaID    string       `json:"pessoaId"`
	Conclusao   string       `json:"conclusao"` // INDICIOS, AUSENCIA, INCONCLUSIVO ou ""
	Ocorrencia  string       `json:"ocorrencia"`
	Autoria     string       `json:"autoria"`
	Fundamentos []Fundamento `json:"fundamentos"`
	Lacunas     []string     `json:"lacunas"`
	Diligencias []string     `json:"diligencias"`
	Pontos      []string     `json:"pontos"`
	Pendencias  []string     `json:"pendencias"`
}

type saidaAnaliseSR struct {
	Fato        string      `json:"fato_id"`
	Sindicado   string      `json:"sindicado_id"`
	Conclusao   string      `json:"conclusao_proposta"`
	Ocorrencia  string      `json:"ocorrencia_do_fato"`
	Autoria     string      `json:"autoria"`
	Fundamentos []saidaFund `json:"fundamentos"`
	Lacunas     []string    `json:"lacunas"`
	Diligencias []string    `json:"diligencias"`
}

func schemaAnaliseSR(fato, pessoa string, evs []string) map[string]any {
	return objeto(map[string]any{
		"fato_id": enum(fato), "sindicado_id": enum(pessoa),
		"conclusao_proposta": enum("INDICIOS", "AUSENCIA", "INCONCLUSIVO"),
		"ocorrencia_do_fato": enum("DEMONSTRADA", "NAO_DEMONSTRADA", "INCERTA"),
		"autoria":            enum("DEMONSTRADA", "NAO_DEMONSTRADA", "INCERTA"),
		"fundamentos":        lista(fundSchema(evs), 1, 10),
		"lacunas":            lista(str(300), 0, 6),
		"diligencias":        lista(str(300), 0, 6),
	})
}

// AnalisarFatoPessoa propõe a conclusão de um fato quanto a um sindicado.
func AnalisarFatoPessoa(ctx context.Context, x *Executor, fato, sindicado AlvoExtracao, objeto policy.Secao, pontos []Ponto) (AnaliseFatoPessoa, error) {
	r := AnaliseFatoPessoa{FatoID: fato.ID, PessoaID: sindicado.ID}
	if len(pontos) == 0 {
		r.Pendencias = append(r.Pendencias, "Nenhum ponto de prova sobre este fato foi encontrado nos autos; a conclusão fica com o usuário.")
		return r, nil
	}
	const reserva = 2200
	limite := len(pontos)
	for {
		sel := selecionarPontos(pontos, limite)
		evs := make([]string, len(sel))
		docs := make([]policy.Evidencia, len(sel))
		fontes := make([]string, len(sel))
		validos := map[string]bool{}
		for i, p := range sel {
			evs[i], docs[i], fontes[i], validos[p.ID] = p.ID, p.Evidencia(), p.Trecho, true
		}
		var res AnaliseFatoPessoa
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaAnalise, Alvo: fato.ID + "×" + sindicado.ID,
			Secoes:     []policy.Secao{objeto, {Nome: "FATO_EM_ANALISE", Dados: fato}, {Nome: "SINDICADO_EM_ANALISE", Dados: sindicado}},
			Evidencias: docs,
			Tarefa:     fmt.Sprintf("analise o fato %s quanto ao sindicado %s, com base nas evidências acima.", fato.ID, sindicado.ID),
			NomeSchema: "analise_fato_sindicado", Schema: schemaAnaliseSR(fato.ID, sindicado.ID, evs), Reserva: reserva,
		}, func(js []byte) error {
			var s saidaAnaliseSR
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			if s.Fato != fato.ID || s.Sindicado != sindicado.ID {
				return fmt.Errorf("fato_id e sindicado_id devem ser %s e %s", fato.ID, sindicado.ID)
			}
			if !em(s.Conclusao, "INDICIOS", "AUSENCIA", "INCONCLUSIVO") || !em(s.Ocorrencia, "DEMONSTRADA", "NAO_DEMONSTRADA", "INCERTA") ||
				!em(s.Autoria, "DEMONSTRADA", "NAO_DEMONSTRADA", "INCERTA") {
				return fmt.Errorf("conclusão, ocorrência ou autoria fora das opções do schema")
			}
			var err error
			res = AnaliseFatoPessoa{FatoID: fato.ID, PessoaID: sindicado.ID, Conclusao: s.Conclusao, Ocorrencia: s.Ocorrencia, Autoria: s.Autoria}
			if res.Fundamentos, err = conferirFundamentos("fundamento", s.Fundamentos, validos, fontes); err != nil {
				return err
			}
			if len(res.Fundamentos) == 0 {
				return fmt.Errorf("ao menos um fundamento com evidência é obrigatório")
			}
			if s.Conclusao == "INDICIOS" && s.Autoria == "NAO_DEMONSTRADA" {
				return fmt.Errorf("indícios suficientes exigem autoria demonstrada ou incerta, não 'não demonstrada'")
			}
			for _, l := range append(append([]string{}, s.Lacunas...), s.Diligencias...) {
				if err := conferirTextoLivre("lacuna/diligência", l, fontes); err != nil {
					return err
				}
			}
			res.Lacunas, res.Diligencias = limpar(s.Lacunas), limpar(s.Diligencias)
			return nil
		})
		if policy.IsFatiar(err) && limite > 4 {
			limite = limite * 2 / 3
			continue
		}
		if err != nil {
			if IsSemSeguranca(err) {
				r.Pendencias = append(r.Pendencias, fmt.Sprintf("Não foi possível analisar o fato %s quanto a %s com segurança.", fato.ID, sindicado.Descricao))
				return r, nil
			}
			return r, err
		}
		res.Pontos = evs
		if len(sel) < len(pontos) {
			res.Pendencias = append(res.Pendencias, fmt.Sprintf("Por limite de memória, a análise considerou %d de %d pontos de prova.", len(sel), len(pontos)))
		}
		return res, nil
	}
}

func limpar(xs []string) []string {
	var out []string
	for _, x := range xs {
		if t := strings.TrimSpace(x); t != "" {
			out = append(out, t)
		}
	}
	return out
}
