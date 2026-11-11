package workflow

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// Pergunta pontual que pausa a tarefa (V4 §31.2). Cada opção leva a um
// destino explícito: "" volta à etapa que perguntou (que lê a resposta),
// RecebendoUpload pede outro PDF, Cancelada encerra a tarefa. Campos (quando
// houver) são validados pelo Go antes de a etapa continuar.
type Pergunta struct {
	ID       string
	Titulo   string
	Contexto string
	Opcoes   []Opcao
	Campos   []Campo
	Grupos   []GrupoCampos
	// Revisavel: a resposta pode ser alterada depois (a etapa dona refaz o
	// que depende dela; F4-05).
	Revisavel bool
	// Resumo da resposta para a lista "Respostas dadas" (preenchido pela etapa).
	Resumir func(Resposta) string
	// Conferir valida o conjunto dos valores (ex.: ao menos uma imputação);
	// o erro aparece ao usuário como TAR-006.
	Conferir func(valores map[string]string) error
}

// Opcao de resposta.
type Opcao struct {
	ID      string
	Rotulo  string
	Destino Estado
	// Validar: a opção exige os campos preenchidos (ex.: "Confirmar").
	Validar bool
}

// TipoCampo de formulário.
type TipoCampo string

const (
	CampoTexto      TipoCampo = "TEXTO"
	CampoTextoLongo TipoCampo = "TEXTO_LONGO"
	CampoNumero     TipoCampo = "NUMERO"
	CampoData       TipoCampo = "DATA" // DD/MM/AAAA
	CampoEscolha    TipoCampo = "ESCOLHA"
	CampoSimNao     TipoCampo = "SIM_NAO"
	// CampoBusca: escolha numa lista longa, com filtro por digitação (ex.:
	// dispositivos do catálogo de enquadramento). Valida como ESCOLHA.
	CampoBusca TipoCampo = "BUSCA"
)

// Fonte aponta um trecho dos autos (abre o visualizador com realce).
type Fonte struct {
	Rotulo string `json:"rotulo"`
	Pagina int    `json:"pagina"`
	Inicio int    `json:"inicio"`
	Fim    int    `json:"fim"`
	Trecho string `json:"trecho"`
}

// Condicao de visibilidade: o campo só vale se outro campo tiver o valor.
type Condicao struct {
	Campo string `json:"campo"`
	Valor string `json:"valor"`
}

// Campo de formulário.
type Campo struct {
	ID          string       `json:"id"`
	Rotulo      string       `json:"rotulo"`
	Tipo        TipoCampo    `json:"tipo"`
	Opcoes      []OpcaoVisao `json:"opcoes"`
	Obrigatorio bool         `json:"obrigatorio"`
	Valor       string       `json:"valor"` // valor inicial (proposta da IA ou resposta anterior)
	Ajuda       string       `json:"ajuda"`
	Grupo       string       `json:"grupo"`
	Min         int          `json:"min"`
	Max         int          `json:"max"`
	Fontes      []Fonte      `json:"fontes"`
	MostrarSe   *Condicao    `json:"mostrarSe"`
}

// GrupoCampos agrupa campos na tela (ex.: uma imputação), com o contexto e
// as fontes que o usuário precisa ver para decidir.
type GrupoCampos struct {
	ID        string   `json:"id"`
	Titulo    string   `json:"titulo"`
	Descricao []string `json:"descricao"`
	Fontes    []Fonte  `json:"fontes"`
	Alertas   []string `json:"alertas"`
}

// Resposta do usuário a uma pergunta (DecisaoUsuario do plano §6).
type Resposta struct {
	Opcao   string            `json:"opcao"`
	Valores map[string]string `json:"valores"`
	Versao  int               `json:"versao"`
}

// Valor devolve o valor de um campo.
func (r Resposta) Valor(campo string) string { return r.Valores[campo] }

// PerguntaVisao é a pergunta enviada à interface.
type PerguntaVisao struct {
	ID       string        `json:"id"`
	Titulo   string        `json:"titulo"`
	Contexto string        `json:"contexto"`
	Opcoes   []OpcaoVisao  `json:"opcoes"`
	Campos   []Campo       `json:"campos"`
	Grupos   []GrupoCampos `json:"grupos"`
}

// OpcaoVisao é uma opção exibida.
type OpcaoVisao struct {
	ID      string `json:"id"`
	Rotulo  string `json:"rotulo"`
	Validar bool   `json:"validar"`
}

func (p *Pergunta) visao() *PerguntaVisao {
	if p == nil {
		return nil
	}
	v := &PerguntaVisao{ID: p.ID, Titulo: p.Titulo, Contexto: p.Contexto, Campos: p.Campos, Grupos: p.Grupos}
	for _, o := range p.Opcoes {
		v.Opcoes = append(v.Opcoes, OpcaoVisao{ID: o.ID, Rotulo: o.Rotulo, Validar: o.Validar})
	}
	return v
}

// visivel aplica MostrarSe.
// visivel: a condição do campo vale e o campo de que ele depende também
// está visível (condições encadeadas: "incluir" → "diploma" → "dispositivo").
func visivel(c Campo, valores map[string]string, porID map[string]Campo) bool {
	for prof := 0; c.MostrarSe != nil; prof++ {
		if prof > 8 || valores[c.MostrarSe.Campo] != c.MostrarSe.Valor {
			return false
		}
		pai, ok := porID[c.MostrarSe.Campo]
		if !ok {
			return true
		}
		c = pai
	}
	return true
}

// ValidarCampos confere obrigatoriedade e formato; devolve os valores
// normalizados só dos campos visíveis (TAR-006 em caso de problema).
func ValidarCampos(cs []Campo, valores map[string]string) (map[string]string, error) {
	out := map[string]string{}
	var problemas []string
	porID := make(map[string]Campo, len(cs))
	for _, c := range cs {
		porID[c.ID] = c
	}
	for _, c := range cs {
		if !visivel(c, valores, porID) {
			continue
		}
		v := strings.TrimSpace(valores[c.ID])
		if v == "" {
			if c.Obrigatorio {
				problemas = append(problemas, c.Rotulo+": obrigatório")
			}
			continue
		}
		switch c.Tipo {
		case CampoNumero:
			n, err := strconv.Atoi(v)
			if err != nil || (c.Min != 0 || c.Max != 0) && (n < c.Min || n > c.Max) {
				problemas = append(problemas, fmt.Sprintf("%s: número entre %d e %d", c.Rotulo, c.Min, c.Max))
				continue
			}
			v = strconv.Itoa(n)
		case CampoData:
			t, err := time.Parse("02/01/2006", v)
			if err != nil || t.Year() < 1900 || t.Year() > 2100 {
				problemas = append(problemas, c.Rotulo+": data no formato DD/MM/AAAA")
				continue
			}
		case CampoEscolha, CampoSimNao, CampoBusca:
			ok := false
			for _, o := range opcoesCampo(c) {
				ok = ok || o.ID == v
			}
			if !ok {
				problemas = append(problemas, c.Rotulo+": escolha uma das opções")
				continue
			}
		case CampoTexto, CampoTextoLongo:
			if len([]rune(v)) > 4000 {
				problemas = append(problemas, c.Rotulo+": texto longo demais")
				continue
			}
		}
		out[c.ID] = v
	}
	if len(problemas) > 0 {
		return nil, erros.Com(erros.TarefaRespostaInvalid, fmt.Errorf("%s", strings.Join(problemas, "; ")))
	}
	return out, nil
}

func opcoesCampo(c Campo) []OpcaoVisao {
	if c.Tipo == CampoSimNao && len(c.Opcoes) == 0 {
		return []OpcaoVisao{{ID: "sim", Rotulo: "Sim"}, {ID: "nao", Rotulo: "Não"}}
	}
	return c.Opcoes
}

// RespostaVisao é uma resposta já dada, listada para revisão.
type RespostaVisao struct {
	PerguntaID string `json:"perguntaId"`
	Titulo     string `json:"titulo"`
	Resumo     string `json:"resumo"`
	Revisavel  bool   `json:"revisavel"`
}

// PainelAnalise é a visão genérica do que a análise produziu (delimitação,
// pontos de prova, propostas), para o usuário acompanhar e conferir.
type PainelAnalise struct {
	Secoes []SecaoPainel `json:"secoes"`
}

// SecaoPainel agrupa itens.
type SecaoPainel struct {
	Titulo string       `json:"titulo"`
	Itens  []ItemPainel `json:"itens"`
}

// ItemPainel é uma linha do painel.
type ItemPainel struct {
	Titulo   string   `json:"titulo"`
	Situacao string   `json:"situacao"`
	Linhas   []string `json:"linhas"`
	Fontes   []Fonte  `json:"fontes"`
}

// BlocoMinuta é um bloco da minuta na interface (título, seção, parágrafo…).
type BlocoMinuta struct {
	Tipo  string `json:"tipo"`
	Texto string `json:"texto"`
	// Origem: "FIXO" (composto pelo aplicativo) ou "IA" (redigido pela IA).
	Origem   string  `json:"origem"`
	Pendente bool    `json:"pendente"`
	Fontes   []Fonte `json:"fontes"`
}

// MinutaVisao é a minuta mostrada ao usuário para revisão (V4 §31.3).
type MinutaVisao struct {
	Blocos          []BlocoMinuta `json:"blocos"`
	Pendencias      []string      `json:"pendencias"`
	CamposPendentes int           `json:"camposPendentes"`
	// Validada: passou pela validação automática (V4 §42) sem falha.
	Validada bool `json:"validada"`
	// Achados: falhas da validação que não puderam ser corrigidas sozinhas.
	Achados []string `json:"achados"`
	// Versao da minuta nesta tarefa (cresce a cada versão diferente).
	Versao int `json:"versao"`
	// Ajustes pedidos nesta tarefa e o que foi feito (só memória).
	Ajustes []AjusteVisao `json:"ajustes"`
}

// AjusteVisao é um pedido de ajuste e o seu resultado.
type AjusteVisao struct {
	Pedido    string `json:"pedido"`
	Resultado string `json:"resultado"`
	Aplicado  bool   `json:"aplicado"`
}

// ResumoAutos é o que a interface mostra sobre os autos lidos (cobertura,
// peças e mídias). Não contém texto dos autos.
type ResumoAutos struct {
	TotalPaginas    int            `json:"totalPaginas"`
	Paginas         []PaginaResumo `json:"paginas"`
	Pecas           []PecaResumo   `json:"pecas"`
	Midias          []MidiaResumo  `json:"midias"`
	Compatibilidade string         `json:"compatibilidade"`
	PecaCentral     string         `json:"pecaCentral"`
}

// PaginaResumo é uma linha do mapa de cobertura.
type PaginaResumo struct {
	Numero    int     `json:"numero"`
	Situacao  string  `json:"situacao"`
	Origem    string  `json:"origem"`
	Confianca float64 `json:"confianca"`
	Peca      string  `json:"peca"`
}

// PecaResumo é uma peça identificada.
type PecaResumo struct {
	ID            string `json:"id"`
	Titulo        string `json:"titulo"`
	Funcao        string `json:"funcao"`
	Rotulo        string `json:"rotulo"`
	PaginaInicial int    `json:"paginaInicial"`
	PaginaFinal   int    `json:"paginaFinal"`
	SEI           string `json:"sei"`
	Provavel      bool   `json:"provavel"`
	Externo       bool   `json:"externo"`
	Situacao      string `json:"situacao"` // pior situação de leitura entre as páginas
}

// MidiaResumo é uma menção a mídia não incorporada aos autos.
type MidiaResumo struct {
	Pagina int    `json:"pagina"`
	Peca   string `json:"peca"`
	Termo  string `json:"termo"`
	Inicio int    `json:"inicio"`
	Fim    int    `json:"fim"`
}
