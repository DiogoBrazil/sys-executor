// Package segment divide os autos em peças e identifica a função de cada uma
// (plano §7 "Segmentação"; D-09.5; casos A01, A02, T04–T06, A08–A10).
//
// Tudo aqui é determinístico. A IA (Fase 4) só recebe as peças cuja função
// ficou indeterminada.
//
// Sinais de limite de peça (o formato real do PDF consolidado do SEI ainda
// será confirmado — pendência P-09 —, por isso os sinais são independentes):
//  1. número SEI no rodapé ("SEI nº 0000000") ou no código verificador;
//  2. título reconhecido no alto da página;
//  3. mudança entre página com e sem número SEI (documento externo/anexo).
package segment

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// Funcao documental de uma peça.
type Funcao string

const (
	MemorandoDisciplinar Funcao = "MEMORANDO_DISCIPLINAR"
	PortariaSR           Funcao = "PORTARIA_INSTAURACAO_SR"
	Portaria             Funcao = "PORTARIA"
	Notificacao          Funcao = "NOTIFICACAO"
	DefesaPrevia         Funcao = "DEFESA_PREVIA"
	AlegacoesFinais      Funcao = "ALEGACOES_FINAIS"
	TermoInquiricao      Funcao = "TERMO_INQUIRICAO"
	TermoDeclaracoes     Funcao = "TERMO_DECLARACOES"
	Interrogatorio       Funcao = "INTERROGATORIO"
	AtaVideoconferencia  Funcao = "ATA_VIDEOCONFERENCIA"
	Ata                  Funcao = "ATA"
	Despacho             Funcao = "DESPACHO"
	Oficio               Funcao = "OFICIO"
	Certidao             Funcao = "CERTIDAO"
	Relatorio            Funcao = "RELATORIO"
	TermoRemessa         Funcao = "TERMO_REMESSA"
	Manifestacao         Funcao = "MANIFESTACAO" // ambígua: resolvida pela cronologia
	Anexo                Funcao = "ANEXO"
	Indeterminada        Funcao = "INDETERMINADA"
)

// Base informa como a função foi decidida.
type Base string

const (
	BaseTitulo     Base = "TITULO"
	BaseCronologia Base = "CRONOLOGIA" // A01/A02: "provável", ver Provavel
	BaseNenhuma    Base = "NENHUMA"
)

// Peca é um documento dos autos.
type Peca struct {
	ID            string `json:"id"`
	PaginaInicial int    `json:"paginaInicial"` // base 1
	PaginaFinal   int    `json:"paginaFinal"`
	SEI           string `json:"sei"` // número SEI do documento, se houver
	Titulo        string `json:"titulo"`
	Funcao        Funcao `json:"funcao"`
	Base          Base   `json:"base"`
	Provavel      bool   `json:"provavel"` // função inferida (não declarada no título)
	Externo       bool   `json:"externo"`  // sem marcação SEI (anexo/escaneado)
}

var (
	reSEI         = regexp.MustCompile(`(?i)\bSEI\s*n[º°o.]*\s*:?\s*(\d{6,10})\b`)
	reVerificador = regexp.MustCompile(`(?i)c[óo]digo verificador\s*:?\s*(\d{6,10})`)
)

// regras de título, na ordem de prioridade (texto já sem acentos e minúsculo).
var regrasTitulo = []struct {
	re *regexp.Regexp
	f  Funcao
}{
	{regexp.MustCompile(`memorando\s+disciplinar`), MemorandoDisciplinar},
	{regexp.MustCompile(`portaria.{0,80}sindicancia|instaura.{0,40}sindicancia`), PortariaSR},
	{regexp.MustCompile(`defesa\s+previa`), DefesaPrevia},
	{regexp.MustCompile(`alegacoes\s+finais|razoes\s+finais`), AlegacoesFinais},
	{regexp.MustCompile(`termo\s+de\s+(inquiricao|oitiva|depoimento)`), TermoInquiricao},
	{regexp.MustCompile(`termo\s+de\s+declarac`), TermoDeclaracoes},
	{regexp.MustCompile(`interrogatorio|termo\s+de\s+qualificacao`), Interrogatorio},
	{regexp.MustCompile(`\bata\b.{0,40}videoconferencia`), AtaVideoconferencia},
	{regexp.MustCompile(`^\s*ata\b`), Ata},
	{regexp.MustCompile(`notificacao|citacao|mandado|intimacao`), Notificacao},
	{regexp.MustCompile(`termo\s+de\s+remessa`), TermoRemessa},
	{regexp.MustCompile(`^\s*portaria`), Portaria},
	{regexp.MustCompile(`^\s*despacho`), Despacho},
	{regexp.MustCompile(`^\s*oficio`), Oficio},
	{regexp.MustCompile(`^\s*certidao`), Certidao},
	{regexp.MustCompile(`^\s*relatorio`), Relatorio},
	{regexp.MustCompile(`manifestacao|peticao|requerimento`), Manifestacao},
}

// pistaPagina é o que se extrai de uma página para segmentar.
type pistaPagina struct {
	sei    string
	titulo string
	funcao Funcao
}

func pistas(p autos.Pagina) pistaPagina {
	var pp pistaPagina
	if m := reSEI.FindAllStringSubmatch(p.Texto, -1); len(m) > 0 {
		pp.sei = m[len(m)-1][1] // o rodapé fica no fim do texto da página
	} else if m := reVerificador.FindStringSubmatch(p.Texto); m != nil {
		pp.sei = m[1]
	}
	// Título: até as 3 primeiras linhas não vazias.
	var cab []string
	for _, l := range strings.Split(p.Texto, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			if cab = append(cab, l); len(cab) == 3 {
				break
			}
		}
	}
	for _, l := range cab {
		lc := texto.SemAcentos(l)
		for _, r := range regrasTitulo {
			if r.re.MatchString(lc) {
				pp.titulo, pp.funcao = limitar(l, 120), r.f
				// "PORTARIA Nº x" com "instaura sindicância" na linha seguinte.
				if r.f == Portaria && reSR.MatchString(texto.SemAcentos(strings.Join(cab, " "))) {
					pp.funcao = PortariaSR
				}
				return pp
			}
		}
	}
	return pp
}

var reSR = regexp.MustCompile(`portaria.{0,160}sindicancia|instaura.{0,60}sindicancia`)

func limitar(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// Segmentar divide as páginas em peças e decide a função de cada uma.
func Segmentar(a *autos.Autos) []Peca {
	var pecas []Peca
	var atual *Peca
	for _, p := range a.Paginas {
		pp := pistas(p)
		novo := atual == nil
		if atual != nil {
			switch {
			case pp.sei != "" && pp.sei != atual.SEI:
				novo = true
			case pp.sei == "" && atual.SEI != "" && p.Situacao != autos.Vazia:
				novo = true // passou de documento SEI para externo
			case pp.funcao != "" && pp.sei == "" && atual.SEI == "":
				novo = true // novo título dentro de um bloco externo
			}
		}
		if p.Situacao == autos.Vazia && atual != nil {
			atual.PaginaFinal = p.Numero // página em branco fica com a peça anterior
			continue
		}
		if novo {
			pecas = append(pecas, Peca{
				ID: fmt.Sprintf("PEC-%02d", len(pecas)+1), PaginaInicial: p.Numero, PaginaFinal: p.Numero,
				SEI: pp.sei, Titulo: pp.titulo, Funcao: pp.funcao, Externo: pp.sei == "",
			})
			atual = &pecas[len(pecas)-1]
			continue
		}
		atual.PaginaFinal = p.Numero
		if atual.Funcao == "" && pp.funcao != "" {
			atual.Titulo, atual.Funcao = pp.titulo, pp.funcao
		}
	}
	for i := range pecas {
		switch {
		case pecas[i].Funcao == "" && pecas[i].Externo:
			pecas[i].Funcao, pecas[i].Base = Anexo, BaseNenhuma
		case pecas[i].Funcao == "":
			pecas[i].Funcao, pecas[i].Base = Indeterminada, BaseNenhuma
		default:
			pecas[i].Base = BaseTitulo
		}
	}
	resolverCronologia(pecas)
	return pecas
}

// resolverCronologia decide "Manifestação" pela posição (D-09.5; A01/A02):
// antes do primeiro termo de inquirição → defesa prévia provável; depois do
// último → alegações finais prováveis; entre termos → indeterminada (a IA ou
// o usuário decide).
func resolverCronologia(pecas []Peca) {
	primeiro, ultimo := -1, -1
	for i, p := range pecas {
		if p.Funcao == TermoInquiricao || p.Funcao == TermoDeclaracoes || p.Funcao == Interrogatorio {
			if primeiro < 0 {
				primeiro = i
			}
			ultimo = i
		}
	}
	for i := range pecas {
		if pecas[i].Funcao != Manifestacao {
			continue
		}
		switch {
		case primeiro < 0:
			pecas[i].Funcao, pecas[i].Base = Indeterminada, BaseNenhuma
		case i < primeiro:
			pecas[i].Funcao, pecas[i].Base, pecas[i].Provavel = DefesaPrevia, BaseCronologia, true
		case i > ultimo:
			pecas[i].Funcao, pecas[i].Base, pecas[i].Provavel = AlegacoesFinais, BaseCronologia, true
		default:
			pecas[i].Funcao, pecas[i].Base = Indeterminada, BaseCronologia
		}
	}
}
