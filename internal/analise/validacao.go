package analise

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
)

// Verificação de suporte (etapa 14 do plano §5.4; V4 §42.1, itens 1, 4 e 7):
// uma segunda passada da mesma IA confere se as evidências citadas sustentam
// cada parágrafo redigido e, na fundamentação, se o texto não contradiz o
// resultado confirmado pelo usuário.

// Suporte é o veredito de um parágrafo.
type Suporte string

const (
	Suporta    Suporte = "SUPORTA"
	Parcial    Suporte = "PARCIAL"
	NaoSuporta Suporte = "NAO_SUPORTA"
)

// Verificacao é o resultado da conferência de um parágrafo.
type Verificacao struct {
	// Indice do parágrafo na lista enviada (base 0).
	Indice  int
	Suporte Suporte
	// Contradiz: o parágrafo contraria o resultado confirmado pelo usuário.
	Contradiz bool
	Motivo    string
}

// Problema informa se o parágrafo precisa ser corrigido.
func (v Verificacao) Problema() bool { return v.Suporte == NaoSuporta || v.Contradiz }

type fraseAVerificar struct {
	N     int      `json:"n"`
	Texto string   `json:"texto"`
	Evid  []string `json:"evid_refs"`
}

type saidaVerificacao struct {
	Verificacoes []struct {
		N         int    `json:"n"`
		Suporte   string `json:"suporte"`
		Coerencia string `json:"coerencia"`
		Motivo    string `json:"motivo"`
	} `json:"verificacoes"`
}

// VerificarSuporte confere os parágrafos de um trecho. resultado é o
// resultado confirmado da imputação ("" quando o trecho não é fundamentação).
// Devolve uma verificação por parágrafo, na ordem. ErrSemSeguranca significa
// que a conferência não pôde ser feita (vira pendência, não aprovação).
func VerificarSuporte(ctx context.Context, x *Executor, alvo string, ps []Paragrafo, pontos []Ponto, resultado string) ([]Verificacao, error) {
	if len(ps) == 0 {
		return nil, nil
	}
	porID := map[string]Ponto{}
	for _, p := range pontos {
		porID[p.ID] = p
	}
	var frases []fraseAVerificar
	var docs []policy.Evidencia
	usado := map[string]bool{}
	for i, p := range ps {
		frases = append(frases, fraseAVerificar{N: i + 1, Texto: p.Texto, Evid: p.Evid})
		for _, id := range p.Evid {
			if pt, ok := porID[id]; ok && !usado[id] {
				usado[id] = true
				docs = append(docs, pt.Evidencia())
			}
		}
	}
	tarefa := "confira cada item de FRASES_A_VERIFICAR contra as evidências citadas em evid_refs dele (e só contra elas). " +
		"suporte: SUPORTA quando tudo o que o texto afirma como fato está nas evidências citadas; PARCIAL quando o essencial está, mas há detalhe sem apoio; " +
		"NAO_SUPORTA quando o texto afirma fato, data, fala ou conclusão de prova que as evidências citadas não trazem ou contradizem. " +
		"Valoração jurídica e conectivos não contam como fato. Em motivo, diga em uma linha o que falta ou o que diverge (vazio quando SUPORTA). " +
		"Não escreva identificadores (EV-…, IMP-…) no motivo."
	props := map[string]any{"n": inteiro(1, len(ps)), "suporte": enum(string(Suporta), string(Parcial), string(NaoSuporta)), "motivo": str(300)}
	secoes := []policy.Secao{{Nome: "FRASES_A_VERIFICAR", Dados: frases}}
	if resultado != "" {
		if x.Fluxo == policy.FluxoSR {
			tarefa += fmt.Sprintf(" As conclusões confirmadas pelo usuário para este trecho são: %s. coerencia: CONTRADIZ quando o texto conclui em sentido contrário a alguma delas (ou trata a conclusão como inconclusiva); COERENTE nos demais casos.", resultado)
		} else {
			tarefa += fmt.Sprintf(" O resultado desta imputação, confirmado pelo usuário, é %s. coerencia: CONTRADIZ quando o texto conclui em sentido contrário a esse resultado; COERENTE nos demais casos.", resultado)
		}
		props["coerencia"] = enum("COERENTE", "CONTRADIZ")
		secoes = append(secoes, policy.Secao{Nome: "RESULTADO_CONFIRMADO", Dados: resultado})
	}
	var out []Verificacao
	err := x.Chamar(ctx, Chamada{
		Etapa: catalog.EtapaValidacao, Alvo: alvo, Secoes: secoes, Evidencias: docs, Tarefa: tarefa,
		NomeSchema: "verificacao_suporte", Reserva: 200 + 120*len(ps),
		Schema: objeto(map[string]any{"verificacoes": lista(objeto(props), len(ps), len(ps))}),
	}, func(js []byte) error {
		var s saidaVerificacao
		if err := decodificarEstrito(js, &s); err != nil {
			return err
		}
		res := make([]Verificacao, len(ps))
		visto := map[int]bool{}
		for _, v := range s.Verificacoes {
			if v.N < 1 || v.N > len(ps) || visto[v.N] {
				return fmt.Errorf("verificacoes deve trazer um item para cada n de 1 a %d, sem repetir", len(ps))
			}
			visto[v.N] = true
			m := strings.TrimSpace(v.Motivo)
			if reIDInterno.MatchString(m) {
				m = reIDInterno.ReplaceAllString(m, "a evidência citada")
			}
			if Suporte(v.Suporte) != Suporta && m == "" {
				return fmt.Errorf("verificação %d: informe o motivo quando o suporte não é SUPORTA", v.N)
			}
			res[v.N-1] = Verificacao{Indice: v.N - 1, Suporte: Suporte(v.Suporte), Contradiz: v.Coerencia == "CONTRADIZ", Motivo: m}
		}
		if len(visto) != len(ps) {
			return fmt.Errorf("verificacoes deve trazer um item para cada n de 1 a %d", len(ps))
		}
		out = res
		return nil
	})
	if policy.IsFatiar(err) && len(ps) > 1 {
		meio := len(ps) / 2
		a, err := VerificarSuporte(ctx, x, alvo, ps[:meio], pontos, resultado)
		if err != nil {
			return nil, err
		}
		b, err := VerificarSuporte(ctx, x, alvo, ps[meio:], pontos, resultado)
		if err != nil {
			return nil, err
		}
		for i := range b {
			b[i].Indice += meio
		}
		return append(a, b...), nil
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Ajuste solicitado pelo usuário (V4 §31.3).

// TipoAjuste classifica o pedido.
type TipoAjuste string

const (
	// AjusteRedacao: muda o texto de um ou mais trechos redigidos pela IA.
	AjusteRedacao TipoAjuste = "REDACAO"
	// AjusteDecisao: muda algo já confirmado (resultado, punição, dias,
	// pontos, circunstâncias). Não é aplicado pela IA.
	AjusteDecisao TipoAjuste = "DECISAO"
	// AjusteTextoFixo: muda texto padrão composto pelo aplicativo.
	AjusteTextoFixo TipoAjuste = "TEXTO_FIXO"
	// AjusteIndefinido: não deu para saber o que mudar.
	AjusteIndefinido TipoAjuste = "INDEFINIDO"
)

// TrechoAjustavel descreve à IA um trecho que pode ser reescrito.
type TrechoAjustavel struct {
	ID        string `json:"id"`
	Descricao string `json:"descricao"`
}

// ClassificacaoAjuste é a leitura do pedido.
type ClassificacaoAjuste struct {
	Tipo    TipoAjuste
	Trechos []string
}

type saidaAjuste struct {
	Tipo    string   `json:"tipo"`
	Trechos []string `json:"trechos"`
}

// ClassificarAjuste diz o que o pedido do usuário atinge. O pedido não muda
// decisões: quem as muda é o usuário, nas respostas (plano §6).
func ClassificarAjuste(ctx context.Context, x *Executor, pedido string, trechos []TrechoAjustavel) (ClassificacaoAjuste, error) {
	ids := make([]string, len(trechos))
	val := map[string]bool{}
	for i, t := range trechos {
		ids[i], val[t.ID] = t.ID, true
	}
	if len(ids) == 0 {
		return ClassificacaoAjuste{Tipo: AjusteIndefinido}, nil
	}
	var out ClassificacaoAjuste
	err := x.Chamar(ctx, Chamada{
		Etapa: catalog.EtapaAjuste, Alvo: "pedido de ajuste",
		Secoes:     []policy.Secao{{Nome: "TRECHOS_AJUSTAVEIS", Dados: trechos}, {Nome: "PEDIDO_DO_USUARIO", Dados: pedido}},
		Tarefa:     tarefaClassificarAjuste(x.Fluxo),
		NomeSchema: "classificacao_ajuste", Reserva: 300,
		Schema: objeto(map[string]any{
			"tipo":    enum(string(AjusteRedacao), string(AjusteDecisao), string(AjusteTextoFixo), string(AjusteIndefinido)),
			"trechos": lista(enum(ids...), 0, len(ids))}),
	}, func(js []byte) error {
		var s saidaAjuste
		if err := decodificarEstrito(js, &s); err != nil {
			return err
		}
		c := ClassificacaoAjuste{Tipo: TipoAjuste(s.Tipo)}
		if c.Tipo == AjusteRedacao {
			for _, id := range dedup(s.Trechos) {
				if !val[id] {
					return fmt.Errorf("trecho inexistente %q", id)
				}
				c.Trechos = append(c.Trechos, id)
			}
			if len(c.Trechos) == 0 {
				return fmt.Errorf("com tipo REDACAO, liste em trechos ao menos um id de TRECHOS_AJUSTAVEIS")
			}
		}
		out = c
		return nil
	})
	if err != nil {
		if IsSemSeguranca(err) {
			return ClassificacaoAjuste{Tipo: AjusteIndefinido}, nil
		}
		return ClassificacaoAjuste{}, err
	}
	return out, nil
}

func tarefaClassificarAjuste(fluxo string) string {
	decisao := "DECISAO quando pede mudar resultado de imputação, punição, natureza, número de dias, pontos, comportamento, atenuantes, agravantes ou reincidência; "
	fixo := "TEXTO_FIXO quando trata só de cabeçalho, título, dispositivo da decisão, providências, fecho, assinatura ou formatação; "
	if fluxo == policy.FluxoSR {
		decisao = "DECISAO quando pede mudar a conclusão (indícios ou ausência de indícios), o enquadramento (crime ou transgressão), os fatos ou os sindicados; "
		fixo = "TEXTO_FIXO quando trata só de cabeçalho, título, dados da sindicância (seção I), solução, fecho, assinatura ou formatação; "
	}
	return "classifique o PEDIDO_DO_USUARIO; não reescreva nada agora. tipo: " +
		"REDACAO quando ele pede mudança de redação, complemento, correção ou supressão em trecho(s) de TRECHOS_AJUSTAVEIS (liste em trechos os ids atingidos; se o pedido for geral, liste todos os que ele atinge); " +
		decisao + fixo +
		"INDEFINIDO quando não dá para saber o que mudar. Se o pedido misturar redação com decisão, use o tipo DECISAO. Quando o tipo não for REDACAO, deixe trechos vazio."
}
