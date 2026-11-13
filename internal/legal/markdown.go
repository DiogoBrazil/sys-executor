package legal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.yaml.in/yaml/v3"
)

// Normas em Markdown (decisão do usuário de 12/11/2026): o texto já vem
// estruturado, com um cabeçalho YAML de metadados, títulos com "#",
// dispositivos com "**Art. N.**", "**§ N**" e "**Parágrafo único.**", e as
// rubricas da norma em negrito numa linha própria ("**Peculato**"). Não há
// páginas: a localização de uma passagem é o próprio dispositivo.

// OrigemTexto são os metadados do cabeçalho do Markdown (V4 §40.3: fonte e
// versão do texto). Não provam vigência.
type OrigemTexto struct {
	Fonte       string `json:"fonte,omitempty"`
	VersaoTexto string `json:"versao_texto,omitempty"`
	BaixadoEm   string `json:"baixado_em,omitempty"`
}

var (
	reMdTitulo   = regexp.MustCompile(`^#{1,6}\s+(.*)$`)
	reMdNegrito  = regexp.MustCompile(`^\*\*([^*]+)\*\*$`)
	reMdArtigo   = regexp.MustCompile(`^\*\*\s*Art(?:igo)?\.?\s`)
	reMdMaiuscul = regexp.MustCompile(`\p{Ll}`)
)

// LerMarkdown lê uma norma em Markdown: os metadados do cabeçalho e as
// partes com os dispositivos.
func LerMarkdown(b []byte) (OrigemTexto, []Parte, error) {
	texto := strings.ReplaceAll(string(b), "\r", "")
	texto = strings.TrimPrefix(texto, "\ufeff")
	var origem OrigemTexto
	if strings.HasPrefix(texto, "---\n") {
		fim := strings.Index(texto[4:], "\n---")
		if fim < 0 {
			return origem, nil, fmt.Errorf("cabeçalho YAML sem fechamento")
		}
		var meta map[string]any
		if err := yaml.Unmarshal([]byte(texto[4:4+fim]), &meta); err != nil {
			return origem, nil, fmt.Errorf("cabeçalho YAML: %w", err)
		}
		origem = OrigemTexto{Fonte: campo(meta, "fonte"), VersaoTexto: campo(meta, "versao_texto"), BaixadoEm: campo(meta, "baixado_em")}
		texto = texto[4+fim+4:]
	}
	ls := linhasMarkdown(texto)
	aceito, reinicio := map[int]bool{}, map[int]bool{}
	ult := 0
	for i, l := range ls {
		if l.tipo != linhaComum || !l.artigo {
			continue
		}
		m := reArtigo.FindStringSubmatch(l.texto)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		aceito[i] = true
		if n == 1 && ult > 1 {
			reinicio[i] = true
		}
		ult = n
	}
	if len(aceito) == 0 {
		return origem, nil, fmt.Errorf("nenhum artigo marcado com **Art.**")
	}
	plano := make([]linha, len(ls))
	for i, l := range ls {
		plano[i] = l.linha
	}
	return origem, analisarLinhas(plano, aceito, reinicio), nil
}

func campo(meta map[string]any, k string) string {
	v, ok := meta[k]
	if !ok || v == nil {
		return ""
	}
	if t, ok := v.(time.Time); ok { // "baixado_em: 2026-10-13" vem como data
		return t.Format("2006-01-02")
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

type linhaMd struct {
	linha
	artigo bool // começava com "**Art."
}

// linhasMarkdown tira a marcação e classifica as linhas. Um negrito sozinho
// na linha é o nome do título logo acima dele, um título (em maiúsculas) ou
// a rubrica do dispositivo seguinte.
func linhasMarkdown(texto string) []linhaMd {
	var out []linhaMd
	aposTitulo := false
	for _, bruta := range strings.Split(texto, "\n") {
		bruta = strings.TrimSpace(bruta)
		if bruta == "" {
			continue
		}
		if strings.HasPrefix(bruta, ">") || bruta == "---" {
			aposTitulo = false
			continue // nota editorial do arquivo, não é texto da norma
		}
		if m := reMdTitulo.FindStringSubmatch(bruta); m != nil {
			t := limparMarkdown(m[1])
			out = append(out, linhaMd{linha: linha{texto: t, tipo: linhaTitulo}})
			// "## TÍTULO I" sem o nome: o negrito seguinte é o nome.
			mt := reTitulo.FindStringSubmatch(t)
			aposTitulo = mt != nil && strings.TrimSpace(mt[3]) == ""
			continue
		}
		l := linhaMd{linha: linha{texto: limparMarkdown(bruta)}, artigo: reMdArtigo.MatchString(bruta)}
		if l.texto == "" {
			continue
		}
		if m := reMdNegrito.FindStringSubmatch(bruta); m != nil && !l.artigo && !reParagrafo.MatchString(l.texto) && !reParUnico.MatchString(l.texto) {
			nome := strings.TrimSpace(m[1])
			switch {
			case aposTitulo || !reMdMaiuscul.MatchString(nome):
				l.tipo = linhaTitulo
			case !strings.HasSuffix(nome, ":"):
				l.tipo = linhaRubrica
			}
		}
		aposTitulo = false
		out = append(out, l)
	}
	// Rubrica sem negrito (como no CPM): linha curta, sem pontuação final,
	// logo acima de um artigo ou parágrafo.
	for i := 0; i+1 < len(out); i++ {
		if out[i].tipo == linhaComum && !out[i].artigo && out[i+1].tipo == linhaComum && abreDispositivo(out[i+1]) {
			if r, ok := rubricaSimples(out[i].texto); ok {
				out[i].texto, out[i].tipo = r, linhaRubrica
			}
		}
	}
	return out
}

func abreDispositivo(l linhaMd) bool {
	return l.artigo || reParagrafo.MatchString(l.texto) || reParUnico.MatchString(l.texto)
}

var reNotaFinal = regexp.MustCompile(`\s*\([^()]*\)\s*$`)

// rubricaSimples reconhece "Peculato" ou "Desacato a servidor público
// (Nome jurídico com redação dada…)" e devolve o nome sem a nota.
func rubricaSimples(s string) (string, bool) {
	r := strings.TrimSpace(reNotaFinal.ReplaceAllString(s, ""))
	rs := []rune(r)
	if len(rs) < 3 || len(rs) > 90 || !unicode.IsUpper(rs[0]) || strings.ContainsAny(r[len(r)-1:], ".;:,") {
		return "", false
	}
	if !reMdMaiuscul.MatchString(r) || strings.HasPrefix(r, "Pena") || reInciso.MatchString(r) || reAlinea.MatchString(r) || reTitulo.MatchString(r) {
		return "", false
	}
	return r, true
}

// limparMarkdown tira o negrito e o itálico ("*") e desfaz os escapes
// ("\-" vira "-").
func limparMarkdown(s string) string {
	var sb strings.Builder
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		switch {
		case rs[i] == '\\' && i+1 < len(rs) && strings.ContainsRune(`\-*_#>.()[]!+`, rs[i+1]):
			i++
			sb.WriteRune(rs[i])
		case rs[i] == '*':
		default:
			sb.WriteRune(rs[i])
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}
