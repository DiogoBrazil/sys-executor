package analise

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
)

// Redação por trechos (etapa 12 do plano §5.4): a IA redige um trecho por
// chamada; a estrutura, os títulos e os textos fixos são do aplicativo. Cada
// parágrafo cita as evidências que o sustentam, e o Go confere os
// identificadores, as citações legais e os marcadores proibidos.

// Paragrafo é um parágrafo redigido pela IA, com as evidências de apoio.
type Paragrafo struct {
	Texto string   `json:"texto"`
	Evid  []string `json:"evid"`
}

// Trecho é o resultado de um trecho da minuta.
type Trecho struct {
	Paragrafos []Paragrafo `json:"paragrafos"`
	Lacunas    []string    `json:"lacunas"`
	// Pendencia: preenchida quando a IA não produziu o trecho com segurança
	// (a minuta leva um campo a preencher; nada é inventado).
	Pendencia string `json:"pendencia"`
}

type saidaTrecho struct {
	Paragrafos []saidaFund `json:"paragrafos"`
	Lacunas    []string    `json:"lacunas"`
}

type saidaFundamentacao struct {
	Imputacao  string      `json:"imputacao_id"`
	Resultado  string      `json:"resultado"`
	Paragrafos []saidaFund `json:"paragrafos"`
	Lacunas    []string    `json:"lacunas"`
}

type saidaArt36 struct {
	Fatores []struct {
		Inciso int      `json:"inciso"`
		Texto  string   `json:"texto"`
		Evid   []string `json:"evid_refs"`
	} `json:"fatores"`
}

func paragrafoSchema(evs []string, min int) map[string]any {
	return objeto(map[string]any{"texto": str(1200), "evid_refs": lista(enum(evs...), min, 10)})
}

// TRV-PRV-002: a minuta não cita página nem folha.
var reCitaPagina = regexp.MustCompile(`(?i)\b(p[áa]g(ina)?s?\.?|fls?\.?|folhas?)\s*(n[º°o.]*\s*)?\d`)

// PADS-RED-009: nunca afirmar ter visto ou ouvido mídia.
var reViuMidia = regexp.MustCompile(`(?i)\b(assisti|assistimos|visualizei|visualizamos|ouvi|ouvimos|ao assistir|ao visualizar|ao ouvir|verifiquei no v[íi]deo|vê-se no v[íi]deo|observa-se no v[íi]deo|nas imagens v[êe]-se)\b`)

// PADS-RED-010 / TRV-VAL-014: o art. 41, § 3º, do RDPM não é citado (D-01).
var reArt41Par3 = regexp.MustCompile(`(?i)(art(igo)?\.?\s*41\s*[º°]?\s*,?\s*(§|par[áa]grafo)\s*3|(§|par[áa]grafo)\s*3\s*[º°o]?\s*,?\s*do\s+art(igo)?\.?\s*41)`)

// identificadores internos que não podem aparecer no texto da minuta.
var reIDInterno = regexp.MustCompile(`\b(EV-\d+|IMP-\d+|PES-\d+|PEC-\d+)\b`)

// conferirParagrafos valida os parágrafos de um trecho. fontes são os textos
// que podem sustentar citação legal e endereço web (evidências, imputações e
// dispositivos recuperados da biblioteca).
func conferirParagrafos(campo string, ps []saidaFund, validos map[string]bool, fontes []string, exigirEvid bool) ([]Paragrafo, error) {
	var out []Paragrafo
	for k, p := range ps {
		t := strings.TrimSpace(p.Texto)
		if t == "" {
			return nil, fmt.Errorf("%s %d sem texto", campo, k+1)
		}
		if exigirEvid && len(p.Evid) == 0 {
			return nil, fmt.Errorf("%s %d sem evid_refs: cada parágrafo factual cita as evidências que o sustentam", campo, k+1)
		}
		for _, e := range p.Evid {
			if !validos[e] {
				return nil, fmt.Errorf("%s %d cita evidência inexistente %q", campo, k+1, e)
			}
		}
		if err := conferirTextoMinuta(fmt.Sprintf("%s %d", campo, k+1), t, fontes); err != nil {
			return nil, err
		}
		out = append(out, Paragrafo{Texto: t, Evid: dedup(p.Evid)})
	}
	return out, nil
}

// conferirTextoMinuta aplica as proibições de forma ao texto que vai à minuta.
func conferirTextoMinuta(campo, t string, fontes []string) error {
	if err := conferirTextoLivre(campo, t, fontes); err != nil {
		return err
	}
	if err := conferirCitacoesLegais(campo, t, fontes); err != nil {
		return err
	}
	return ConferirFormaMinuta(campo, t, true)
}

// ConferirFormaMinuta aplica as proibições de forma a um texto da minuta
// (também usada pela validação final, V4 §42). redigidoPelaIA acrescenta as
// que só valem para o texto da IA (página, mídia, marcador de preenchimento).
func ConferirFormaMinuta(campo, t string, redigidoPelaIA bool) error {
	if reArt41Par3.MatchString(t) {
		return fmt.Errorf("o campo %s cita o art. 41, § 3º, do RDPM, que não pode ser citado como dispositivo vigente", campo)
	}
	if m := reIDInterno.FindString(t); m != "" {
		return fmt.Errorf("o campo %s traz o identificador interno %q no texto; os identificadores vão só em evid_refs", campo, m)
	}
	if !redigidoPelaIA {
		return nil
	}
	if m := reCitaPagina.FindString(t); m != "" {
		return fmt.Errorf("o campo %s cita página ou folha (%q); cite a peça pelo nome e pelo número SEI, se houver", campo, m)
	}
	if m := reViuMidia.FindString(t); m != "" {
		return fmt.Errorf("o campo %s afirma observação direta de mídia (%q); use só a descrição que consta dos autos", campo, m)
	}
	if strings.Contains(t, "[A PREENCHER]") {
		return fmt.Errorf("o campo %s usa marcador de preenchimento; se faltar informação, registre em lacunas", campo)
	}
	return nil
}

// Reescrita pede uma nova versão de um trecho já redigido: para corrigir o
// que a validação apontou (V4 §42.2) ou para atender a um pedido de ajuste
// do usuário (V4 §31.3). nil = primeira redação.
type Reescrita struct {
	// Anterior: o texto atual do trecho.
	Anterior []string `json:"texto_atual"`
	// Problemas apontados pela validação automática.
	Problemas []string `json:"problemas_apontados,omitempty"`
	// Pedido do usuário, com as palavras dele.
	Pedido string `json:"pedido_do_usuario,omitempty"`
}

func (r *Reescrita) secoes() []policy.Secao {
	if r == nil {
		return nil
	}
	return []policy.Secao{{Nome: "TRECHO_EM_REESCRITA", Dados: r}}
}

func (r *Reescrita) tarefa() string {
	if r == nil {
		return ""
	}
	t := " ATENÇÃO: este trecho já foi redigido (texto_atual em TRECHO_EM_REESCRITA) e deve ser REESCRITO."
	if len(r.Problemas) > 0 {
		t += " Corrija os problemas_apontados: retire ou reformule o que as evidências citadas não sustentam, sem trocar por afirmação nova sem apoio."
	}
	if strings.TrimSpace(r.Pedido) != "" {
		t += " Atenda ao pedido_do_usuario somente no que as evidências sustentarem; ele não muda o resultado confirmado, a punição, os dias nem os pontos, e não autoriza citar norma fora das fontes."
	}
	return t + " Mantenha o que não foi apontado. Devolva o trecho inteiro, com todas as regras desta tarefa."
}

func prepararPontos(pontos []Ponto, limite int) (evs []string, docs []policy.Evidencia, fontes []string, validos map[string]bool, n int) {
	sel := selecionarPontos(pontos, limite)
	evs, docs, fontes = make([]string, len(sel)), make([]policy.Evidencia, len(sel)), make([]string, len(sel))
	validos = map[string]bool{}
	for i, p := range sel {
		evs[i], docs[i], fontes[i], validos[p.ID] = p.ID, p.Evidencia(), p.Trecho, true
	}
	return evs, docs, fontes, validos, len(sel)
}

func limparLacunas(ls []string, fontes []string) []string {
	var out []string
	for _, l := range ls {
		if t := strings.TrimSpace(l); t != "" && conferirTextoLivre("lacuna", t, fontes) == nil {
			out = append(out, t)
		}
	}
	return out
}

// PecaResumo identifica uma peça dos autos para o relatório.
type PecaResumo struct {
	Titulo string `json:"titulo"`
	SEI    string `json:"sei,omitempty"`
	Funcao string `json:"funcao,omitempty"`
}

// RedigirRelatorio redige os parágrafos do RELATÓRIO do PADS (PADS-RED-001):
// acusação, defesa prévia, provas colhidas, alegações finais e cronologia,
// sem antecipar culpa.
func RedigirRelatorio(ctx context.Context, x *Executor, congeladas policy.Secao, pecas []PecaResumo, pontos []Ponto, permitidos []string, re *Reescrita) (Trecho, error) {
	if len(pontos) == 0 {
		return Trecho{Pendencia: "Sem pontos de prova extraídos, o relatório dos fatos e das provas não pôde ser redigido."}, nil
	}
	limite := len(pontos)
	for {
		evs, docs, fontes, validos, _ := prepararPontos(pontos, limite)
		fontes = append(fontes, fmt.Sprint(congeladas.Dados))
		fontes = append(fontes, permitidos...)
		var res Trecho
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaRedacao, Alvo: "relatório",
			Secoes: append([]policy.Secao{congeladas, {Nome: "PECAS_DOS_AUTOS", Dados: pecas}}, re.secoes()...), Evidencias: docs,
			Tarefa: "redija os parágrafos do RELATÓRIO, nesta ordem: (1) os fatos atribuídos ao acusado, como descritos no Memorando Disciplinar; " +
				"(2) a defesa prévia, se houver; (3) as provas colhidas na instrução, peça por peça; (4) as alegações finais, se houver. " +
				"Relate o que consta dos autos e o que cada parte sustenta, sem avaliar a prova e sem antecipar culpa. " +
				"Não repita a lista de imputações nem os dispositivos: o aplicativo já os escreve. " +
				"Cite cada peça pelo nome e pelo número SEI, se houver; nunca cite página. Não escreva identificadores (EV-…, IMP-…) no texto." + re.tarefa(),
			NomeSchema: "relatorio_pads", Reserva: 3000,
			Schema: objeto(map[string]any{"paragrafos": lista(paragrafoSchema(evs, 1), 1, 14), "lacunas": lista(str(300), 0, 6)}),
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
				return Trecho{Pendencia: "Não foi possível redigir o relatório com segurança; o trecho fica a preencher."}, nil
			}
			return Trecho{}, err
		}
		return res, nil
	}
}

// DadosFundamentacao é o que a redação da fundamentação recebe sobre a
// imputação: o resultado já confirmado pelo usuário e a análise conferida.
type DadosFundamentacao struct {
	Imputacao   string       `json:"imputacao_id"`
	Conduta     string       `json:"conduta"`
	Dispositivo string       `json:"dispositivo"`
	Resultado   string       `json:"resultado_confirmado"`
	Fundamentos []Fundamento `json:"fundamentos_da_analise"`
	Teses       []Fundamento `json:"teses_da_defesa"`
	Lacunas     []string     `json:"lacunas,omitempty"`
}

// RedigirFundamentacao redige a fundamentação de UMA imputação (PADS-RED-002
// e 003), coerente com o resultado confirmado pelo usuário.
func RedigirFundamentacao(ctx context.Context, x *Executor, congeladas policy.Secao, dados DadosFundamentacao, pontos []Ponto, permitidos []string, re *Reescrita) (Trecho, error) {
	if len(pontos) == 0 {
		return Trecho{Pendencia: fmt.Sprintf("Sem pontos de prova sobre a imputação %s, a fundamentação fica a preencher.", dados.Imputacao)}, nil
	}
	limite := len(pontos)
	for {
		evs, docs, fontes, validos, _ := prepararPontos(pontos, limite)
		fontes = append(fontes, dados.Conduta, dados.Dispositivo, fmt.Sprint(congeladas.Dados))
		fontes = append(fontes, permitidos...)
		orient := "Explique por que a prova sustenta a autoria e a materialidade, e por que as teses da defesa não afastam a transgressão."
		if dados.Resultado == "IMPROCEDENTE" {
			orient = "Explique por que a prova não sustenta a imputação (insuficiência, contradição ou ausência de elemento da transgressão), aplicando o in dubio pro reo quando cabível."
		}
		var res Trecho
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaRedacao, Alvo: dados.Imputacao,
			Secoes: append([]policy.Secao{congeladas, {Nome: "IMPUTACAO_EM_REDACAO", Dados: dados}}, re.secoes()...), Evidencias: docs,
			Tarefa: fmt.Sprintf("redija a FUNDAMENTAÇÃO somente da imputação %s. O resultado_confirmado (%s) foi decidido pelo usuário: fundamente-o, sem alterá-lo. %s "+
				"Confronte os elementos favoráveis e desfavoráveis ao acusado e responda às teses da defesa. Não trate de punição, dias, pontos ou comportamento: isso é escrito pelo aplicativo. "+
				"Cite cada peça pelo nome e pelo número SEI, se houver; nunca cite página. Não escreva identificadores (EV-…, IMP-…) no texto.",
				dados.Imputacao, dados.Resultado, orient) + re.tarefa(),
			NomeSchema: "fundamentacao_imputacao", Reserva: 3000,
			Schema: objeto(map[string]any{"imputacao_id": enum(dados.Imputacao), "resultado": enum(dados.Resultado),
				"paragrafos": lista(paragrafoSchema(evs, 1), 1, 10), "lacunas": lista(str(300), 0, 6)}),
		}, func(js []byte) error {
			var s saidaFundamentacao
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			if s.Imputacao != dados.Imputacao || s.Resultado != dados.Resultado {
				return fmt.Errorf("imputacao_id deve ser %s e resultado deve ser %s (confirmado pelo usuário)", dados.Imputacao, dados.Resultado)
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
				return Trecho{Pendencia: fmt.Sprintf("Não foi possível redigir a fundamentação da imputação %s com segurança; o trecho fica a preencher.", dados.Imputacao)}, nil
			}
			return Trecho{}, err
		}
		return res, nil
	}
}

// FatorArt36 é a análise de um dos sete fatores do art. 36 do RDPM.
type FatorArt36 struct {
	Inciso int      `json:"inciso"`
	Texto  string   `json:"texto"`
	Evid   []string `json:"evid"`
}

// AusenciaDeElementos é o texto do fator sem apoio nos autos (V4 §6.1).
const AusenciaDeElementos = "Não há elementos nos autos sobre este fator."

// RedigirArt36 examina os sete fatores do art. 36 para uma imputação
// procedente (PADS-DOS-001). Devolve sempre os sete; o que a IA não
// sustentar com os autos fica como ausência de elementos.
func RedigirArt36(ctx context.Context, x *Executor, congeladas policy.Secao, calculo policy.Secao, imputacao string, fatores []string, pontos []Ponto, permitidos []string, re *Reescrita) ([]FatorArt36, string, error) {
	padrao := func() []FatorArt36 {
		out := make([]FatorArt36, 7)
		for i := range out {
			out[i] = FatorArt36{Inciso: i + 1, Texto: AusenciaDeElementos}
		}
		return out
	}
	if len(pontos) == 0 {
		return padrao(), "", nil
	}
	limite := len(pontos)
	for {
		evs, docs, fontes, validos, _ := prepararPontos(pontos, limite)
		fontes = append(fontes, fmt.Sprint(congeladas.Dados), fmt.Sprint(calculo.Dados), strings.Join(fatores, " "))
		fontes = append(fontes, permitidos...)
		res := padrao()
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaRedacao, Alvo: "art. 36 — " + imputacao,
			Secoes: append([]policy.Secao{congeladas, calculo, {Nome: "FATORES_DO_ART_36", Dados: fatores}}, re.secoes()...), Evidencias: docs,
			Tarefa: fmt.Sprintf("examine, para a imputação %s (procedente), cada um dos sete fatores listados em FATORES_DO_ART_36, na ordem (inciso 1 a 7). "+
				"Para cada fator, escreva de uma a três frases com base nas evidências e nos dados de CALCULO_OFICIAL (antecedentes e comportamento informados). "+
				"Se os autos não trouxerem elemento sobre o fator, escreva exatamente: %q e deixe evid_refs vazio. "+
				"Não repita o enunciado do fator, não proponha punição nem dias. Nunca cite página. Não escreva identificadores (EV-…, IMP-…) no texto.",
				imputacao, AusenciaDeElementos) + re.tarefa(),
			NomeSchema: "fatores_art36", Reserva: 2400,
			Schema: objeto(map[string]any{"fatores": lista(objeto(map[string]any{
				"inciso": inteiro(1, 7), "texto": str(700), "evid_refs": lista(enum(evs...), 0, 6)}), 7, 7)}),
		}, func(js []byte) error {
			var s saidaArt36
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			vistos := map[int]bool{}
			out := padrao()
			for _, f := range s.Fatores {
				if f.Inciso < 1 || f.Inciso > 7 || vistos[f.Inciso] {
					return fmt.Errorf("fatores deve trazer os incisos 1 a 7, um de cada")
				}
				vistos[f.Inciso] = true
				t := strings.TrimSpace(f.Texto)
				if t == "" {
					return fmt.Errorf("fator %d sem texto", f.Inciso)
				}
				for _, e := range f.Evid {
					if !validos[e] {
						return fmt.Errorf("fator %d cita evidência inexistente %q", f.Inciso, e)
					}
				}
				if err := conferirTextoMinuta(fmt.Sprintf("fator %d", f.Inciso), t, fontes); err != nil {
					return err
				}
				out[f.Inciso-1] = FatorArt36{Inciso: f.Inciso, Texto: t, Evid: dedup(f.Evid)}
			}
			if len(vistos) != 7 {
				return fmt.Errorf("fatores deve trazer os incisos 1 a 7, um de cada")
			}
			res = out
			return nil
		})
		if policy.IsFatiar(err) && limite > 4 {
			limite = limite * 2 / 3
			continue
		}
		if err != nil {
			if IsSemSeguranca(err) {
				return padrao(), "Não foi possível examinar os fatores do art. 36 com segurança; confira e complete cada um.", nil
			}
			return nil, "", err
		}
		return res, "", nil
	}
}
