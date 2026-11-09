package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/ggufsint"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/llamafalso"
)

func TestMain(m *testing.M) {
	if llamafalso.Executar() {
		return
	}
	os.Exit(m.Run())
}

func eCod(err error, c erros.Codigo) bool { return errors.Is(err, erros.Novo(c)) }

// motorTeste: pasta de modelos com um gerador "do catálogo" sintético.
func motorTeste(t *testing.T, livreGB float64) (*MotorLocal, *inference.GerenciadorModelos) {
	t.Helper()
	dir := t.TempDir()
	g, err := inference.NovoGerenciadorModelos(filepath.Join(dir, "modelos"))
	if err != nil {
		t.Fatal(err)
	}
	loja, _ := prefs.Abrir(filepath.Join(dir, "preferencias.json"))
	exe, _ := os.Executable()
	m := &MotorLocal{Modelos: g, Loja: loja,
		Exe: Executaveis{CPU: func() (string, error) { return exe, nil }},
		Hardware: func() (inference.Hardware, error) {
			return inference.Hardware{NucleosFisicos: 6, NucleosLogicos: 12, AVX2: true, RAMTotal: 16 << 30, RAMLivre: uint64(livreGB * (1 << 30))}, nil
		}}
	return m, g
}

func instalarSintetico(t *testing.T, g *inference.GerenciadorModelos, testado bool) inference.Modelo {
	t.Helper()
	b := ggufsint.Montar("qwen35", true, 0, 4096)
	s := sha256.Sum256(b)
	md := inference.Modelo{ID: "sint-q4", Nome: "Sintético", Papel: inference.PapelGerador, Arquivo: "sint.gguf",
		SHA256: hex.EncodeToString(s[:]), Tamanho: int64(len(b)), PrivadoMB: 500, Qualidade: 9, Testado: testado}
	g.Catalogo.Modelos = append(g.Catalogo.Modelos, md)
	orig := filepath.Join(t.TempDir(), "x.gguf")
	os.WriteFile(orig, b, 0o600)
	if _, err := g.Importar(context.Background(), orig, nil); err != nil {
		t.Fatal(err)
	}
	return md
}

// Cobre: T27, T15, MOD-01
func TestMotorLocalEscolha(t *testing.T) {
	m, g := motorTeste(t, 12)
	if _, _, err := m.Escolher(); !eCod(err, erros.ModeloAusente) {
		t.Fatalf("sem modelo → MOD-001 (assistente de download/importação): %v", err)
	}
	md := instalarSintetico(t, g, true)
	inst, plano, err := m.Escolher()
	if err != nil || inst.Modelo.ID != md.ID || plano.Contexto != 16384 {
		t.Fatalf("%+v %+v %v", inst, plano, err)
	}
	// Sem memória livre: mensagem amigável, sem sucesso simulado (T15).
	m2, g2 := motorTeste(t, 1.0)
	instalarSintetico(t, g2, true)
	if _, _, err := m2.Escolher(); !eCod(err, erros.ModeloSemMemoria) && !eCod(err, erros.ModeloHardwareFraco) {
		t.Fatalf("T15: %v", err)
	}
}

// Cobre: T14, MOD-01
func TestMotorLocalGPUComFallbackCPU(t *testing.T) {
	m, g := motorTeste(t, 12)
	instalarSintetico(t, g, true)
	t.Setenv(llamafalso.Variavel, "ok")
	ctx := context.Background()
	var atividades []string
	at := func(s string) { atividades = append(atividades, s) }

	// GPU nunca sondada: CPU (D-08), nada de placa de vídeo.
	gpuChamada := false
	m.Exe.GPU = func() (string, error) { gpuChamada = true; return "", errors.New("x") }
	mot, fechar, err := m.Abrir(ctx, at)
	if err != nil || gpuChamada || !strings.Contains(strings.Join(atividades, "|"), "Carregando o modelo de IA (") {
		t.Fatalf("%v gpu=%v %v", err, gpuChamada, atividades)
	}
	if err := g.Remover("sint-q4"); !eCod(err, erros.ModeloEmUso) {
		t.Fatalf("modelo em uso não pode ser removido: %v", err)
	}
	_ = mot
	fechar()

	// GPU estável: prioriza a GPU (T14).
	m.Loja.Atualizar(func(p *prefs.Preferencias) { p.GPU = prefs.GPU{Estado: prefs.GPUEstavel} })
	exe, _ := os.Executable()
	m.Exe.GPU = func() (string, error) { return exe, nil }
	atividades = nil
	_, fechar, err = m.Abrir(ctx, at)
	if err != nil || !strings.Contains(strings.Join(atividades, "|"), "placa de vídeo") {
		t.Fatalf("GPU priorizada: %v %v", err, atividades)
	}
	fechar()

	// GPU falha ao carregar → segue em CPU, sem bloquear a tarefa (T14/V4 §38.1).
	bin := filepath.Join(t.TempDir(), "nao-existe")
	m.Exe.GPU = func() (string, error) { return bin, nil }
	atividades = nil
	_, fechar, err = m.Abrir(ctx, at)
	if err != nil || !strings.Contains(strings.Join(atividades, "|"), "Carregando o modelo de IA (") {
		t.Fatalf("fallback CPU: %v %v", err, atividades)
	}
	fechar()
	if err := g.Remover("sint-q4"); err != nil {
		t.Fatalf("liberado depois de fechar: %v", err)
	}
}

// Cobre: MOD-02, MOD-04
func TestModelosServiceVisao(t *testing.T) {
	m, g := motorTeste(t, 12)
	s := &ModelosService{Motor: m}
	v := s.Estado()
	if v.Pronto || !v.Servidor || len(v.Modelos) < 3 {
		t.Fatalf("%+v", v)
	}
	rec := ""
	for _, mv := range v.Modelos {
		if mv.Recomendado {
			rec = mv.ID
		}
	}
	if rec != "qwen3.5-4b-q4km" {
		t.Fatalf("recomendado em 16 GB/12 livres: %q", rec)
	}
	instalarSintetico(t, g, true)
	v = s.Selecionar("sint-q4")
	if v.Manual != "sint-q4" || !v.Pronto {
		t.Fatalf("%+v", v)
	}
	v = s.Remover("sint-q4")
	if v.Erro != nil || v.Manual != "" || v.Pronto {
		t.Fatalf("remoção: %+v", v.Erro)
	}
}
