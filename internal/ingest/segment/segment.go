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
	Capa                 Funcao = "CAPA"
	TermoJuntada         Funcao = "TERMO_JUNTADA"
	Representacao        Funcao = "REPRESENTACAO"
	Procuracao           Funcao = "PROCURACAO"
	Informacao           Funcao = "INFORMACAO"
	Email                Funcao = "EMAIL"
	Ciencia              Funcao = "CIENCIA" // comprovante de ciência de notificação/despacho
	Anexo                Funcao = "ANEXO"
	Indeterminada        Funcao = "INDETERMINADA"
)

// Base informa como a função foi decidida.
type Base string

const (
	BaseTitulo     Base = "TITULO"
	BaseCronologia Base = "CRONOLOGIA" // A01/A02: "provável", ver Provavel
	BaseNenhuma    Base = "NENHUMA"
	BaseIA         Base = "IA" // classificação residual pela IA (TRV-ING-001), sempre "provável"
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
	// Declarada: ID da peça no cadastro do usuário (BaseUsuario).
	Declarada string `json:"declarada,omitempty"`
}

var (
	reSEI         = regexp.MustCompile(`(?i)\bSEI\s*n[º°o.]*\s*:?\s*(\d{6,10})\b`)
	reVerificador = regexp.MustCompile(`(?i)c[óo]digo verificador\s*:?\s*(\d{6,10})`)
)

// regras de título, na ordem de prioridade (texto já sem acentos e minúsculo).
// A ordem importa: "Anexo - Ciência da defesa - Alegações finais" é ciência e
// "Notificação 64 - Alegações finais de defesa" é notificação (nomes reais da
// árvore do SEI/RO).
var regrasTitulo = []struct {
	re *regexp.Regexp
	f  Funcao
}{
	{regexp.MustCompile(`memorando\s+disciplinar`), MemorandoDisciplinar},
	{regexp.MustCompile(`portaria.{0,80}sindicancia|instaura.{0,40}sindicancia`), PortariaSR},
	{regexp.MustCompile(`^(pads\s+|sr\s+)?capa\b`), Capa}, // "Anexo … - CAPA" não é a capa do processo
	{regexp.MustCompile(`\bciencia\b`), Ciencia},
	{regexp.MustCompile(`notificacao|citacao|mandado|intimacao`), Notificacao},
	{regexp.MustCompile(`defesa\s+previa`), DefesaPrevia},
	{regexp.MustCompile(`alegacoes\s+finais|razoes\s+finais`), AlegacoesFinais},
	{regexp.MustCompile(`termo\s+de\s+(inquiricao|oitiva|depoimento)`), TermoInquiricao},
	{regexp.MustCompile(`termo\s+de\s+declarac`), TermoDeclaracoes},
	{regexp.MustCompile(`interrogatorio|termo\s+de\s+qualificacao`), Interrogatorio},
	{regexp.MustCompile(`\bata\b.{0,40}videoconferencia`), AtaVideoconferencia},
	{regexp.MustCompile(`(^|\s)ata\b`), Ata},
	{regexp.MustCompile(`termo\s+de\s+remessa`), TermoRemessa},
	{regexp.MustCompile(`termo\s+de\s+juntada`), TermoJuntada},
	{regexp.MustCompile(`(^|\s)representacao`), Representacao},
	{regexp.MustCompile(`(^|\s)procuracao`), Procuracao},
	{regexp.MustCompile(`(^|\s)informacao`), Informacao},
	{regexp.MustCompile(`\be-?mail\b|correio eletronico`), Email},
	{regexp.MustCompile(`(^|\s)portaria`), Portaria},
	{regexp.MustCompile(`(^|\s)despacho`), Despacho},
	{regexp.MustCompile(`(^|\s)oficio`), Oficio},
	{regexp.MustCompile(`(^|\s)certidao`), Certidao},
	{regexp.MustCompile(`(^|\s)relatorio`), Relatorio},
	{regexp.MustCompile(`manifestacao|peticao|requerimento`), Manifestacao},
}

// classificar aplica as regras a um nome (da árvore do SEI ou título).
func classificar(nome string) (Funcao, bool) {
	lc := texto.SemAcentos(nome)
	for _, r := range regrasTitulo {
		if r.re.MatchString(lc) {
			return r.f, true
		}
	}
	return "", false
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
		if f, ok := classificar(l); ok {
			pp.titulo, pp.funcao = limitar(l, 120), f
			// "PORTARIA Nº x" com "instaura sindicância" na linha seguinte.
			if f == Portaria && reSR.MatchString(texto.SemAcentos(strings.Join(cab, " "))) {
				pp.funcao = PortariaSR
			}
			return pp
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

// carimboDaPagina escolhe o carimbo deste processo (o dominante nos autos),
// preferindo o que tem "pg." igual ao número da página.
func carimboDaPagina(p autos.Pagina, processo string) (autos.Carimbo, bool) {
	var achado *autos.Carimbo
	for i := range p.Carimbos {
		c := p.Carimbos[i]
		if c.Processo != processo {
			continue
		}
		if c.PaginaPG == p.Numero {
			return c, true
		}
		achado = &p.Carimbos[i]
	}
	if achado != nil {
		return *achado, true
	}
	return autos.Carimbo{}, false
}

func processoDominante(a *autos.Autos) string {
	cont := map[string]int{}
	for _, p := range a.Paginas {
		for _, c := range p.Carimbos {
			cont[c.Processo]++
		}
	}
	melhor, n := "", 0
	for proc, k := range cont {
		if k > n || (k == n && proc < melhor) {
			melhor, n = proc, k
		}
	}
	return melhor
}

// Segmentar divide as páginas em peças e decide a função de cada uma. Com os
// carimbos do SEI (formato real), cada número SEI é uma peça e o nome da
// árvore define a função; sem carimbos, usa os sinais de reserva.
func Segmentar(a *autos.Autos) []Peca {
	if proc := processoDominante(a); proc != "" {
		return segmentarPorCarimbo(a, proc)
	}
	return segmentarPorSinais(a)
}

func segmentarPorCarimbo(a *autos.Autos, processo string) []Peca {
	var pecas []Peca
	var atual *Peca
	for _, p := range a.Paginas {
		c, ok := carimboDaPagina(p, processo)
		if !ok || (atual != nil && c.SEI == atual.SEI) {
			if atual != nil {
				atual.PaginaFinal = p.Numero // sem carimbo (ex.: página em branco) ou mesma peça
				continue
			}
			if !ok {
				c = autos.Carimbo{}
			}
		}
		titulo := c.Nome
		f, achou := classificar(c.Nome)
		if !achou {
			// Nome da árvore genérico ("Anexo …"): tenta o título no corpo.
			if pp := pistas(p); pp.funcao != "" {
				titulo, f, achou = pp.titulo, pp.funcao, true
			}
		}
		pecas = append(pecas, Peca{
			ID: fmt.Sprintf("PEC-%02d", len(pecas)+1), PaginaInicial: p.Numero, PaginaFinal: p.Numero,
			SEI: c.SEI, Titulo: limitar(titulo, 120), Funcao: f,
		})
		atual = &pecas[len(pecas)-1]
		if !achou {
			atual.Funcao = ""
		}
	}
	finalizar(pecas)
	return pecas
}

func finalizar(pecas []Peca) {
	for i := range pecas {
		switch {
		case pecas[i].Funcao == "" && (pecas[i].Externo || strings.HasPrefix(texto.SemAcentos(pecas[i].Titulo), "anexo")):
			pecas[i].Funcao, pecas[i].Base = Anexo, BaseNenhuma
		case pecas[i].Funcao == "":
			pecas[i].Funcao, pecas[i].Base = Indeterminada, BaseNenhuma
		default:
			pecas[i].Base = BaseTitulo
		}
	}
	resolverCronologia(pecas)
}

func segmentarPorSinais(a *autos.Autos) []Peca {
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
	finalizar(pecas)
	return pecas
}

// resolverCronologia decide "Manifestação" pela posição (D-09.5; A01/A02).
// Atos de instrução: termos de inquirição/declarações, interrogatório e atas
// de audiência por videoconferência (forma usada no SEI/RO). Antes do primeiro
// ato de instrução → defesa prévia provável; depois do
// último → alegações finais prováveis; entre termos → indeterminada (a IA ou
// o usuário decide).
func resolverCronologia(pecas []Peca) {
	primeiro, ultimo := -1, -1
	for i, p := range pecas {
		if p.Funcao == TermoInquiricao || p.Funcao == TermoDeclaracoes || p.Funcao == Interrogatorio || p.Funcao == AtaVideoconferencia {
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
