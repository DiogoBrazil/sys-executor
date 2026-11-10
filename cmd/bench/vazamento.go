package main

// Métrica de vazamento (plano D-11 §4.9): nenhum nome nem dado pessoal dos
// casos sintéticos pode aparecer em nenhum pedido enviado ao provedor.
//
// A conferência é feita sobre o corpo exato da requisição (gancho
// Inspecionar do transporte, depois da barreira), percorrendo todas as
// strings do JSON. Independe do mapa de pseudônimos: o gabarito são os nomes
// declarados nos casos (casos.Caso.Pessoas/Identificadores), e não o que a
// detecção aprendeu. Assim a métrica pega justamente o que o mapa deixou
// passar.
//
// Formas procuradas, sem caixa e sem acento, com limite de palavra:
//   - o nome completo;
//   - cada par de termos consecutivos do nome, sem os conectores
//     ("ANTÔNIO FICTÍCIO", "FICTÍCIO SILVA", "CARLOS INVENTADO");
//   - os identificadores (matrícula).
//
// Termos soltos não entram: alguns sobrenomes fictícios são palavras comuns
// ("suposto", "inventado" aparece nas próprias políticas) e dariam falso
// positivo. Essa limitação fica registrada no relatório da etapa 8.

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

var conectores = map[string]bool{"da": true, "de": true, "do": true, "das": true, "dos": true, "e": true}

// normal: minúsculas, sem acento e com um único espaço entre palavras
// (letras e dígitos; o resto vira separador).
func normal(s string) string {
	s = strings.ToLower(texto.SemAcentos(s))
	var b strings.Builder
	espaco := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			espaco = false
			continue
		}
		if !espaco {
			b.WriteByte(' ')
			espaco = true
		}
	}
	return strings.TrimSpace(b.String())
}

// formasProcuradas devolve as formas normalizadas de um caso, sem repetição.
func formasProcuradas(pessoas, identificadores []string) []string {
	vistas := map[string]bool{}
	var out []string
	add := func(f string) {
		if f != "" && !vistas[f] {
			vistas[f] = true
			out = append(out, f)
		}
	}
	for _, p := range pessoas {
		n := normal(p)
		add(n)
		var termos []string
		for _, t := range strings.Fields(n) {
			if !conectores[t] {
				termos = append(termos, t)
			}
		}
		for i := 0; i+1 < len(termos); i++ {
			add(termos[i] + " " + termos[i+1])
		}
	}
	for _, id := range identificadores {
		add(normal(id))
	}
	sort.Strings(out)
	return out
}

// contem: a forma aparece em t (ambos normalizados) com limite de palavra.
func contem(t, forma string) bool {
	for i := 0; ; {
		k := strings.Index(t[i:], forma)
		if k < 0 {
			return false
		}
		ini, fim := i+k, i+k+len(forma)
		if (ini == 0 || t[ini-1] == ' ') && (fim == len(t) || t[fim] == ' ') {
			return true
		}
		i = ini + 1
	}
}

// vazamento acumula a métrica de todos os pedidos de uma rodada.
type vazamento struct {
	mu        sync.Mutex
	formas    []string // do caso em andamento
	caso      string
	pedidos   int            // corpos conferidos
	comNome   int            // corpos com alguma forma
	bloqueios int            // corpos barrados pela barreira (não saíram)
	achados   map[string]int // "caso: forma" → ocorrências (em corpos distintos)
	ilegiveis int            // corpos que não são JSON (conferidos como texto)
}

func novoVazamento() *vazamento { return &vazamento{achados: map[string]int{}} }

// iniciarCaso troca o gabarito (o bench roda os casos em sequência).
func (v *vazamento) iniciarCaso(nome string, pessoas, identificadores []string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.caso, v.formas = nome, formasProcuradas(pessoas, identificadores)
}

// bloqueio registra um corpo barrado pela barreira de privacidade.
func (v *vazamento) bloqueio() {
	v.mu.Lock()
	v.bloqueios++
	v.mu.Unlock()
}

// conferir examina o corpo exato que vai sair. Nunca bloqueia: só mede (o
// bloqueio é papel da barreira, que roda antes).
func (v *vazamento) conferir(corpo []byte) {
	textos := stringsDoJSON(corpo)
	v.mu.Lock()
	defer v.mu.Unlock()
	if textos == nil {
		v.ilegiveis++
		textos = []string{string(corpo)}
	}
	v.pedidos++
	achou := map[string]bool{}
	for _, t := range textos {
		n := normal(t)
		for _, f := range v.formas {
			if !achou[f] && contem(n, f) {
				achou[f] = true
			}
		}
	}
	if len(achou) == 0 {
		return
	}
	v.comNome++
	for f := range achou {
		v.achados[v.caso+": "+f]++
	}
}

// stringsDoJSON devolve todas as strings (valores e chaves) do JSON; nil se
// o corpo não for JSON.
func stringsDoJSON(corpo []byte) []string {
	var x any
	if err := json.Unmarshal(corpo, &x); err != nil {
		return nil
	}
	out := []string{}
	var andar func(any)
	andar = func(x any) {
		switch y := x.(type) {
		case string:
			out = append(out, y)
		case []any:
			for _, z := range y {
				andar(z)
			}
		case map[string]any:
			for k, z := range y {
				out = append(out, k)
				andar(z)
			}
		}
	}
	andar(x)
	return out
}

// listaAchados em ordem estável, para o relatório.
func (v *vazamento) listaAchados() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	out := make([]string, 0, len(v.achados))
	for k := range v.achados {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
