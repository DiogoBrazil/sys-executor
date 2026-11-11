// Package legal é a biblioteca jurídica (V4 §40; plano §10): leitura
// estruturada das normas, acervo inicial, busca lexical e notas de vigência.
//
// Escopo atual (F5 essencial, decisão do usuário de 11/11/2026): acervo
// inicial de docs/law_library embutido no executável e busca somente lexical
// (V4 §47.7). A gestão do acervo pelo usuário, a detecção de duplicidade e a
// pesquisa web ficam para depois.
package legal

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// PaginaTexto é o texto de uma página do original (numeração a partir de 1).
type PaginaTexto struct {
	Numero int
	Texto  string
}

// Unidade é um dispositivo de norma: caput de artigo, parágrafo, inciso ou
// alínea, com a posição na hierarquia (V4 §40.3). Em documento sem artigos,
// é um trecho de página (Artigo vazio).
type Unidade struct {
	Artigo    string `json:"a,omitempty"` // "39", "83-A"
	Paragrafo string `json:"p,omitempty"` // "1", "unico"
	Inciso    string `json:"i,omitempty"` // "VII"
	Alinea    string `json:"l,omitempty"` // "a"
	Texto     string `json:"t"`
	Pagina    int    `json:"g"`
	// Contexto: títulos, capítulos e seções acima do artigo.
	Contexto string `json:"c,omitempty"`
	// Historica: redação substituída por outra posterior no mesmo documento.
	Historica bool `json:"h,omitempty"`
	// Revogada: o próprio texto registra a revogação.
	Revogada bool `json:"r,omitempty"`
	// Nota de alteração no fim do texto: "(Redação dada pelo Decreto…)".
	Nota string `json:"n,omitempty"`
}

// Parte é um diploma dentro do arquivo (a numeração dos artigos recomeça):
// o decreto que aprova e o regulamento aprovado, ou cada norma de uma
// coletânea.
type Parte struct {
	Titulo   string    `json:"titulo"`
	Pagina   int       `json:"pagina"`
	Unidades []Unidade `json:"unidades"`
}

// Artigos conta os artigos distintos da parte.
func (p Parte) Artigos() int {
	n, ult := 0, ""
	for _, u := range p.Unidades {
		if u.Artigo != "" && u.Artigo != ult {
			n, ult = n+1, u.Artigo
		}
	}
	return n
}

var (
	// "Art. 1º Texto", "Art. 32. Texto", "Art. 83-A. Texto", "Art. 2º - Texto".
	// Referência no começo de uma linha ("Art. 65, inciso V, da…") não casa:
	// depois do número vem vírgula ou palavra em minúscula.
	reArtigo = regexp.MustCompile(`^Art(?:igo)?\s*\.?\s*(\d{1,3})\s*[º°ªo]?\s*(?:-\s*([A-Z]))?\s*[.:]?\s*[-–—]?\s*(.*)$`)
	// "§ 1º Texto", "§ lº Texto" (l no lugar de 1 em alguns PDFs), "§ 10. Texto".
	reParagrafo = regexp.MustCompile(`^§\s*([0-9l]{1,2})\s*[º°ªo]?\s*\.?\s*[-–—]?\s*(.*)$`)
	reParUnico  = regexp.MustCompile(`^Par[áa]grafo\s+[úu]nico\s*\.?\s*[-–—]?\s*(.*)$`)
	reInciso    = regexp.MustCompile(`^([IVXLCDM]{1,8})\s*[-–—]\s*(.*)$`)
	reAlinea    = regexp.MustCompile(`^([a-z])\)\s*(.*)$`)
	reTitulo    = regexp.MustCompile(`(?i)^(LIVRO|PARTE|T[ÍI]TULO|CAP[ÍI]TULO|SUBSE[ÇC][ÃA]O|SE[ÇC][ÃA]O)\s+([IVXLC]+|[ÚU]NIC[OA])\b\s*[-–—.:]?\s*(.*)$`)
	reNota      = regexp.MustCompile(`(?i)\((?:nova\s+)?(?:reda[çc][ãa]o\s+dada|acrescid[oa]|acrescentad[oa]|inclu[íi]d[oa]|alterad[oa]|revogad[oa]|renumerad[oa])[^)]*\)`)
	reRevogado  = regexp.MustCompile(`(?i)\(\s*revogad[oa]`)
	reSoNumero  = regexp.MustCompile(`^\d{1,4}$`)
	// Linha que nomeia um diploma (título de uma parte).
	reDiploma = regexp.MustCompile(`(?i)^(DECRETO-LEI|DECRETO|LEI COMPLEMENTAR|LEI|RESOLU[ÇC][ÃA]O|PORTARIA|INSTRU[ÇC][ÃA]O|DIRETRIZ|REGULAMENTO|MANUAL|REGIMENTO)\b.{0,160}$`)
)

var nivelTitulo = map[string]int{"livro": 0, "parte": 0, "titulo": 1, "capitulo": 2, "secao": 3, "subsecao": 4}

type linha struct {
	texto  string
	pagina int
}

// comecaTexto: o que vem depois do número é o começo do dispositivo (e não
// a continuação de uma frase que o cita).
func comecaTexto(s string) bool {
	for _, r := range s {
		return unicode.IsUpper(r) || unicode.IsDigit(r) || strings.ContainsRune(`"“'(`, r)
	}
	return true // dispositivo com o texto na linha seguinte
}

func romanoValido(s string) bool {
	v := map[rune]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100, 'D': 500, 'M': 1000}
	total, ant := 0, 0
	for i := len(s) - 1; i >= 0; i-- {
		x := v[rune(s[i])]
		if x < ant {
			total -= x
		} else {
			total += x
		}
		ant = x
	}
	return total > 0 && total < 400
}

// linhasUteis junta as páginas, sem numeração de página nem cabeçalhos
// repetidos (linha igual em mais de um terço das páginas).
func linhasUteis(paginas []PaginaTexto) []linha {
	freq := map[string]int{}
	for _, p := range paginas {
		visto := map[string]bool{}
		for _, l := range strings.Split(p.Texto, "\n") {
			l = strings.TrimSpace(l)
			if l != "" && !visto[l] {
				visto[l] = true
				freq[l]++
			}
		}
	}
	corte := len(paginas)/3 + 2
	var out []linha
	for _, p := range paginas {
		for _, l := range strings.Split(p.Texto, "\n") {
			// O pdfium marca com U+0002 o hífen que cai no fim da linha
			// ("Comandante-/Geral"); nas normas ele é hífen de verdade.
			l = strings.TrimSpace(strings.NewReplacer("\r", "", "\x02", "-").Replace(l))
			if l == "" || reSoNumero.MatchString(l) || (len(paginas) >= 6 && freq[l] >= corte) {
				continue
			}
			out = append(out, linha{l, p.Numero})
		}
	}
	return out
}

// artigosAceitos decide quais linhas "Art. N" abrem artigo. A numeração tem
// de seguir a sequência; "Art. 1º" recomeça uma parte só quando o candidato
// seguinte é o 1º (outra redação) ou o 2º. O resto é citação no começo de
// uma linha.
func artigosAceitos(ls []linha) (aceito map[int]bool, reinicio map[int]bool) {
	type cand struct{ idx, n int }
	var cs []cand
	for i, l := range ls {
		if m := reArtigo.FindStringSubmatch(l.texto); m != nil && comecaTexto(m[3]) {
			n, _ := strconv.Atoi(m[1])
			cs = append(cs, cand{i, n})
		}
	}
	aceito, reinicio = map[int]bool{}, map[int]bool{}
	ult := 0
	for k, c := range cs {
		switch {
		case c.n == 1 && ult > 1:
			if k+1 < len(cs) && (cs[k+1].n == 1 || cs[k+1].n == 2) {
				aceito[c.idx], reinicio[c.idx] = true, true
				ult = 1
			}
		case ult == 0 && c.n <= 3, ult > 0 && c.n >= ult && c.n <= ult+5:
			aceito[c.idx] = true
			ult = c.n
		}
	}
	return aceito, reinicio
}

// Analisar lê o texto de uma norma e devolve as partes com os dispositivos.
// Documento sem artigos vira uma parte com um trecho por página.
func Analisar(paginas []PaginaTexto) []Parte {
	ls := linhasUteis(paginas)
	aceito, reinicio := artigosAceitos(ls)
	if len(aceito) == 0 {
		return trechos(paginas)
	}
	var partes []Parte
	parte := Parte{Pagina: 1}
	var contexto [5]string
	var atual *Unidade
	art, par, inc := "", "", ""
	fechar := func() {
		if atual != nil {
			finalizar(atual)
			parte.Unidades = append(parte.Unidades, *atual)
			atual = nil
		}
	}
	abrir := func(u Unidade, l linha) {
		fechar()
		u.Pagina = l.pagina
		u.Contexto = juntarContexto(contexto)
		atual = &u
	}
	for i := 0; i < len(ls); i++ {
		l := ls[i]
		if aceito[i] {
			m := reArtigo.FindStringSubmatch(l.texto)
			if reinicio[i] {
				fechar()
				if len(parte.Unidades) > 0 {
					partes = append(partes, parte)
				}
				titulo, pag := tituloDaParte(ls, i)
				parte = Parte{Titulo: titulo, Pagina: pag}
				contexto = [5]string{}
			}
			art, par, inc = m[1], "", ""
			if m[2] != "" {
				art += "-" + m[2]
			}
			abrir(Unidade{Artigo: art, Texto: m[3]}, l)
			continue
		}
		if m := reTitulo.FindStringSubmatch(l.texto); m != nil && tituloPlausivel(l.texto) {
			fechar()
			nome := strings.TrimSpace(m[3])
			// O nome costuma vir na linha seguinte ("CAPÍTULO IV" / "DO JULGAMENTO").
			if nome == "" && i+1 < len(ls) && !aceito[i+1] && linhaDeNome(ls[i+1].texto) {
				nome = ls[i+1].texto
				i++
			}
			rot := strings.TrimSpace(m[1] + " " + m[2])
			if nome != "" {
				rot += " — " + nome
			}
			n := nivelTitulo[chaveTitulo(m[1])]
			contexto[n] = rot
			for k := n + 1; k < len(contexto); k++ {
				contexto[k] = ""
			}
			continue
		}
		if art != "" {
			if m := reParagrafo.FindStringSubmatch(l.texto); m != nil && comecaTexto(m[2]) {
				par, inc = strings.ReplaceAll(m[1], "l", "1"), ""
				abrir(Unidade{Artigo: art, Paragrafo: par, Texto: m[2]}, l)
				continue
			}
			if m := reParUnico.FindStringSubmatch(l.texto); m != nil && comecaTexto(m[1]) {
				par, inc = "unico", ""
				abrir(Unidade{Artigo: art, Paragrafo: par, Texto: m[1]}, l)
				continue
			}
			if m := reInciso.FindStringSubmatch(l.texto); m != nil && romanoValido(m[1]) {
				inc = m[1]
				abrir(Unidade{Artigo: art, Paragrafo: par, Inciso: inc, Texto: m[2]}, l)
				continue
			}
			if m := reAlinea.FindStringSubmatch(l.texto); m != nil {
				abrir(Unidade{Artigo: art, Paragrafo: par, Inciso: inc, Alinea: m[1], Texto: m[2]}, l)
				continue
			}
		}
		if atual != nil {
			atual.Texto += " " + l.texto
		}
	}
	fechar()
	if len(parte.Unidades) > 0 {
		partes = append(partes, parte)
	}
	for k := range partes {
		marcarHistoricas(partes[k].Unidades)
		if partes[k].Titulo == "" {
			partes[k].Titulo, _ = tituloDaParte(ls, primeiraLinhaDoArtigo(ls, aceito))
		}
	}
	return partes
}

func primeiraLinhaDoArtigo(ls []linha, aceito map[int]bool) int {
	for i := range ls {
		if aceito[i] {
			return i
		}
	}
	return 0
}

func chaveTitulo(s string) string {
	s = strings.ToLower(s)
	for de, para := range map[string]string{"í": "i", "ç": "c", "ã": "a"} {
		s = strings.ReplaceAll(s, de, para)
	}
	return s
}

// tituloPlausivel evita tomar por título uma frase que começa com "Seção I
// do capítulo…" no meio de um parágrafo: título é curto e não termina em
// pontuação de frase.
func tituloPlausivel(s string) bool {
	return len([]rune(s)) <= 120 && !strings.HasSuffix(s, ",") && !strings.HasSuffix(s, ";")
}

// linhaDeNome: nome de título ("DO JULGAMENTO", "Dos Fatores a Observar").
func linhaDeNome(s string) bool {
	if len([]rune(s)) > 120 || reTitulo.MatchString(s) || reParagrafo.MatchString(s) || reInciso.MatchString(s) {
		return false
	}
	r := []rune(s)
	return len(r) > 0 && unicode.IsUpper(r[0]) && !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, ";") && !strings.HasSuffix(s, ":")
}

func juntarContexto(c [5]string) string {
	var ps []string
	for _, s := range c {
		if s != "" {
			ps = append(ps, s)
		}
	}
	return strings.Join(ps, " › ")
}

// tituloDaParte procura, antes do artigo 1º, a linha que nomeia o diploma.
func tituloDaParte(ls []linha, i int) (string, int) {
	for k := i - 1; k >= 0 && k >= i-40; k-- {
		t := ls[k].texto
		if reDiploma.MatchString(t) && strings.ToUpper(t) == t && !strings.HasSuffix(t, ",") {
			return strings.TrimRight(t, ". "), ls[k].pagina
		}
	}
	if i < len(ls) {
		return "", ls[i].pagina
	}
	return "", 1
}

func finalizar(u *Unidade) {
	u.Texto = strings.Join(strings.Fields(u.Texto), " ")
	if notas := reNota.FindAllString(u.Texto, -1); len(notas) > 0 {
		u.Nota = strings.Join(notas, " ")
	}
	u.Revogada = reRevogado.MatchString(u.Texto)
}

func (u Unidade) chave() string {
	return u.Artigo + "|" + u.Paragrafo + "|" + u.Inciso + "|" + u.Alinea
}

// dentroDe: v é a própria unidade u ou está abaixo dela na hierarquia.
func dentroDe(v, u Unidade) bool {
	if v.Artigo != u.Artigo {
		return false
	}
	if u.Paragrafo == "" && u.Inciso == "" && u.Alinea == "" {
		return true // caput: o artigo inteiro
	}
	if u.Inciso == "" && u.Alinea == "" {
		return v.Paragrafo == u.Paragrafo
	}
	if u.Alinea == "" {
		return v.Paragrafo == u.Paragrafo && v.Inciso == u.Inciso
	}
	return v.chave() == u.chave()
}

// marcarHistoricas: quando o mesmo dispositivo aparece de novo, a redação
// anterior (e o que estava abaixo dela) é histórica. Os textos consolidados
// trazem a redação antiga antes da atual.
func marcarHistoricas(us []Unidade) {
	ultima := map[string]int{}
	for i, u := range us {
		if j, ok := ultima[u.chave()]; ok {
			for k := j; k < i; k++ {
				if dentroDe(us[k], us[j]) {
					us[k].Historica = true
				}
			}
		}
		ultima[u.chave()] = i
	}
}

// trechos divide um documento sem artigos em um trecho por página.
func trechos(paginas []PaginaTexto) []Parte {
	p := Parte{Pagina: 1}
	for _, l := range paginas {
		t := strings.Join(strings.Fields(strings.ReplaceAll(l.Texto, "\x02", "-")), " ")
		if len([]rune(t)) < 40 {
			continue
		}
		p.Unidades = append(p.Unidades, Unidade{Texto: t, Pagina: l.Numero})
	}
	if len(p.Unidades) == 0 {
		return nil
	}
	return []Parte{p}
}
