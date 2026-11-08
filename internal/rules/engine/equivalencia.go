package engine

import (
	"fmt"
	"strings"
)

// Equivalente é a soma de transgressões em unidades (I-01: leve 1, média 2,
// grave 4), expressa como graves equivalentes e o resto, sem arredondar
// (V4 §15.1: 2 médias + 2 leves = 1 grave e 1 média; 3 leves = 1 média e 1 leve).
type Equivalente struct {
	Unidades int
	Graves   int
	Medias   int // resto: 0 ou 1
	Leves    int // resto: 0 ou 1
	Usadas   []string
}

// Equivaler soma as transgressões válidas, cada uma UMA vez (V4 §15.1;
// EQV-DUP-01). Acusações pendentes são ignoradas (V4 §8.7; EQV-PEND-01).
func Equivaler(ts []Transgressao) Equivalente {
	var e Equivalente
	vistos := map[string]bool{}
	for _, t := range ts {
		if t.Origem == Pendente || !t.Natureza.Valida() || vistos[t.ID] {
			continue
		}
		vistos[t.ID] = true
		e.Unidades += t.Natureza.Unidades()
		e.Usadas = append(e.Usadas, t.ID)
	}
	e.Graves = e.Unidades / 4
	resto := e.Unidades % 4
	e.Medias, e.Leves = resto/2, resto%2
	return e
}

func (e Equivalente) String() string {
	var partes []string
	plural := func(n int, s, p string) string {
		if n == 1 {
			return "1 " + s
		}
		return fmt.Sprintf("%d %s", n, p)
	}
	if e.Graves > 0 {
		partes = append(partes, plural(e.Graves, "grave", "graves"))
	}
	if e.Medias > 0 {
		partes = append(partes, plural(e.Medias, "média", "médias"))
	}
	if e.Leves > 0 {
		partes = append(partes, plural(e.Leves, "leve", "leves"))
	}
	if len(partes) == 0 {
		return "nenhuma transgressão"
	}
	return strings.Join(partes, " e ")
}
