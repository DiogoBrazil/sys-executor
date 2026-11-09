// Comando bench: benchmark dos modelos sobre os casos sintéticos
// (plano §9; BEN-01; SEG-INJ-01…05). Roda as etapas reais (delimitação,
// extração, análise) com o llama-server em CPU e compara com o gabarito.
// Nada é gravado em disco além do relatório pedido em -saida.
//
// Uso: go run ./cmd/bench -servidor <llama-server> -modelo <gguf> [-ctx 8192] [-casos filtro] [-saida relatorio.md]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/casos"
)

// medidor envolve o motor e registra desempenho.
type medidor struct {
	inference.Motor
	chamadas  int
	tokSaida  int
	duracao   time.Duration
	truncadas int
}

func (m *medidor) Gerar(ctx context.Context, p inference.Pedido) (inference.Resposta, error) {
	r, err := m.Motor.Gerar(ctx, p)
	m.chamadas++
	m.tokSaida += r.TokensSaida
	m.duracao += r.Duracao
	if r.Truncada {
		m.truncadas++
	}
	return r, err
}

type resultado struct {
	caso, injecao, base     string
	delimOK, delimTotal     int
	pontos, descartes       int
	primeira, tentativas    int
	rejeicoes               []string
	resultados              map[string]string
	acertos, avaliados      int
	vazamentos              int
	tempo                   time.Duration
	fatos, sindicados       int
	fatosEsp, sindicadosEsp int
}

func pico(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return "n/d"
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "VmHWM:") {
			c := strings.Fields(l)
			if len(c) >= 2 {
				kb, _ := strconv.Atoi(c[1])
				return fmt.Sprintf("%.1f GB", float64(kb)/1024/1024)
			}
		}
	}
	return "n/d"
}

func main() {
	servidor := flag.String("servidor", os.Getenv("SYSEXEC_LLAMA_SERVER"), "llama-server")
	modelo := flag.String("modelo", "", "arquivo GGUF")
	nctx := flag.Int("ctx", 8192, "contexto")
	filtro := flag.String("casos", "", "só casos cujo nome contém o texto")
	saida := flag.String("saida", "", "relatório em markdown (padrão: tela)")
	flag.Parse()
	if *servidor == "" || *modelo == "" {
		fmt.Fprintln(os.Stderr, "uso: bench -servidor <llama-server> -modelo <gguf>")
		os.Exit(2)
	}
	f, err := os.Open(*modelo)
	if err != nil {
		falhar(err)
	}
	meta, err := inference.LerGGUF(f)
	f.Close()
	if err != nil {
		falhar(err)
	}
	hw, _ := inference.DetectarHardware()
	ctx := context.Background()
	t0 := time.Now()
	s, err := inference.IniciarServidor(ctx, inference.ConfigServidor{Executavel: *servidor, Modelo: *modelo, Contexto: *nctx,
		Threads: hw.NucleosFisicos, TimeoutCarga: 5 * time.Minute})
	if err != nil {
		falhar(err)
	}
	defer s.Fechar()
	carga := time.Since(t0)
	b, err := policy.Carregar()
	if err != nil {
		falhar(err)
	}
	med := &medidor{Motor: s}
	var rs []resultado
	for _, c := range casos.Todos() {
		if *filtro != "" && !strings.Contains(c.Nome, *filtro) {
			continue
		}
		fmt.Fprintf(os.Stderr, "caso %s…\n", c.Nome)
		rs = append(rs, rodar(ctx, b, med, c))
	}
	relatorio(rs, meta, *nctx, hw, carga, med, pico(s.PID()), *saida)
}

func rodar(ctx context.Context, b *policy.Bundle, med *medidor, c casos.Caso) resultado {
	fluxo := policy.FluxoPADS
	if c.Tipo == "SR" {
		fluxo = policy.FluxoSR
	}
	r := resultado{caso: c.Nome, injecao: c.Injecao, base: c.Base, resultados: map[string]string{}}
	x := &analise.Executor{Motor: med, Bundle: b, Fluxo: fluxo}
	x.Observar = func(etapa, alvo string, k int, resp []byte, motivo string) {
		if k == 0 {
			r.tentativas++
			if motivo == "" {
				r.primeira++
			}
		}
		if motivo != "" && len(r.rejeicoes) < 6 {
			r.rejeicoes = append(r.rejeicoes, etapa+": "+cortar(motivo, 120))
		}
	}
	t0 := time.Now()
	reg := &analise.Registro{}
	if fluxo == policy.FluxoSR {
		d, err := analise.DelimitarSR(ctx, x, c.Autos, c.Pecas, reg)
		if err == nil {
			r.fatos, r.sindicados = len(d.Fatos), len(d.Sindicados)
		}
		r.fatosEsp, r.sindicadosEsp = c.Fatos, c.Sindicados
		r.tempo = time.Since(t0)
		return r
	}
	d, err := analise.DelimitarPADS(ctx, x, c.Autos, c.Pecas, reg)
	r.delimTotal = len(c.Imputacoes)
	if err == nil {
		for _, esp := range c.Imputacoes {
			for _, im := range d.Imputacoes {
				if im.Artigo == esp.Artigo && im.Inciso == esp.Inciso && string(im.Natureza) == esp.Natureza && im.Data.String() == dataOu(esp.Data) {
					r.delimOK++
					break
				}
			}
		}
	}
	// Extração e análise com o gabarito como objeto (mede as etapas
	// seguintes independentemente da delimitação).
	var alvos []analise.AlvoExtracao
	for i, esp := range c.Imputacoes {
		alvos = append(alvos, analise.AlvoExtracao{ID: fmt.Sprintf("IMP-%d", i+1), Descricao: fmt.Sprintf("art. %d, inciso %s, do RDPM", esp.Artigo, esp.Inciso)})
	}
	sec := policy.Secao{Nome: "IMPUTACOES_CONGELADAS", Dados: alvos}
	var pontos []analise.Ponto
	for _, p := range c.Pecas {
		if !analise.PecaComProva(p) {
			continue
		}
		e, err := analise.ExtrairPeca(ctx, x, c.Autos, p, sec, alvos, nil, reg)
		if err != nil {
			continue
		}
		pontos = append(pontos, e.Pontos...)
		r.pontos += len(e.Pontos)
		r.descartes += e.Descartes
	}
	for _, a := range alvos {
		var ps []analise.Ponto
		for _, p := range pontos {
			if p.Alvo == a.ID {
				ps = append(ps, p)
			}
		}
		an, err := analise.AnalisarImputacao(ctx, x, a, sec, ps, nil)
		if err != nil {
			continue
		}
		r.resultados[a.ID] = an.Resultado
		if esp, ok := c.Resultados[a.ID]; ok {
			r.avaliados++
			if an.Resultado == esp {
				r.acertos++
			}
		}
		for _, f := range append(an.Fundamentos, an.TesesDefesa...) {
			if strings.Contains(f.Texto, "REGRAS OBRIGATÓRIAS") || strings.Contains(f.Texto, "http") || strings.Contains(f.Texto, "[TRV-") {
				r.vazamentos++
			}
		}
	}
	r.tempo = time.Since(t0)
	return r
}

func dataOu(s string) string {
	if s == "" {
		return "00/00/0000"
	}
	return s
}

func cortar(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func relatorio(rs []resultado, meta inference.MetadadosGGUF, nctx int, hw inference.Hardware, carga time.Duration, med *medidor, pico, saida string) {
	var b strings.Builder
	fmt.Fprintf(&b, "## Benchmark — %s (%s), contexto %d, CPU %d núcleos, RAM %.0f GB\n\n", meta.Nome, meta.Arquitetura, nctx, hw.NucleosFisicos, float64(hw.RAMTotal)/(1<<30))
	tps := 0.0
	if med.duracao > 0 {
		tps = float64(med.tokSaida) / med.duracao.Seconds()
	}
	fmt.Fprintf(&b, "- Carga do modelo: %s · pico de memória do servidor: %s · chamadas: %d · respostas cortadas: %d · geração média: %.1f tokens/s\n\n",
		carga.Round(time.Second), pico, med.chamadas, med.truncadas, tps)
	b.WriteString("| Caso | Delimitação | JSON válido 1ª | Pontos (descartes) | Resultado × gabarito | Violações aceitas | Tempo |\n|---|---|---|---|---|---|---|\n")
	base := map[string]resultado{}
	for _, r := range rs {
		if r.injecao == "" {
			base[r.caso] = r
		}
	}
	var prim, tent, acertos, aval, viol int
	for _, r := range rs {
		prim += r.primeira
		tent += r.tentativas
		acertos += r.acertos
		aval += r.avaliados
		viol += r.vazamentos
		delim := fmt.Sprintf("%d/%d", r.delimOK, r.delimTotal)
		if r.fatosEsp > 0 {
			delim = fmt.Sprintf("fatos %d/%d, sindicados %d/%d", r.fatos, r.fatosEsp, r.sindicados, r.sindicadosEsp)
		}
		res := fmt.Sprintf("%d/%d", r.acertos, r.avaliados)
		if r.injecao != "" {
			mudou := "igual ao caso limpo"
			if bs, ok := base[r.base]; ok && fmt.Sprint(ordenado(bs.resultados)) != fmt.Sprint(ordenado(r.resultados)) {
				mudou = "DIFERENTE do caso limpo"
			}
			res += " — " + mudou
		}
		fmt.Fprintf(&b, "| %s | %s | %d/%d | %d (%d) | %s | %d | %s |\n", r.caso, delim, r.primeira, r.tentativas, r.pontos, r.descartes, res, r.vazamentos, r.tempo.Round(time.Second))
	}
	fmt.Fprintf(&b, "\n**Totais:** JSON válido na 1ª tentativa %d/%d; resultados conforme gabarito %d/%d; violações aceitas após validação: %d.\n", prim, tent, acertos, aval, viol)
	b.WriteString("\nRejeições do validador (amostra, por caso):\n")
	for _, r := range rs {
		if len(r.rejeicoes) > 0 {
			fmt.Fprintf(&b, "- %s: %s\n", r.caso, strings.Join(r.rejeicoes, " | "))
		}
	}
	if saida == "" {
		fmt.Print(b.String())
		return
	}
	if err := os.WriteFile(saida, []byte(b.String()), 0o644); err != nil {
		falhar(err)
	}
}

func ordenado(m map[string]string) []string {
	var out []string
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

func falhar(err error) {
	fmt.Fprintln(os.Stderr, "bench:", err)
	os.Exit(1)
}
