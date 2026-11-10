package pseudonimo

// Filtros da revisão de possíveis nomes sem âncora (fase 2) e divisão de
// nomes ligados por "e". Tudo aqui só reduz falsos positivos de forma
// conservadora: na dúvida, o trecho continua indo à revisão do usuário.

import (
	"regexp"
	"strings"
	"unicode"
)

// dividirE separa "Fulano Silva e Beltrano Costa" em duas pessoas. Só
// divide quando os dois lados têm ao menos duas palavras de nome; senão o
// "e" faz parte do nome ("Ribeiro e Silva") ou o pedaço solto se perderia
// ("Renata Maria e Evelly" continua inteiro).
func dividirE(s string) []string {
	var partes [][]string
	atual := []string{}
	for _, w := range strings.Fields(s) {
		if w == "e" || w == "E" {
			partes = append(partes, atual)
			atual = []string{}
			continue
		}
		atual = append(atual, w)
	}
	partes = append(partes, atual)
	if len(partes) == 1 {
		return []string{s}
	}
	var out [][]string
	for _, p := range partes {
		if len(out) > 0 && (palavrasDeNome(p) < 2 || palavrasDeNome(out[len(out)-1]) < 2) {
			ult := append(out[len(out)-1], "e")
			out[len(out)-1] = append(ult, p...)
			continue
		}
		out = append(out, p)
	}
	res := make([]string, 0, len(out))
	for _, p := range out {
		if t := strings.Join(p, " "); t != "" {
			res = append(res, t)
		}
	}
	return res
}

func palavrasDeNome(ws []string) int {
	n := 0
	for _, w := range ws {
		if !eConector(w) {
			n++
		}
	}
	return n
}

// ruido: palavra que não pode ser nome, típica de OCR ruim ou sigla
// impronunciável. Três letras ou mais sem nenhuma vogal ("CNE" tem vogal e
// não entra; "BPM" entra) ou a mesma letra três vezes seguidas ("MULLLSL",
// "AAA"). Não usa o tamanho da palavra: "ANA DIAS", "EVA LIMA" e
// "RUI REIS" continuam sendo nomes. Nem sequência de consoantes:
// "Schmidt" e "Hirschfeld" são sobrenomes.
func ruido(w string) bool {
	letras := []rune{}
	for _, r := range w {
		if unicode.IsLetter(r) {
			letras = append(letras, unicode.ToLower(r))
		}
	}
	if len(letras) < 3 {
		return false
	}
	vogal := false
	for i, r := range letras {
		if strings.ContainsRune("aeiouyáàâãéèêíìîóòôõúùûü", r) {
			vogal = true
		}
		if i >= 2 && r == letras[i-1] && r == letras[i-2] {
			return true
		}
	}
	return !vogal
}

// Valor de um campo da qualificação que nunca é nome de pessoa
// ("Escolaridade: Ensino Médio", "Estado civil: União Estável",
// "Religião: Igreja …"). O valor vai até a vírgula, o ponto e vírgula, os
// dois-pontos, o ponto final, o "com" ("união estável com Fulana", que é
// âncora da fase 1) ou o fim da linha, com no máximo 40 caracteres.
var reValorQualificacao = regexp.MustCompile(`(?i:\b(?:escolaridade|grau[ \t]+de[ \t]+instru[çc][ãa]o|instru[çc][ãa]o|estado[ \t]+civil|` +
	`religi[ãa]o|profiss[ãa]o|ocupa[çc][ãa]o|naturalidade|nacionalidade|ve[íi]culo|cor|ra[çc]a|sexo|g[êe]nero)[ \t]*[:.\-–][ \t]*` +
	`|\bnatural[ \t]+de[ \t]+)` +
	`([^,;:\n]{1,40}?)(?:[ \t]+(?i:com)\b|[,;:\n]|\.(?:[ \t]|$)|$)`)

// mascararQualificacao troca o valor desses campos por "#" (mesmo tamanho
// em bytes), para que ele não forme candidato a nome.
func mascararQualificacao(linha string) string {
	ms := reValorQualificacao.FindAllStringSubmatchIndex(linha, -1)
	if ms == nil {
		return linha
	}
	b := []byte(linha)
	for _, m := range ms {
		for i := m[2]; i < m[3]; i++ {
			b[i] = '#'
		}
	}
	return string(b)
}

// frasesNaoNome: lugares com forma de nome de pessoa. Só valem quando o
// trecho inteiro é o lugar ("Campo Grande"); dentro de um trecho maior
// ("Fulano Rolim de Moura") nada é cortado e o trecho vai à revisão.
// A grafia é a que sobra no candidato: "Alta Floresta D'Oeste" chega como
// "Alta Floresta", e "Ji-Paraná" nunca junta as duas palavras.
var frasesNaoNome = map[string]bool{}

func init() {
	for _, f := range strings.Split(`Alta Floresta|Alto Alegre dos Parecis|Alto Paraíso|Campo Novo|Candeias do Jamari|`+
		`Colorado do Oeste|Costa Marques|Governador Jorge Teixeira|Itapuã do Oeste|Ministro Andreazza|Mirante da Serra|`+
		`Monte Negro|Nova Brasilândia|Nova Mamoré|Novo Horizonte do Oeste|Ouro Preto do Oeste|Pimenta Bueno|`+
		`Presidente Médici|Rio Crespo|Rolim de Moura|Santa Luzia|São Felipe|São Francisco do Guaporé|`+
		`São Miguel do Guaporé|Vale do Anari|Vale do Paraíso|Vista Alegre do Abunã|Nova Califórnia|União Bandeirantes|`+
		`Porto Velho|Rio Branco|Campo Grande|Belo Horizonte|Porto Alegre|Rio de Janeiro|São Paulo|São Luís|`+
		`João Pessoa|Boa Vista|Mato Grosso|Mato Grosso do Sul|Rio Grande do Sul|Rio Grande do Norte|Espírito Santo|`+
		`Santa Catarina|Minas Gerais|Distrito Federal`, "|") {
		frasesNaoNome[chaveNormal(f)] = true
	}
}

func fraseNaoNome(s string) bool {
	return frasesNaoNome[chaveNormal(aparConectores(s))]
}

// aparConectores tira conectores soltos nas pontas ("de Cujubim" → "Cujubim").
func aparConectores(s string) string {
	ws := strings.Fields(s)
	for len(ws) > 0 && eConector(ws[0]) {
		ws = ws[1:]
	}
	for len(ws) > 0 && eConector(ws[len(ws)-1]) {
		ws = ws[:len(ws)-1]
	}
	return strings.Join(ws, " ")
}

// temMinusculaForaConector: a linha tem minúscula numa palavra que não é
// conector. "NECESSIDADE e PROPORCIONALIDADE DA REAÇÃO" é título: o "e"
// minúsculo não conta.
func temMinusculaForaConector(linha string) bool {
	for _, w := range strings.Fields(linha) {
		if eConector(w) {
			continue
		}
		if strings.IndexFunc(w, unicode.IsLower) >= 0 {
			return true
		}
	}
	return false
}
