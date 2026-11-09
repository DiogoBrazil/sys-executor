// Comando bench: benchmark dos modelos Gemini sobre os casos SINTÉTICOS
// (plano §9; BEN-01; SEG-INJ-01…05; V4 §47.5). Roda as etapas reais
// (delimitação, extração, análise) com a Gemini API e compara com o
// gabarito. Nunca use autos reais. Nada é gravado em disco além do
// relatório pedido em -saida.
//
// A chave de desenvolvimento vem de SYSEXEC_GEMINI_API_KEY (nunca de
// argumento, para não ficar no histórico do terminal).
//
// Uso: go run ./cmd/bench -modelo gemini-… [-ctx 32768] [-pensamento LOW] [-pseudonimizar=false] [-casos filtro] [-saida relatorio.md]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/pseudonimo"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/casos"
)

// medidor envolve o motor e registra desempenho.
type medidor struct {
	inference.Motor
	chamadas  int
	tokEnt    int
	tokSaida  int
	tokPensa  int
	duracao   time.Duration
	truncadas int
	falhas    int
}

func (m *medidor) Gerar(ctx context.Context, p inference.Pedido) (inference.Resposta, error) {
	r, err := m.Motor.Gerar(ctx, p)
	m.chamadas++
	m.tokEnt += r.TokensEntrada
	m.tokSaida += r.TokensSaida
	m.tokPensa += r.TokensPensamento
	m.duracao += r.Duracao
	if r.Truncada {
		m.truncadas++
	}
	if err != nil {
		m.falhas++
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
	pseudo                  string
	erros                   []string
}

type config struct {
	modelo, pensamento string
	ctx                int
	pseudonimizar      bool
}

func main() {
	var cfg config
	flag.StringVar(&cfg.modelo, "modelo", os.Getenv("SYSEXEC_GEMINI_MODELO"), "modelo Gemini (ex.: gemini-3.6-flash)")
	flag.IntVar(&cfg.ctx, "ctx", inference.ContextoPadrao, "janela de trabalho (tokens)")
	flag.StringVar(&cfg.pensamento, "pensamento", "", "nível de raciocínio (MINIMAL, LOW, MEDIUM, HIGH; vazio = padrão do modelo)")
	flag.BoolVar(&cfg.pseudonimizar, "pseudonimizar", true, "trocar dados pessoais por marcadores (como no aplicativo)")
	filtro := flag.String("casos", "", "só casos cujo nome contém o texto")
	saida := flag.String("saida", "", "relatório em markdown (padrão: tela)")
	flag.Parse()
	chave := os.Getenv("SYSEXEC_GEMINI_API_KEY")
	if chave == "" || cfg.modelo == "" {
		fmt.Fprintln(os.Stderr, "uso: SYSEXEC_GEMINI_API_KEY=… bench -modelo <gemini-…>")
		os.Exit(2)
	}
	ctx := context.Background()
	g, err := inference.NovoMotorGemini(ctx, inference.ConfigGemini{Chave: []byte(chave), Modelo: cfg.modelo,
		Contexto: cfg.ctx, Pensamento: cfg.pensamento})
	if err != nil {
		falhar(err)
	}
	defer g.Fechar()
	cfg.ctx = g.Contexto()
	b, err := policy.Carregar()
	if err != nil {
		falhar(err)
	}
	med := &medidor{Motor: g}
	var rs []resultado
	t0 := time.Now()
	for _, c := range casos.Todos() {
		if *filtro != "" && !strings.Contains(c.Nome, *filtro) {
			continue
		}
		fmt.Fprintf(os.Stderr, "caso %s…\n", c.Nome)
		rs = append(rs, rodar(ctx, b, med, c, cfg.pseudonimizar))
	}
	relatorio(rs, cfg, med, time.Since(t0), *saida)
}

func rodar(ctx context.Context, b *policy.Bundle, med *medidor, c casos.Caso, pseudo bool) resultado {
	fluxo := policy.FluxoPADS
	if c.Tipo == "SR" {
		fluxo = policy.FluxoSR
	}
	r := resultado{caso: c.Nome, injecao: c.Injecao, base: c.Base, resultados: map[string]string{}}
	var motor inference.Motor = med
	if pseudo {
		// Como no aplicativo: um mapa por tarefa, a partir de todas as páginas.
		mapa := pseudonimo.NovoMapa()
		for _, pg := range c.Autos.Paginas {
			mapa.Aprender(pg.Texto)
		}
		r.pseudo = mapa.Resumir().Texto()
		motor = pseudonimo.Envolver(med, mapa)
	}
	x := &analise.Executor{Motor: motor, Bundle: b, Fluxo: fluxo}
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
		if err != nil {
			r.erros = append(r.erros, "delimitação SR: "+err.Error())
		}
		if err == nil {
			r.fatos, r.sindicados = len(d.Fatos), len(d.Sindicados)
		}
		r.fatosEsp, r.sindicadosEsp = c.Fatos, c.Sindicados
		r.tempo = time.Since(t0)
		return r
	}
	d, err := analise.DelimitarPADS(ctx, x, c.Autos, c.Pecas, reg)
	r.delimTotal = len(c.Imputacoes)
	if err != nil {
		r.erros = append(r.erros, "delimitação: "+err.Error())
	}
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
			r.erros = append(r.erros, "extração "+p.ID+": "+err.Error())
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
			r.erros = append(r.erros, "análise "+a.ID+": "+err.Error())
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

func relatorio(rs []resultado, cfg config, med *medidor, total time.Duration, saida string) {
	var b strings.Builder
	pens := cfg.pensamento
	if pens == "" {
		pens = "padrão do modelo"
	}
	fmt.Fprintf(&b, "## Benchmark — %s (Gemini API), janela de trabalho %d, raciocínio %s, pseudonimização %v\n\n", cfg.modelo, cfg.ctx, pens, cfg.pseudonimizar)
	fmt.Fprintf(&b, "- Chamadas: %d (falhas %d, respostas cortadas %d) · tokens: entrada %d, saída %d, raciocínio %d · tempo total %s\n\n",
		med.chamadas, med.falhas, med.truncadas, med.tokEnt, med.tokSaida, med.tokPensa, total.Round(time.Second))
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
	if cfg.pseudonimizar {
		b.WriteString("\nPseudonimização por caso:\n")
		for _, r := range rs {
			fmt.Fprintf(&b, "- %s: %s\n", r.caso, r.pseudo)
		}
	}
	b.WriteString("\nErros das etapas (pendências, falhas da API):\n")
	for _, r := range rs {
		for _, e := range r.erros {
			fmt.Fprintf(&b, "- %s: %s\n", r.caso, cortar(e, 200))
		}
	}
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
