// Package minuta monta o documento (DocModel do plano §12) a partir das
// decisões confirmadas, do cálculo oficial e dos trechos redigidos pela IA.
// A estrutura, os títulos e os textos fixos são do aplicativo (plano §5.5):
// a IA nunca escreve a decisão, a dosimetria, os pontos nem as providências.
package minuta

import "strings"

// APreencher marca um campo que o aplicativo não tem como preencher.
const APreencher = "[A PREENCHER]"

// Tipo de bloco do documento.
type Tipo string

const (
	Titulo     Tipo = "TITULO"     // título do documento
	Cabecalho  Tipo = "CABECALHO"  // linha de identificação (autoridade, acusado, referência)
	Secao      Tipo = "SECAO"      // "1. RELATÓRIO"
	Subtitulo  Tipo = "SUBTITULO"  // divisão dentro da seção
	Paragrafo  Tipo = "PARAGRAFO"  // parágrafo corrido
	Item       Tipo = "ITEM"       // inciso ou item de lista ("I – …")
	Citacao    Tipo = "CITACAO"    // texto de norma transcrito
	Fecho      Tipo = "FECHO"      // "CUMPRA-SE.", local e data
	Assinatura Tipo = "ASSINATURA" // espaço de assinatura, sem assinatura
)

// Origem do texto do bloco.
type Origem string

const (
	// Fixo: composto pelo aplicativo a partir de dados confirmados.
	Fixo Origem = "FIXO"
	// IA: redigido pela IA, com evidências conferidas.
	IA Origem = "IA"
)

// Bloco é uma unidade do documento.
type Bloco struct {
	Tipo   Tipo   `json:"tipo"`
	Texto  string `json:"texto"`
	Origem Origem `json:"origem"`
	// Trecho identifica o slot de redação ("relatorio", "fundamentacao:IMP-1",
	// "art36:IMP-1"), para os ajustes e a validação.
	Trecho string `json:"trecho,omitempty"`
	// Evid: evidências dos autos que sustentam o bloco (só na origem IA).
	Evid []string `json:"evid,omitempty"`
	// Pendente: o bloco tem campo a preencher ou não pôde ser redigido.
	Pendente bool `json:"pendente,omitempty"`
}

// Documento é a minuta montada.
type Documento struct {
	// Procedimento: "PADS" ou "SR".
	Procedimento string  `json:"procedimento"`
	Blocos       []Bloco `json:"blocos"`
	// Pendencias: o que o usuário precisa completar ou conferir.
	Pendencias []string `json:"pendencias"`
}

func (d *Documento) add(t Tipo, texto string) *Bloco {
	d.Blocos = append(d.Blocos, Bloco{Tipo: t, Texto: texto, Origem: Fixo, Pendente: strings.Contains(texto, APreencher)})
	return &d.Blocos[len(d.Blocos)-1]
}

func (d *Documento) pendencia(s string) {
	for _, p := range d.Pendencias {
		if p == s {
			return
		}
	}
	d.Pendencias = append(d.Pendencias, s)
}

// Pendencia acrescenta um item ao que o usuário precisa conferir.
func (d *Documento) Pendencia(s string) { d.pendencia(s) }

// CamposPendentes conta os blocos com campo a preencher.
func (d *Documento) CamposPendentes() int {
	n := 0
	for _, b := range d.Blocos {
		if b.Pendente {
			n++
		}
	}
	return n
}

// Texto devolve a minuta em texto simples (para conferência e testes).
func (d *Documento) Texto() string {
	var sb strings.Builder
	for _, b := range d.Blocos {
		switch b.Tipo {
		case Titulo, Secao:
			sb.WriteString("\n" + b.Texto + "\n\n")
		case Subtitulo:
			sb.WriteString("\n" + b.Texto + "\n")
		default:
			sb.WriteString(b.Texto + "\n")
		}
	}
	return strings.TrimSpace(sb.String()) + "\n"
}

var extenso = []string{"zero", "um", "dois", "três", "quatro", "cinco", "seis", "sete", "oito", "nove", "dez",
	"onze", "doze", "treze", "quatorze", "quinze", "dezesseis", "dezessete", "dezoito", "dezenove", "vinte",
	"vinte e um", "vinte e dois", "vinte e três", "vinte e quatro", "vinte e cinco", "vinte e seis", "vinte e sete",
	"vinte e oito", "vinte e nove", "trinta", "trinta e um", "trinta e dois", "trinta e três", "trinta e quatro",
	"trinta e cinco", "trinta e seis", "trinta e sete", "trinta e oito", "trinta e nove", "quarenta"}

// numeroExtenso: "2 (dois)"; fora da tabela, só o algarismo.
func numeroExtenso(n int) string {
	if n >= 0 && n < len(extenso) {
		return itoa(n) + " (" + extenso[n] + ")"
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

func plural(n int, um, varios string) string {
	if n == 1 {
		return um
	}
	return varios
}
