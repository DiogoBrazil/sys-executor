package engine

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
)

// fatosAleatorios gera até n fatos com datas em ~3 anos (inclui 29/02/2028
// às vezes, para exercitar o caso-limite) e pelo menos um atual.
func fatosAleatorios(rng *rand.Rand, n int) []Transgressao {
	nats := []Natureza{Leve, Media, Grave}
	var fs []Transgressao
	for i := 0; i < n; i++ {
		d := civil.D(1+rng.Intn(28), 1+rng.Intn(12), 2026+rng.Intn(3))
		origem := Historico
		if i == 0 || rng.Intn(3) == 0 {
			origem = Atual
		}
		fs = append(fs, Transgressao{ID: fmt.Sprintf("F%d", i), Natureza: nats[rng.Intn(3)], Data: d, Origem: origem})
	}
	return fs
}

// forcaBruta: existe subconjunto com ≥1 atual, ≥ 8 unidades e datas em menos de um ano?
func forcaBruta(fs []Transgressao) bool {
	n := len(fs)
	for mask := 1; mask < 1<<n; mask++ {
		var sub []Transgressao
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				sub = append(sub, fs[i])
			}
		}
		minD, maxD, atual := sub[0].Data, sub[0].Data, false
		for _, f := range sub {
			if f.Data.Cmp(minD) < 0 {
				minD = f.Data
			}
			if f.Data.Cmp(maxD) > 0 {
				maxD = f.Data
			}
			atual = atual || f.Origem == Atual
		}
		if ok, _ := civil.MenosDeUmAno(minD, maxD, civil.SemEscolha29Fev); ok && atual && Equivaler(sub).Unidades >= 8 {
			return true
		}
	}
	return false
}

// A janela deslizante encontra o gatilho se e somente se a força bruta encontra.
// Regra: TRV-IA-002
func TestPropriedadeJanelaIgualForcaBruta(t *testing.T) {
	rng := rand.New(rand.NewSource(20261108))
	for k := 0; k < 3000; k++ {
		fs := fatosAleatorios(rng, 1+rng.Intn(9))
		r := GatilhoBomOtimo(fs, civil.SemEscolha29Fev)
		if got, want := r.Acionado, forcaBruta(fs); got != want {
			t.Fatalf("caso %d: janela=%v, força bruta=%v; fatos=%+v", k, got, want, fs)
		}
	}
}

// A ordem de processamento não altera o resultado (motivação da regra §15.4).
func TestPropriedadeOrdemNaoImporta(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	comps := []struct {
		c Comportamento
		p int
	}{{Otimo, 35}, {Bom, 22}, {Insuficiente, 7}, {Excepcional, 44}}
	for k := 0; k < 2000; k++ {
		fs := fatosAleatorios(rng, 1+rng.Intn(7))
		for i := range fs {
			fs[i].PeriodoExcepcionalAtual = rng.Intn(2) == 0
		}
		cp := comps[rng.Intn(len(comps))]
		base := Pontuacao(EntradaPontuacao{Comportamento: comp(cp.c), Pontos: pts(cp.p), Sancoes: sancoesDe(fs), Fatos: fs})
		emb := append([]Transgressao(nil), fs...)
		rng.Shuffle(len(emb), func(i, j int) { emb[i], emb[j] = emb[j], emb[i] })
		s := sancoesDe(emb)
		rng.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
		outra := Pontuacao(EntradaPontuacao{Comportamento: comp(cp.c), Pontos: pts(cp.p), Sancoes: s, Fatos: emb})
		if base.PontosFinais != outra.PontosFinais || base.ComportamentoFinal != outra.ComportamentoFinal ||
			base.Regressao != outra.Regressao || base.Concluido != outra.Concluido {
			t.Fatalf("caso %d: ordem alterou o resultado: %+v × %+v", k, base, outra)
		}
	}
}

// Regressão especial nunca convive com descontos ordinários; sem regressão,
// o saldo final é exatamente o informado menos a soma dos descontos.
func TestPropriedadeValorFixoOuDescontos(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	for k := 0; k < 2000; k++ {
		fs := fatosAleatorios(rng, 1+rng.Intn(6))
		p := rng.Intn(46)
		c := ClassificarSaldo(p)
		r := Pontuacao(EntradaPontuacao{Comportamento: comp(c), Pontos: pts(p), Sancoes: sancoesDe(fs), Fatos: fs})
		if !r.Concluido {
			continue
		}
		if r.Regressao != SemRegressao {
			if len(r.Descontos) != 0 {
				t.Fatalf("caso %d: regressão com descontos: %+v", k, r)
			}
			continue
		}
		soma := 0
		for _, d := range r.Descontos {
			soma += d.Pontos
		}
		if r.PontosFinais != p-soma || r.ComportamentoFinal != ClassificarSaldo(p-soma) {
			t.Fatalf("caso %d: %+v", k, r)
		}
		if c == Insuficiente && r.PontosFinais > p {
			t.Fatal("insuficiente nunca é reposto")
		}
	}
}
