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
	re       *regexp.Regexp // alternância das grafias (refeita a cada mudança)
}

// NovoMapa cria um mapa vazio.
func NovoMapa() *Mapa {
	return &Mapa{marcador: map[string]string{}, forma: map[string]string{}, pessoa: map[string]int{},
		variante: map[int]int{}, conta: map[Tipo]int{}}
}

// Aprender detecta dados pessoais no texto e os registra.
func (m *Mapa) Aprender(texto string) {
	ach := detectar(texto)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range ach {
		m.registrar(a.tipo, a.forma)
		if a.tipo == Pessoa {
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
		}
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
	r := Resumo{Pessoas: m.pessoas, Formas: len(m.marcador), PorTipo: map[Tipo]int{}}
	if m.pessoas > 0 {
		r.PorTipo[Pessoa] = m.pessoas
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

func (m *Mapa) regex() *regexp.Regexp {
	if m.re != nil || len(m.marcador) == 0 {
		return m.re
	}
	formas := make([]string, 0, len(m.marcador))
	for f := range m.marcador {
		formas = append(formas, f)
	}
	// Mais longas primeiro: "João da Silva Souza" antes de "João da Silva".
	sort.Slice(formas, func(i, j int) bool {
		if len(formas[i]) != len(formas[j]) {
			return len(formas[i]) > len(formas[j])
		}
		return formas[i] < formas[j]
	})
	alt := make([]string, len(formas))
	for i, f := range formas {
		alt[i] = regexp.QuoteMeta(f)
	}
	m.re = regexp.MustCompile(strings.Join(alt, "|"))
	return m.re
}

func letraOuDigito(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// Ocultar troca cada grafia registrada pelo seu marcador, respeitando
// limites de palavra (não troca "Silva" dentro de "Silvana").
func (m *Mapa) Ocultar(texto string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	re := m.regex()
	if re == nil {
		return texto
	}
	var b strings.Builder
	ult := 0
	for _, ix := range re.FindAllStringIndex(texto, -1) {
		if ix[0] > 0 {
			r, _ := utf8.DecodeLastRuneInString(texto[:ix[0]])
			if letraOuDigito(r) {
				continue
			}
		}
		if ix[1] < len(texto) {
			r, _ := utf8.DecodeRuneInString(texto[ix[1]:])
			if letraOuDigito(r) {
				continue
			}
		}
		b.WriteString(texto[ult:ix[0]])
		b.WriteString(m.marcador[texto[ix[0]:ix[1]]])
		ult = ix[1]
	}
	b.WriteString(texto[ult:])
	return b.String()
}

var (
	reMarcador = regexp.MustCompile(`\[(?:PESSOA|CPF|RG|MATRICULA|TELEFONE|EMAIL|CEP|ENDERECO|NASCIMENTO|OAB)-\d{2,}(?:\.\d+)?\]`)
	// Formas deformadas (procuradas na resposta ainda pseudonimizada, sem os
	// marcadores válidos): sem colchetes, com espaço ou sublinhado, ou com
	// colchete aberto. Só em maiúsculas e com limite de palavra, para não
	// confundir "matrícula 100012345" dos autos com marcador.
	reDeformado = regexp.MustCompile(`\b(?:PESSOA|MATRICULA|TELEFONE|ENDERECO|NASCIMENTO)[ _\-]?\d{1,3}(?:\.\d+)?\b|\[(?:PESSOA|CPF|RG|MATRICULA|TELEFONE|EMAIL|CEP|ENDERECO|NASCIMENTO|OAB)[ _\-]?\d{1,3}(?:\.\d+)?\b`)
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
