package workflow

import (
	"os"
	"regexp"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// Visao é o retrato da tarefa enviado à interface. Não contém texto gerado
// pela IA nem conteúdo dos autos: só estado, etapas, progresso mensurável e
// erro do catálogo (V4 §31.1; T07).
type Visao struct {
	ID        string         `json:"id"`
	Tipo      Tipo           `json:"tipo"`
	Entrada   Entrada        `json:"entrada"`
	Estado    Estado         `json:"estado"`
	Descricao string         `json:"descricao"`
	Terminal  bool           `json:"terminal"`
	Etapas    []Marco        `json:"etapas"`
	Progresso *Progresso     `json:"progresso"` // nil quando não há métrica real
	Erro      *ErroVisao     `json:"erro"`      // terminal: motivo; ativa: alerta da última etapa
	Aviso     *ErroVisao     `json:"aviso"`     // ex.: limpeza incompleta (SES-003)
	Pergunta  *PerguntaVisao `json:"pergunta"`
	Pedido    *PedidoVisao   `json:"pedido"` // dados que o usuário precisa informar (SEI)
	Autos     *ResumoAutos   `json:"autos"`
	Parada    bool           `json:"parada"` // processamento parado aguardando versão futura (TAR-004)
}

// Situacao de uma etapa da linha do tempo.
type Situacao string

const (
	Pendente  Situacao = "pendente"
	Ativa     Situacao = "ativa"
	Concluida Situacao = "concluida"
	Falhou    Situacao = "falhou"
)

// Marco é uma etapa da linha do tempo exibida no processamento.
type Marco struct {
	Chave    string   `json:"chave"`
	Rotulo   string   `json:"rotulo"`
	Situacao Situacao `json:"situacao"`
}

// Progresso só existe quando há métrica real (páginas, peças) — V4 §31.1.
type Progresso struct {
	Atual   int    `json:"atual"`
	Total   int    `json:"total"`
	Unidade string `json:"unidade"`
}

// ErroVisao é a parte exibível de um erro do catálogo (sem a causa técnica).
type ErroVisao struct {
	Codigo   string `json:"codigo"`
	Titulo   string `json:"titulo"`
	Mensagem string `json:"mensagem"`
	Acao     string `json:"acao"`
	// Detalhe técnico, só no modo de diagnóstico (SYSEXEC_DIAGNOSTICO=1), com
	// dígitos mascarados. Vazio no uso normal (V4 §33).
	Detalhe string `json:"detalhe"`
}

var digitos = regexp.MustCompile(`[0-9]`)

// VisaoErro converte um erro do catálogo para a interface.
func VisaoErro(e *erros.Erro) *ErroVisao {
	if e == nil {
		return nil
	}
	v := &ErroVisao{Codigo: string(e.Codigo), Titulo: e.Titulo, Mensagem: e.Mensagem, Acao: e.Acao}
	if e.Causa != nil && os.Getenv("SYSEXEC_DIAGNOSTICO") == "1" {
		d := digitos.ReplaceAllString(e.Causa.Error(), "#")
		if r := []rune(d); len(r) > 300 {
			d = string(r[:300]) + "…"
		}
		v.Detalhe = d
	}
	return v
}

func visaoErro(e *erros.Erro) *ErroVisao { return VisaoErro(e) }

// grupo da linha do tempo (V4 §31.1: obtendo autos, extraindo texto/OCR,
// classificando peças, analisando provas, consultando fontes, elaborando,
// validando, aguardando revisão).
type grupo struct {
	chave, rotulo string
	estados       []Estado
	soPADS        bool
}

var linhaDoTempo = []grupo{
	{"obter", "Obtendo os autos", []Estado{EntradaEscolhida, SEIConectividade, SEILogin, SEIOTP, SEILocalizando, SEIReautenticacao, SEIGerandoPDF, SEIBaixando, RecebendoUpload, Verificando}, false},
	{"extrair", "Extraindo texto e OCR", []Estado{Extraindo}, false},
	{"classificar", "Classificando peças", []Estado{Classificando, Compatibilidade}, false},
	{"delimitar", "Delimitando o objeto", []Estado{Delimitando}, false},
	{"analisar", "Analisando provas", []Estado{Analisando}, false},
	{"fontes", "Consultando fontes jurídicas", []Estado{ConsultandoFontes}, false},
	{"calcular", "Calculando comportamento e sanção", []Estado{Calculando}, true},
	{"redigir", "Elaborando a minuta", []Estado{Redigindo, Ajustando}, false},
	{"validar", "Validando a minuta", []Estado{Validando}, false},
	{"revisar", "Aguardando sua revisão", []Estado{AguardandoRevisao, Finalizando, Finalizada}, false},
}

var descricoes = map[Estado]string{
	Nova:                     "Escolha o tipo de minuta.",
	TipoEscolhido:            "Escolha como obter os autos.",
	EntradaEscolhida:         "Preparando a obtenção dos autos.",
	SEIConectividade:         "Verificando a conexão com o SEI/RO.",
	SEILogin:                 "Entrando no SEI/RO.",
	SEIOTP:                   "Aguardando o código de autenticação em dois fatores.",
	SEILocalizando:           "Localizando o processo no SEI/RO.",
	SEIReautenticacao:        "O processo é sigiloso: o SEI/RO pede sua senha novamente.",
	SEIGerandoPDF:            "O SEI/RO está gerando o PDF do processo.",
	SEIBaixando:              "Baixando o PDF do processo.",
	RecebendoUpload:          "Selecione o PDF completo do processo.",
	Verificando:              "Conferindo o arquivo recebido.",
	Extraindo:                "Extraindo o texto das páginas.",
	Classificando:            "Identificando as peças dos autos.",
	Compatibilidade:          "Conferindo se os autos correspondem ao tipo escolhido.",
	Delimitando:              "Delimitando o objeto do procedimento.",
	Analisando:               "Analisando as provas.",
	ConsultandoFontes:        "Consultando a biblioteca jurídica.",
	Calculando:               "Calculando comportamento, reincidência e sanção.",
	Redigindo:                "Elaborando a minuta.",
	Validando:                "Validando a minuta.",
	AguardandoRevisao:        "A minuta está pronta para sua revisão.",
	Ajustando:                "Aplicando os ajustes solicitados.",
	Finalizando:              "Finalizando e descartando os arquivos temporários.",
	AguardandoEsclarecimento: "Aguardando sua resposta para continuar.",
	Finalizada:               "Tarefa finalizada.",
	Cancelada:                "Tarefa cancelada.",
	Falha:                    "A tarefa não pôde continuar.",
}

// marcos calcula a linha do tempo a partir do estado de referência (o estado
// ativo ou, em pausa/falha, o último estado de trabalho).
func marcos(tipo Tipo, ref Estado, estado Estado) []Marco {
	idx := -1
	for i, g := range linhaDoTempo {
		for _, e := range g.estados {
			if e == ref {
				idx = i
			}
		}
	}
	var out []Marco
	for i, g := range linhaDoTempo {
		if g.soPADS && tipo != TipoPADS {
			continue
		}
		s := Pendente
		switch {
		case idx < 0:
		case i < idx:
			s = Concluida
		case i == idx && estado == Falha:
			s = Falhou
		case i == idx && estado == Finalizada:
			s = Concluida
		case i == idx && estado != Cancelada:
			s = Ativa
		}
		out = append(out, Marco{Chave: g.chave, Rotulo: g.rotulo, Situacao: s})
	}
	return out
}
