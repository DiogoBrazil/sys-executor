package analise

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
)

// Proposta de enquadramento da SR (V4 §25.3; regra do usuário de 12/11/2026):
// para uma conclusão de indícios suficientes, a IA propõe o que foi
// indiciado, por subsunção dos fatos aos requisitos de cada infração:
//   - transgressão disciplinar: um inciso dos arts. 15, 16 ou 17 do RDPM ou
//     dos arts. 29 ou 32 do Estatuto (só do catálogo);
//   - crime militar: um tipo do CPM ou de outra lei penal; fora do CPM, só
//     nas hipóteses do art. 9º, inciso II, do CPM, com a alínea indicada;
//   - crime comum: um tipo do CP ou de outra lei penal (não do CPM).
// O aplicativo confere cada item na biblioteca (o chamador passa conferir) e
// o usuário confirma, troca ou acrescenta. Nada é citado de memória.

// Tipos de item (os mesmos de sr.TipoEnquadramento).
const (
	EnqTransgressao = "TRANSGRESSAO"
	EnqCrimeMilitar = "CRIME_MILITAR"
	EnqCrimeComum   = "CRIME_COMUM"
)

// TransgressaoCandidata é um item do catálogo oferecido à IA.
type TransgressaoCandidata struct {
	ID         string `json:"id"`
	Referencia string `json:"referencia"`
	Texto      string `json:"texto"`
}

// DiplomaPenal é uma lei penal da biblioteca que a IA pode indicar.
type DiplomaPenal struct {
	Sigla string `json:"sigla"`
	Nome  string `json:"nome"`
}

// DadosEnquadramentoSR é a conclusão confirmada a enquadrar.
type DadosEnquadramentoSR struct {
	FatoID      string       `json:"fato_id"`
	Fato        string       `json:"fato"`
	SindicadoID string       `json:"sindicado_id"`
	Sindicado   string       `json:"sindicado"`
	Conclusao   string       `json:"conclusao_confirmada"`
	Ocorrencia  string       `json:"ocorrencia_do_fato,omitempty"`
	Autoria     string       `json:"autoria,omitempty"`
	Fundamentos []Fundamento `json:"fundamentos_da_analise"`
}

// ItemEnquadramento é um possível crime ou transgressão proposto.
type ItemEnquadramento struct {
	Tipo string `json:"tipo"`
	// Transgressão: ID do catálogo.
	TransgressaoID string `json:"transgressao_id"`
	// Crime: sigla do diploma e o dispositivo.
	Diploma   string `json:"diploma"`
	Artigo    string `json:"artigo"`
	Paragrafo string `json:"paragrafo"`
	Inciso    string `json:"inciso"`
	Alinea    string `json:"alinea"`
	// Art9Alinea: alínea do art. 9º, II, do CPM (crime militar fora do CPM).
	Art9Alinea string   `json:"art9_alinea"`
	Fundamento string   `json:"fundamento"`
	Evid       []string `json:"evid_refs"`
}

type saidaEnquadramento struct {
	Itens []ItemEnquadramento `json:"itens"`
}

// ProporEnquadramentoSR pede a proposta. conferir confere um item na
// biblioteca e no catálogo; o erro volta à IA no reenvio. Sem resposta
// segura, devolve a pendência (o usuário informa o enquadramento).
func ProporEnquadramentoSR(ctx context.Context, x *Executor, dados DadosEnquadramentoSR, transgressoes []TransgressaoCandidata, diplomas []DiplomaPenal,
	art9 []string, pontos []Ponto, conferir func(ItemEnquadramento) error) ([]ItemEnquadramento, string, error) {
	if len(pontos) == 0 {
		return nil, "Sem pontos de prova sobre o fato, o assistente não propôs enquadramento: informe-o.", nil
	}
	ids := make([]string, 0, len(transgressoes)+1)
	ids = append(ids, "")
	for _, t := range transgressoes {
		ids = append(ids, t.ID)
	}
	siglas := []string{""}
	for _, d := range diplomas {
		siglas = append(siglas, d.Sigla)
	}
	limite := len(pontos)
	for {
		evs, docs, fontes, validos, _ := prepararPontos(pontos, limite)
		var res []ItemEnquadramento
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaEnquadramento, Alvo: "enquadramento " + dados.FatoID + " × " + dados.SindicadoID,
			Secoes: []policy.Secao{
				{Nome: "CONCLUSAO_CONFIRMADA", Dados: dados},
				{Nome: "TRANSGRESSOES_POSSIVEIS", Dados: transgressoes},
				{Nome: "DIPLOMAS_PENAIS", Dados: diplomas},
				{Nome: "ART_9_INCISO_II_DO_CPM", Dados: art9},
			},
			Evidencias: docs,
			Tarefa: "o usuário confirmou que há indícios suficientes quanto ao fato e ao sindicado de CONCLUSAO_CONFIRMADA. " +
				"Proponha o enquadramento do que foi indiciado, por subsunção: primeiro os fatos e as provas, depois os requisitos de cada infração; não escolha por mera semelhança de tema. " +
				"tipo TRANSGRESSAO: só um inciso de TRANSGRESSOES_POSSIVEIS (transgressao_id), quando a conduta contraria aquele inciso; deixe diploma e dispositivo vazios. " +
				"tipo CRIME_MILITAR: um tipo do CPM ou de outra lei de DIPLOMAS_PENAIS; fora do CPM, só quando o fato se enquadra numa das alíneas de ART_9_INCISO_II_DO_CPM, informada em art9_alinea. " +
				"tipo CRIME_COMUM: um tipo do CP ou de outra lei de DIPLOMAS_PENAIS, nunca do CPM; art9_alinea vazio. " +
				"Para crime, informe diploma (sigla), artigo (só o número, com a letra se houver: \"303\", \"7-B\") e, se for o caso, paragrafo (\"1\", \"unico\"), inciso (romano) e alinea; deixe transgressao_id vazio. " +
				"Indique só o que tem suporte concreto nas evidências; pode haver mais de um item (crime e transgressão juntos), sem somar itens irrelevantes. " +
				"Se houver dúvida real entre dois dispositivos, traga os dois e diga a dúvida no fundamento. " +
				"Em fundamento, uma ou duas frases sobre por que a conduta se enquadra, sem nomes de pessoas e sem identificadores (EV-…). Nunca invente artigo nem pena.",
			NomeSchema: "enquadramento_sr", Reserva: 1500,
			Schema: objeto(map[string]any{"itens": lista(objeto(map[string]any{
				"tipo":            enum(EnqTransgressao, EnqCrimeMilitar, EnqCrimeComum),
				"transgressao_id": enum(ids...),
				"diploma":         enum(siglas...),
				"artigo":          str(8), "paragrafo": str(8), "inciso": str(8), "alinea": str(2),
				"art9_alinea": enum("", "a", "b", "c", "d", "e"),
				"fundamento":  str(500),
				"evid_refs":   lista(enum(evs...), 1, 6),
			}), 1, 6)}),
		}, func(js []byte) error {
			var s saidaEnquadramento
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			var out []ItemEnquadramento
			for k, it := range s.Itens {
				campo := fmt.Sprintf("item %d", k+1)
				it.Fundamento = strings.TrimSpace(it.Fundamento)
				if it.Fundamento == "" {
					return fmt.Errorf("%s sem fundamento", campo)
				}
				if err := conferirTextoLivre(campo, it.Fundamento, fontes); err != nil {
					return err
				}
				if m := reIDInterno.FindString(it.Fundamento); m != "" {
					return fmt.Errorf("%s: o fundamento traz o identificador interno %q; use só evid_refs", campo, m)
				}
				for _, e := range it.Evid {
					if !validos[e] {
						return fmt.Errorf("%s cita evidência inexistente %q", campo, e)
					}
				}
				it.Evid = dedup(it.Evid)
				if err := conferir(it); err != nil {
					return fmt.Errorf("%s: %v", campo, err)
				}
				out = append(out, it)
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
				return nil, "Não foi possível propor o enquadramento com segurança: informe-o.", nil
			}
			return nil, "", err
		}
		return res, "", nil
	}
}
