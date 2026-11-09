package analise

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
)

// Etapas com o modelo real (opcional; CPU). SYSEXEC_LLAMA_SERVER e
// SYSEXEC_MODELO_TESTE apontam o servidor e o GGUF.
func TestIntegracaoEtapasModeloReal(t *testing.T) {
	exe, modelo := os.Getenv("SYSEXEC_LLAMA_SERVER"), os.Getenv("SYSEXEC_MODELO_TESTE")
	if exe == "" || modelo == "" {
		t.Skip("SYSEXEC_LLAMA_SERVER/SYSEXEC_MODELO_TESTE não definidos")
	}
	h, _ := inference.DetectarHardware()
	s, err := inference.IniciarServidor(context.Background(), inference.ConfigServidor{
		Executavel: exe, Modelo: modelo, Contexto: 8192, Threads: h.NucleosFisicos, TimeoutCarga: 3 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Fechar()
	b, _ := policy.Carregar()
	x := &Executor{Motor: s, Bundle: b, Fluxo: policy.FluxoPADS}
	x.Observar = func(etapa, alvo string, k int, r []byte, motivo string) {
		if motivo != "" {
			t.Logf("  [%s/%s tentativa %d] rejeitada: %s\n  resposta: %s", etapa, alvo, k, motivo, r)
		}
	}
	a, ps := autosTeste()
	ctx := context.Background()
	reg := &Registro{}

	if os.Getenv("SYSEXEC_SO_EXTRACAO") == "" {
		t0 := time.Now()
		d, err := DelimitarPADS(ctx, x, a, ps, reg)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("delimitação (%s, %d chamadas): %d imputações, %d acusados, pendências %v", time.Since(t0).Round(time.Second), x.Chamadas, len(d.Imputacoes), len(d.Acusados), d.Pendencias)
		for _, i := range d.Imputacoes {
			t.Logf("  %s: %s | %s | natureza %q | data %s | pend %v", i.ID, i.Conduta, i.Dispositivo(), i.Natureza, i.Data, i.Pendencias)
		}
		if len(d.Imputacoes) != 2 {
			t.Errorf("esperadas 2 imputações")
		}
	}
	t0 := time.Now()

	alvos := []AlvoExtracao{{"IMP-1", "deixar de comparecer ao serviço"}, {"IMP-2", "faltar com a verdade"}}
	sec := policy.Secao{Nome: "IMPUTACOES_CONGELADAS", Dados: alvos}
	var pontos []Ponto
	for _, p := range ps[1:] {
		t0 = time.Now()
		e, err := ExtrairPeca(ctx, x, a, p, sec, alvos, nil, reg)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("extração %s (%s): %d pontos, %d pessoas, achados %d, descartes %d", p.Titulo, time.Since(t0).Round(time.Second), len(e.Pontos), len(e.Pessoas), len(e.Achados), e.Descartes)
		for _, pt := range e.Pontos {
			t.Logf("  %s %s %s: %q", pt.ID, pt.Alvo, pt.Sentido, pt.Trecho)
		}
		for _, ac := range e.Achados {
			t.Logf("  achado %s: %s", ac.Tipo, ac.Descricao)
		}
		pontos = append(pontos, e.Pontos...)
	}
	var doIMP1 []Ponto
	for _, p := range pontos {
		if p.Alvo == "IMP-1" {
			doIMP1 = append(doIMP1, p)
		}
	}
	t0 = time.Now()
	an, err := AnalisarImputacao(ctx, x, alvos[0], sec, doIMP1, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("análise IMP-1 (%s): %s; fundamentos %+v; defesa %+v; lacunas %v; pend %v", time.Since(t0).Round(time.Second), an.Resultado, an.Fundamentos, an.TesesDefesa, an.Lacunas, an.Pendencias)
	t.Logf("total de chamadas: %d", x.Chamadas)
}
