package pseudonimo

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// Mapa guarda, só em memória e só durante a tarefa, a correspondência entre
// grafias originais e marcadores. Nada disto é gravado nem enviado à IA.
type Mapa struct {
	mu       sync.Mutex
	marcador map[string]string // grafia exata → marcador
	forma    map[string]string // marcador → grafia exata
	pessoa   map[string]int    // chave normalizada da pessoa → número
	variante map[int]int       // número da pessoa → formas já usadas
	conta    map[Tipo]int      // próximo número por tipo (exceto pessoa)
	pessoas  int
	// Casamento sem diferença de caixa e acentos (TRV-SEG-011 v2): cada
	// grafia vira uma chave normalizada (tokens em minúsculas, sem acento);
	// as chaves formam a alternância procurada no texto normalizado.
	chave     map[string]Tipo  // chave normalizada → tipo
	liberados map[string]bool  // chaves que o usuário disse não serem dado pessoal
	re        *regexp.Regexp   // alternância das chaves (refeita quando surge chave nova)
	ancorados []*regexp.Regexp // cada chave ancorada, da mais longa à mais curta (2ª tentativa)
	// Nomes de pessoa de onde saem os pedaços ocultados por derivação.
	origens []origem
	fraca   map[string]bool // chaves de palavra solta derivada (fora da prova de resíduo)
	semDono map[int]bool    // números de pedaços comuns a mais de uma pessoa
}

// origem: nome de pessoa registrado. isolar diz se as palavras soltas dele
// também são ocultadas (cadastro e detecção com âncora; não o que o usuário
// digitou para ocultar, que pode não ser nome de pessoa).
type origem struct {
	forma  string
	isolar bool
}

// NovoMapa cria um mapa vazio.
func NovoMapa() *Mapa {
	return &Mapa{marcador: map[string]string{}, forma: map[string]string{}, pessoa: map[string]int{},
		variante: map[int]int{}, conta: map[Tipo]int{}, chave: map[string]Tipo{}, liberados: map[string]bool{},
		fraca: map[string]bool{}, semDono: map[int]bool{}}
}

// Aprender detecta dados pessoais no texto e os registra.
func (m *Mapa) Aprender(texto string) {
	ach := detectar(texto)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range ach {
		m.registrar(a.tipo, a.forma)
		if a.tipo == Pessoa {
			m.origens = append(m.origens, origem{a.forma, true})
			// A mesma pessoa em maiúsculas (cabeçalhos e assinaturas).
			if up := strings.ToUpper(a.forma); up != a.forma && strings.Contains(texto, up) {
				m.registrar(Pessoa, up)
			}
		}
	}
	m.re = nil
}

// AcrescentarNomes registra nomes ou dados informados pelo usuário na
// confirmação de envio (o que a detecção não pegou).
func (m *Mapa) AcrescentarNomes(itens []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range itens {
		s = strings.Join(strings.Fields(s), " ")
		if utf8.RuneCountInString(s) < 3 {
			continue
		}
		tp := Pessoa
		if strings.IndexFunc(s, unicode.IsLetter) < 0 {
			tp = Matricula // só dígitos e sinais: identificador numérico
		}
		m.registrar(tp, s)
		if tp == Pessoa {
			m.registrar(Pessoa, strings.ToUpper(s))
			m.origens = append(m.origens, origem{s, false})
		}
	}
	m.re = nil
}

// CadastrarPessoa registra um envolvido informado no cadastro de peças
// (V4 §48.4): o nome, a forma em maiúsculas e as outras grafias ou apelidos
// informados, todos como variantes da mesma pessoa ([PESSOA-NN.v]).
func (m *Mapa) CadastrarPessoa(nome string, grafias []string) {
	nome = strings.Join(strings.Fields(nome), " ")
	if utf8.RuneCountInString(nome) < 3 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registrar(Pessoa, nome)
	m.registrar(Pessoa, strings.ToUpper(nome))
	m.origens = append(m.origens, origem{nome, true})
	n := m.pessoa[chavePessoa(nome)]
	for _, g := range grafias {
		g = strings.Join(strings.Fields(g), " ")
		if utf8.RuneCountInString(g) < 3 {
			continue
		}
		if _, ok := m.pessoa[chavePessoa(g)]; !ok {
			m.pessoa[chavePessoa(g)] = n // a grafia é da mesma pessoa
		}
		m.registrar(Pessoa, g)
		m.registrar(Pessoa, strings.ToUpper(g))
		m.origens = append(m.origens, origem{g, true})
	}
	m.re = nil
}

func (m *Mapa) registrar(tp Tipo, forma string) {
	if _, ok := m.marcador[forma]; ok || forma == "" {
		return
	}
	var mk string
	if tp == Pessoa {
		k := chavePessoa(forma)
		n, ok := m.pessoa[k]
		if !ok {
			m.pessoas++
			n = m.pessoas
			m.pessoa[k] = n
		}
		m.variante[n]++
		if v := m.variante[n]; v == 1 {
			mk = fmt.Sprintf("[PESSOA-%02d]", n)
		} else {
			mk = fmt.Sprintf("[PESSOA-%02d.%d]", n, v)
		}
	} else {
		m.conta[tp]++
		mk = fmt.Sprintf("[%s-%02d]", tp, m.conta[tp])
	}
	m.marcador[forma] = mk
	m.forma[mk] = forma
	if k := chaveNormal(forma); k != "" {
		if _, ok := m.chave[k]; !ok {
			m.chave[k] = tp
			m.re = nil
		}
	}
}

// Vazio informa se nada foi registrado.
func (m *Mapa) Vazio() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.marcador) == 0
}

// Resumo conta o que será substituído (para a confirmação de envio).
type Resumo struct {
	Pessoas int          `json:"pessoas"`
	Formas  int          `json:"formas"`
	PorTipo map[Tipo]int `json:"porTipo"`
}

// Resumir devolve as contagens (sem nenhum dado pessoal).
func (m *Mapa) Resumir() Resumo {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Pedaço comum a duas pessoas ganha número próprio, mas não é outra pessoa.
	np := m.pessoas - len(m.semDono)
	r := Resumo{Pessoas: np, Formas: len(m.marcador), PorTipo: map[Tipo]int{}}
	if np > 0 {
		r.PorTipo[Pessoa] = np
	}
	for tp, n := range m.conta {
		r.PorTipo[tp] = n
	}
	return r
}

// Texto curto do resumo: "9 pessoas, 3 CPFs, 1 RG".
func (r Resumo) Texto() string {
	nomes := map[Tipo][2]string{
		Pessoa: {"pessoa", "pessoas"}, CPF: {"CPF", "CPFs"}, RG: {"RG", "RGs"}, Matricula: {"matrícula", "matrículas"},
		Telefone: {"telefone", "telefones"}, Email: {"e-mail", "e-mails"}, CEP: {"CEP", "CEPs"}, Endereco: {"endereço", "endereços"},
		Nascimento: {"data de nascimento", "datas de nascimento"}, OAB: {"inscrição na OAB", "inscrições na OAB"},
	}
	var partes []string
	for _, tp := range tipos {
		if n := r.PorTipo[tp]; n > 0 {
			nm := nomes[tp][1]
			if n == 1 {
				nm = nomes[tp][0]
			}
			partes = append(partes, fmt.Sprintf("%d %s", n, nm))
		}
	}
	if len(partes) == 0 {
		return "nenhum dado pessoal reconhecido"
	}
	return strings.Join(partes, ", ")
}

// separador entre palavras de um nome: espaços e quebras de linha, também
// na forma escapada de JSON ("\n"), como aparecem nos dados estruturados.
const separador = `(?:\s|\\[nrt])+`

// chaveNormal: tokens do texto em minúsculas e sem acento, separados por um
// espaço ("JOÃO  Carlos" → "joao carlos").
func chaveNormal(s string) string {
	s = strings.NewReplacer(`\n`, " ", `\r`, " ", `\t`, " ").Replace(s)
	return strings.Join(strings.Fields(string(texto.Alinhado(s))), " ")
}

func padraoDaChave(k string) string {
	ts := strings.Fields(k)
	for i, t := range ts {
		ts[i] = regexp.QuoteMeta(t)
	}
	return strings.Join(ts, separador)
}

func (m *Mapa) regex() *regexp.Regexp {
	if m.re != nil {
		return m.re
	}
	m.derivar()
	if len(m.chave) == 0 {
		return nil
	}
	ks := make([]string, 0, len(m.chave))
	for k := range m.chave {
		ks = append(ks, k)
	}
	// Mais longas primeiro: "joao da silva souza" antes de "joao da silva".
	sort.Slice(ks, func(i, j int) bool {
		if len(ks[i]) != len(ks[j]) {
			return len(ks[i]) > len(ks[j])
		}
		return ks[i] < ks[j]
	})
	alt := make([]string, len(ks))
	m.ancorados = make([]*regexp.Regexp, len(ks))
	for i, k := range ks {
		alt[i] = padraoDaChave(k)
		m.ancorados[i] = regexp.MustCompile(`^(?:` + alt[i] + `)`)
	}
	m.re = regexp.MustCompile(strings.Join(alt, "|"))
	return m.re
}

func letraOuDigito(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// desescapar devolve a grafia real de um trecho que veio escapado em JSON.
func desescapar(s string) string {
	return strings.NewReplacer(`\n`, "\n", `\r`, "\r", `\t`, "\t").Replace(s)
}

// textoNormal guarda o texto normalizado e a correspondência de posições
// (byte normalizado → byte original). Alinhado preserva a contagem de runas.
type textoNormal struct {
	norm     string
	origDe   []int // por byte de norm: byte correspondente no original
	original string
}

func normalizar(t string) textoNormal {
	runas := texto.Alinhado(t)
	var b strings.Builder
	origDe := make([]int, 0, len(t)+1)
	ob := 0
	for i, r := range []rune(t) {
		n := runas[i]
		var buf [utf8.UTFMax]byte
		w := utf8.EncodeRune(buf[:], n)
		for j := 0; j < w; j++ {
			origDe = append(origDe, ob)
		}
		b.Write(buf[:w])
		ob += utf8.RuneLen(r)
	}
	origDe = append(origDe, ob)
	return textoNormal{norm: b.String(), origDe: origDe, original: t}
}

// aceitavel aplica as condições de troca a um trecho casado: limites de
// palavra no original e, quando a grafia não é exatamente uma já registrada,
// nome de uma só palavra só se começar com maiúscula ("Machado" sim,
// "machado" — o objeto — não).
func (m *Mapa) aceitavel(t string, oi, of int) bool {
	if oi > 0 {
		r, _ := utf8.DecodeLastRuneInString(t[:oi])
		if letraOuDigito(r) {
			return false
		}
	}
	if of < len(t) {
		r, _ := utf8.DecodeRuneInString(t[of:])
		if letraOuDigito(r) {
			return false
		}
	}
	trecho := desescapar(t[oi:of])
	if _, ok := m.marcador[trecho]; ok {
		return true
	}
	nomes := 0
	for _, w := range strings.Fields(trecho) {
		if !eConector(w) && !eConector(strings.ToUpper(w)) {
			nomes++
		}
	}
	if nomes >= 2 {
		return true
	}
	r, _ := utf8.DecodeRuneInString(trecho)
	return unicode.IsUpper(r) || unicode.IsDigit(r) || !unicode.IsLetter(r)
}

// varrer encontra as ocorrências aceitáveis de grafias registradas e chama
// f com as posições no original e a chave. Quando a grafia mais longa não
// é aceitável numa posição, tenta as mais curtas na mesma posição.
func (m *Mapa) varrer(t string, f func(oi, of int, chave string)) {
	re := m.regex()
	if re == nil || t == "" {
		return
	}
	tn := normalizar(t)
	pos := 0
	for pos < len(tn.norm) {
		loc := re.FindStringIndex(tn.norm[pos:])
		if loc == nil {
			return
		}
		ni, nf := pos+loc[0], pos+loc[1]
		oi, of := tn.origDe[ni], tn.origDe[nf]
		if !m.aceitavel(t, oi, of) {
			achou := false
			for _, a := range m.ancorados {
				l := a.FindStringIndex(tn.norm[ni:])
				if l == nil || l[1] == nf-ni {
					continue
				}
				if ofc := tn.origDe[ni+l[1]]; m.aceitavel(t, oi, ofc) {
					nf, of, achou = ni+l[1], ofc, true
					break
				}
			}
			if !achou {
				_, w := utf8.DecodeRuneInString(tn.norm[ni:])
				pos = ni + w
				continue
			}
		}
		f(oi, of, chaveNormal(t[oi:of]))
		pos = nf
	}
}

// Ocultar troca cada grafia registrada pelo seu marcador, sem diferenciar
// caixa nem acentos e respeitando limites de palavra (não troca "Silva"
// dentro de "Silvana"). Uma grafia nova de uma chave conhecida ("JOAO DA
// SILVA" para "João da Silva") é registrada como variante, para que a
// restauração devolva exatamente o texto original.
func (m *Mapa) Ocultar(t string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var b strings.Builder
	ult := 0
	m.varrer(t, func(oi, of int, k string) {
		trecho := desescapar(t[oi:of])
		mk, ok := m.marcador[trecho]
		if !ok {
			tp, conhecida := m.chave[k]
			if !conhecida {
				return
			}
			m.registrar(tp, trecho)
			mk = m.marcador[trecho]
		}
		b.WriteString(t[ult:oi])
		b.WriteString(mk)
		ult = of
	})
	b.WriteString(t[ult:])
	return b.String()
}

// Liberar registra que o trecho não é dado pessoal (decisão do usuário na
// revisão): deixa de ser apontado como suspeito.
func (m *Mapa) Liberar(forma string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if k := chaveNormal(forma); k != "" {
		m.liberados[k] = true
	}
}

var (
	reMarcador = regexp.MustCompile(`\[(?:PESSOA|CPF|RG|MATRICULA|TELEFONE|EMAIL|CEP|ENDERECO|NASCIMENTO|OAB)-\d{2,}(?:\.\d+)?\]`)
	// Formas deformadas (procuradas na resposta ainda pseudonimizada, sem os
	// marcadores válidos):
	//  - com colchete aberto, em qualquer caixa: "[PESSOA 01", "[pessoa-01]", "[CPF_01";
	//  - sem colchetes, só em maiúsculas e com limite de palavra, para não
	//    confundir "matrícula 100012345" dos autos com marcador: "PESSOA-01",
	//    "PESSOA 01"; para CPF, RG, CEP, EMAIL e OAB, só com hífen ou
	//    sublinhado e sem continuação numérica ("CPF-01", não "CPF 529.982…").
	reDeformado = regexp.MustCompile(`\[(?i:PESSOA|CPF|RG|MATRICULA|TELEFONE|EMAIL|CEP|ENDERECO|NASCIMENTO|OAB)[ _\-]?\d{1,3}(?:\.\d+)?\b` +
		`|\b(?:PESSOA|MATRICULA|TELEFONE|ENDERECO|NASCIMENTO)[ _\-]?\d{1,3}(?:\.\d+)?\b` +
		`|\b(?:CPF|RG|CEP|EMAIL|OAB)[_\-]\d{1,3}(?:\.\d+)?(?:[^\d.]|$)`)
)

// ErrMarcador: a resposta tem marcador desconhecido ou alterado.
type ErrMarcador struct{ Marcadores []string }

func (e ErrMarcador) Error() string {
	return fmt.Sprintf("a resposta contém marcadores inexistentes ou alterados: %s", strings.Join(e.Marcadores, ", "))
}

// Restaurar troca os marcadores pelas grafias originais (texto simples).
func (m *Mapa) Restaurar(texto string) (string, error) {
	return m.restaurar(texto, func(s string) string { return s })
}

// RestaurarJSON restaura dentro de um JSON: a grafia original é escapada
// (aspas, barras, quebras de linha) para o JSON continuar válido.
func (m *Mapa) RestaurarJSON(b []byte) ([]byte, error) {
	s, err := m.restaurar(string(b), func(f string) string {
		q, _ := json.Marshal(f)
		return string(q[1 : len(q)-1])
	})
	return []byte(s), err
}

func (m *Mapa) restaurar(texto string, escapar func(string) string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ruins []string
	// Deformados: na resposta como veio, sem os marcadores válidos.
	ruins = append(ruins, reDeformado.FindAllString(reMarcador.ReplaceAllString(texto, " "), -1)...)
	saida := reMarcador.ReplaceAllStringFunc(texto, func(mk string) string {
		f, ok := m.forma[mk]
		if !ok {
			ruins = append(ruins, mk)
			return mk
		}
		return escapar(f)
	})
	if len(ruins) > 0 {
		return "", ErrMarcador{Marcadores: ruins}
	}
	return saida, nil
}

// derivar registra os pedaços dos nomes de pessoa conhecidos, para que o
// acusado citado só pelos sobrenomes ("Beltrano da Costa"), pelo prenome e
// o último sobrenome ("Fulano Costa") ou por uma palavra só ("Costa") também
// seja ocultado. Antes, o pedaço não era trocado e também não ia à revisão.
//
//   - Pedaço de uma só pessoa vira variante dela ([PESSOA-01.3]): a IA
//     continua sabendo de quem se trata.
//   - Pedaço comum a duas pessoas (irmãos com o mesmo sobrenome) ganha
//     número próprio: é ocultado, sem atribuir a ninguém.
//   - Palavra solta só com inicial maiúscula (regra de aceitavel), com três
//     letras ou mais, fora das paradas; e nunca do que o usuário digitou
//     para ocultar.
//
// Chamado com m.mu travado, antes de montar a alternância.
func (m *Mapa) derivar() {
	if len(m.origens) == 0 {
		return
	}
	donos := map[string]map[int]bool{}
	formaDe := map[string]string{}
	isolada := map[string]bool{}
	for _, o := range m.origens {
		n, ok := m.pessoa[chavePessoa(o.forma)]
		if !ok {
			continue
		}
		for _, f := range fragmentos(o.forma, o.isolar) {
			k := chaveNormal(f)
			if k == "" {
				continue
			}
			if donos[k] == nil {
				donos[k] = map[int]bool{}
				formaDe[k] = f
			}
			donos[k][n] = true
			if !strings.Contains(k, " ") {
				isolada[k] = true
			}
		}
	}
	ks := make([]string, 0, len(donos))
	for k := range donos {
		if _, ok := m.chave[k]; !ok {
			ks = append(ks, k)
		}
	}
	sort.Strings(ks) // numeração estável
	for _, k := range ks {
		f := formaDe[k]
		cp := chavePessoa(f)
		_, tinha := m.pessoa[cp]
		if len(donos[k]) == 1 && !tinha {
			for n := range donos[k] {
				m.pessoa[cp] = n
			}
		}
		m.registrar(Pessoa, f)
		if len(donos[k]) > 1 && !tinha {
			m.semDono[m.pessoa[cp]] = true
		}
		if isolada[k] {
			m.fraca[k] = true
		}
	}
}

// fragmentos de um nome: os trechos contínuos com duas palavras de nome ou
// mais ("Beltrano da Costa"), os pares na ordem ("Fulano Costa") e, com
// isolar, as palavras soltas ("Costa").
func fragmentos(nome string, isolar bool) []string {
	ws := strings.Fields(nome)
	var idx []int // posições das palavras de nome
	for i, w := range ws {
		if !eConector(w) && !eConector(strings.ToLower(w)) {
			idx = append(idx, i)
		}
	}
	if len(idx) < 2 {
		return nil
	}
	var out []string
	for a := 0; a < len(idx); a++ {
		for b := a + 1; b < len(idx); b++ {
			if a != 0 || b != len(idx)-1 { // o nome inteiro já está registrado
				out = append(out, strings.Join(ws[idx[a]:idx[b]+1], " "))
			}
			if idx[b] > idx[a]+1 { // par sem o que há entre eles ("Beltrano Costa")
				out = append(out, ws[idx[a]]+" "+ws[idx[b]])
			}
		}
	}
	if isolar {
		for _, i := range idx {
			w := ws[i]
			lw := strings.ToLower(w)
			if utf8.RuneCountInString(w) < 3 || paradas[lw] || paradas[semAcento(lw)] || reRomano.MatchString(w) ||
				ruido(w) || strings.IndexFunc(w, unicode.IsLetter) < 0 {
				continue
			}
			out = append(out, w)
		}
	}
	return out
}
