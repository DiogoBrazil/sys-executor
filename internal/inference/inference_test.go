package inference

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/llamafalso"
)

func eCod(err error, c erros.Codigo) bool { return errors.Is(err, erros.Novo(c)) }

func TestCatalogoEmbutido(t *testing.T) {
	c, err := Catalogo()
	if err != nil {
		t.Fatal(err)
	}
	g := c.Geradores()
	if len(g) < 2 || g[0].ID != "qwen3.5-4b-q4km" || g[1].ID != "qwen3.5-2b-q4km" {
		t.Fatalf("ordem de qualidade: %+v", g)
	}
	for _, m := range c.Modelos {
		if m.Licenca != "Apache-2.0" || strings.Contains(m.Nome, "Qwen2.5") {
			t.Errorf("%s: licença %s (Qwen2.5-3B fica fora: P-08)", m.ID, m.Licenca)
		}
	}
	ruim := c
	ruim.Modelos = append([]Modelo(nil), c.Modelos...)
	ruim.Modelos[0].URL = "http://exemplo.com/x.gguf"
	if ruim.Validar() == nil {
		t.Fatal("URL não HTTPS/não fixada aceita")
	}
}

func TestLerGGUF(t *testing.T) {
	m, err := LerGGUF(bytes.NewReader(ggufSintetico("qwen3", true, 0, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if m.Arquitetura != "qwen3" || m.Camadas != 8 || m.CabecasKV != 2 || m.DimChave != 64 || !m.TemplateDeChat {
		t.Fatalf("%+v", m)
	}
	if got := m.BytesKV(1000); got != 8*2*(64+64)*2*1000 {
		t.Fatalf("KV: %d", got)
	}
	h := MetadadosGGUF{Camadas: 32, IntervaloAtenc: 4, CabecasKV: 4, DimChave: 256, DimValor: 256}
	if h.CamadasComKV() != 8 || h.BytesKV(8192)>>20 != 256 {
		t.Fatalf("híbrido: %d camadas, %d MiB", h.CamadasComKV(), h.BytesKV(8192)>>20)
	}
	for nome, b := range map[string][]byte{
		"vazio":     nil,
		"texto":     []byte("isto não é um modelo"),
		"truncado":  ggufSintetico("qwen3", true, 0, 0)[:40],
		"versão 9":  append([]byte("GGUF\x09\x00\x00\x00"), make([]byte, 16)...),
		"sem chave": append([]byte("GGUF\x03\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00"), make([]byte, 8)...),
	} {
		if _, err := LerGGUF(bytes.NewReader(b)); !errors.Is(err, ErrNaoGGUF) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}

func hw(totalGB, livreGB float64) Hardware {
	return Hardware{NucleosFisicos: 6, NucleosLogicos: 12, AVX2: true,
		RAMTotal: uint64(totalGB * gib), RAMLivre: uint64(livreGB * gib)}
}

var meta4b = MetadadosGGUF{Camadas: 32, IntervaloAtenc: 4, CabecasKV: 4, DimChave: 256, DimValor: 256, ContextoTreino: 262144}

// Cobre: T16
func TestOrcamentoContextoAdaptativo(t *testing.T) {
	c, _ := Catalogo()
	m4, _ := c.PorID("qwen3.5-4b-q4km")
	m2, _ := c.PorID("qwen3.5-2b-q4km")
	if a, b := FaixaContexto(8 * gib); a != 4096 || b != 8192 {
		t.Fatal("faixa 8 GB")
	}
	if a, b := FaixaContexto(16 * gib); a != 8192 || b != 16384 {
		t.Fatal("faixa 16 GB")
	}
	if a, b := FaixaContexto(64 * gib); a != 16384 || b != 32768 {
		t.Fatal("faixa 32 GB+")
	}
	// 16 GB com muita memória livre: topo da faixa.
	p, err := Planejar(hw(16, 13), m4, &meta4b)
	if err != nil || p.Contexto != 16384 || p.Reduzido || p.Threads != 6 {
		t.Fatalf("%+v %v", p, err)
	}
	// Memória livre apertada: contexto reduzido, abaixo da faixa.
	p, err = Planejar(hw(16, 7.0), m4, &meta4b)
	if err != nil || p.Contexto >= 8192 || !p.Reduzido {
		t.Fatalf("reduzido: %+v %v", p, err)
	}
	// T16: 8 GB, ~5 GB livres: 4B não cabe; 2B cabe com contexto da faixa básica.
	if _, err := Planejar(hw(8, 5), m4, &meta4b); !eCod(err, erros.ModeloSemMemoria) {
		t.Fatalf("4B em 8 GB: %v", err)
	}
	p, err = Planejar(hw(8, 5), m2, nil)
	if err != nil || p.Contexto < 4096 || p.Contexto > 8192 {
		t.Fatalf("2B em 8 GB: %+v %v", p, err)
	}
	// O contexto nunca passa do treino do modelo.
	curto := meta4b
	curto.ContextoTreino = 4096
	if p, _ := Planejar(hw(64, 60), m4, &curto); p.Contexto != 4096 {
		t.Fatalf("contexto de treino: %d", p.Contexto)
	}
}

// Cobre: MOD-02, T15, T27
func TestRecomendacao(t *testing.T) {
	c, _ := Catalogo()
	m4, _ := c.PorID("qwen3.5-4b-q4km")
	m2, _ := c.PorID("qwen3.5-2b-q4km")
	// T27: nada instalado → recomenda o melhor que cabe, para baixar.
	r, err := Recomendar(hw(16, 12), c, nil)
	if err != nil || r.Instalado || r.Plano.Modelo.ID != m4.ID || len(r.Alternativas) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	// MOD-02: em 8 GB a recomendação é o perfil básico.
	r, err = Recomendar(hw(8, 5), c, nil)
	if err != nil || r.Plano.Modelo.ID != m2.ID {
		t.Fatalf("8 GB: %+v %v", r, err)
	}
	// Instalado o 2B e cabendo o 4B: usa o instalado e aponta o melhor.
	inst := []Instalado{{Modelo: m2, Caminho: "x"}}
	r, err = Recomendar(hw(16, 12), c, inst)
	if err != nil || !r.Instalado || r.Plano.Modelo.ID != m2.ID || r.Melhor == nil || r.Melhor.ID != m4.ID {
		t.Fatalf("instalado: %+v %v", r, err)
	}
	// T15: nada cabe → MOD-006, sem sucesso simulado.
	if _, err := Recomendar(hw(4, 1.5), c, inst); !eCod(err, erros.ModeloHardwareFraco) {
		t.Fatalf("T15: %v", err)
	}
	// Importado sem perfil testado nunca é recomendado automaticamente.
	imp := Modelo{ID: "importado-abc", Papel: PapelGerador, Tamanho: 1 << 30, PrivadoMB: 500}
	r, _ = Recomendar(hw(16, 12), c, []Instalado{{Modelo: imp}})
	if r.Plano.Modelo.ID == imp.ID {
		t.Fatal("importado recomendado")
	}
	// Seleção manual: 4B escolhido sem memória → MOD-003 com a alternativa (T15).
	inst = []Instalado{{Modelo: m4, Meta: meta4b}, {Modelo: m2}}
	_, alt, err := PlanejarManual(hw(8, 5), c, inst, m4.ID)
	if !eCod(err, erros.ModeloSemMemoria) || len(alt) != 1 || alt[0].Modelo.ID != m2.ID {
		t.Fatalf("manual: %v %+v", err, alt)
	}
	if _, _, err := PlanejarManual(hw(8, 5), c, inst, "inexistente"); !eCod(err, erros.ModeloAusente) {
		t.Fatal(err)
	}
}

// servidorArquivo serve conteúdo com suporte a Range; falharApos interrompe
// a primeira resposta depois de n bytes.
func servidorArquivo(t *testing.T, conteudo []byte, ignorarRange bool, falharApos int) (*httptest.Server, *int32) {
	var pedidos int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&pedidos, 1)
		ini := 0
		if rg := r.Header.Get("Range"); rg != "" && !ignorarRange {
			ini, _ = strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(rg, "bytes="), "-"))
			w.Header().Set("Content-Range", "bytes "+strconv.Itoa(ini)+"-"+strconv.Itoa(len(conteudo)-1)+"/"+strconv.Itoa(len(conteudo)))
			w.Header().Set("Content-Length", strconv.Itoa(len(conteudo)-ini))
			w.WriteHeader(http.StatusPartialContent)
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(conteudo)))
		}
		corpo := conteudo[ini:]
		if n == 1 && falharApos > 0 {
			w.Write(corpo[:falharApos])
			if hj, ok := w.(http.Hijacker); ok {
				c, _, _ := hj.Hijack()
				c.Close()
			}
			return
		}
		w.Write(corpo)
	}))
	t.Cleanup(s.Close)
	return s, &pedidos
}

func modeloDe(conteudo []byte, url string) Modelo {
	s := sha256.Sum256(conteudo)
	return Modelo{ID: "teste-q4", Nome: "Teste", Papel: PapelGerador, Arquivo: "teste.gguf", URL: url,
		SHA256: hex.EncodeToString(s[:]), Tamanho: int64(len(conteudo)), PrivadoMB: 100, Testado: true}
}

func gerenciadorTeste(t *testing.T, ms ...Modelo) *GerenciadorModelos {
	t.Helper()
	g, err := NovoGerenciadorModelos(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	g.Catalogo.Modelos = append(g.Catalogo.Modelos, ms...)
	return g
}

// Cobre: T27, MOD-04
func TestDownloadRetomavelEIntegridade(t *testing.T) {
	conteudo := ggufSintetico("qwen3", true, 0, 300_000)
	ctx := context.Background()

	// Interrompido no meio: MOD-007 e parcial preservado; a 2ª tentativa
	// retoma com Range e confere o SHA-256.
	srv, pedidos := servidorArquivo(t, conteudo, false, 100_000)
	m := modeloDe(conteudo, srv.URL+"/m.gguf")
	g := gerenciadorTeste(t, m)
	if _, err := g.baixar(ctx, m, nil); !eCod(err, erros.ModeloDownload) {
		t.Fatalf("interrupção: %v", err)
	}
	if st, err := os.Stat(filepath.Join(g.Dir, "teste.gguf.parcial")); err != nil || st.Size() == 0 {
		t.Fatal("parcial não preservado")
	}
	var ultimo int64
	inst, err := g.baixar(ctx, m, func(f, tot int64) { ultimo = f })
	if err != nil || inst.Meta.Arquitetura != "qwen3" || ultimo != m.Tamanho || atomic.LoadInt32(pedidos) != 2 {
		t.Fatalf("retomada: %v %+v %d", err, inst, ultimo)
	}
	l, _ := g.Listar()
	if len(l) != 1 || l[0].Modelo.ID != m.ID {
		t.Fatalf("listar: %+v", l)
	}

	// Servidor que ignora Range: recomeça do zero e ainda confere.
	srv2, _ := servidorArquivo(t, conteudo, true, 50_000)
	m2 := modeloDe(conteudo, srv2.URL)
	g2 := gerenciadorTeste(t, m2)
	g2.baixar(ctx, m2, nil)
	if _, err := g2.baixar(ctx, m2, nil); err != nil {
		t.Fatalf("sem Range: %v", err)
	}

	// Conteúdo adulterado: MOD-008 e nada instalado.
	adulterado := append([]byte(nil), conteudo...)
	adulterado[len(adulterado)-1] ^= 1
	srv3, _ := servidorArquivo(t, adulterado, false, 0)
	m3 := modeloDe(conteudo, srv3.URL)
	g3 := gerenciadorTeste(t, m3)
	if _, err := g3.baixar(ctx, m3, nil); !eCod(err, erros.ModeloIntegridade) {
		t.Fatalf("adulterado: %v", err)
	}
	if ents, _ := os.ReadDir(g3.Dir); len(ents) != 0 {
		t.Fatalf("restos do download adulterado: %v", ents)
	}

	// Remoção (MOD-04), bloqueada enquanto em uso.
	liberar := g.Reservar(m.ID)
	if err := g.Remover(m.ID); !eCod(err, erros.ModeloEmUso) {
		t.Fatalf("em uso: %v", err)
	}
	liberar()
	if err := g.Remover(m.ID); err != nil {
		t.Fatal(err)
	}
	if l, _ := g.Listar(); len(l) != 0 {
		t.Fatal("removido continua listado")
	}
	if _, err := os.Stat(filepath.Join(g.Dir, "teste.gguf")); !os.IsNotExist(err) {
		t.Fatal("arquivo não apagado")
	}
	// Só HTTPS no catálogo.
	if _, err := g.Baixar(ctx, "teste-q4", nil); !eCod(err, erros.ModeloDownload) {
		t.Fatalf("http aceito: %v", err)
	}
}

// Cobre: MOD-03
func TestImportacao(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	escrever := func(nome string, b []byte) string {
		p := filepath.Join(dir, nome)
		os.WriteFile(p, b, 0o600)
		return p
	}
	g := gerenciadorTeste(t)
	for nome, b := range map[string][]byte{
		"texto.gguf":   []byte("não é gguf"),
		"arq.gguf":     ggufSintetico("arquitetura-x", true, 0, 10),
		"semchat.gguf": ggufSintetico("qwen3", false, 0, 10),
	} {
		if _, err := g.Importar(ctx, escrever(nome, b), nil); !eCod(err, erros.ModeloIncompativel) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if ents, _ := os.ReadDir(g.Dir); len(ents) != 0 {
		t.Fatalf("importação inválida deixou arquivos: %v", ents)
	}
	// GGUF válido fora do catálogo: importado, sem perfil testado.
	inst, err := g.Importar(ctx, escrever("meu.gguf", ggufSintetico("qwen3", true, 0, 1000)), nil)
	if err != nil || inst.Modelo.Testado || !strings.HasPrefix(inst.Modelo.ID, "importado-") || inst.Modelo.Papel != PapelGerador {
		t.Fatalf("%+v %v", inst, err)
	}
	// Embeddings (pooling) não exige template de chat.
	emb, err := g.Importar(ctx, escrever("emb.gguf", ggufSintetico("qwen3", false, 3, 10)), nil)
	if err != nil || emb.Modelo.Papel != PapelEmbeddings {
		t.Fatalf("embeddings: %+v %v", emb, err)
	}
	// Mesmo hash de um modelo do catálogo: reconhecido como ele.
	conteudo := ggufSintetico("qwen35", true, 0, 2000)
	cat := modeloDe(conteudo, "https://huggingface.co/x/y/resolve/0000000000000000000000000000000000000000/t.gguf")
	g.Catalogo.Modelos = append(g.Catalogo.Modelos, cat)
	inst, err = g.Importar(ctx, escrever("copia.gguf", conteudo), nil)
	if err != nil || inst.Modelo.ID != cat.ID || !inst.Modelo.Testado {
		t.Fatalf("do catálogo: %+v %v", inst, err)
	}
	if l, _ := g.Listar(); len(l) != 3 {
		t.Fatalf("listar: %d", len(l))
	}
	// Arquivo alterado depois de instalado deixa de ser listado.
	os.WriteFile(inst.Caminho, append(conteudo, 1), 0o600)
	if l, _ := g.Listar(); len(l) != 2 {
		t.Fatalf("alterado ainda listado: %d", len(l))
	}
}

func servidorTeste(t *testing.T, modo string) (*Servidor, error) {
	t.Helper()
	exe, _ := os.Executable()
	t.Setenv(llamafalso.Variavel, modo)
	return IniciarServidor(context.Background(), ConfigServidor{
		Executavel: exe, Modelo: "x.gguf", Contexto: 4096, Threads: 2,
		TimeoutCarga: 20 * time.Second, TimeoutPedido: 10 * time.Second,
	})
}

// Cobre: SEG-SID-01
func TestSidecar(t *testing.T) {
	s, err := servidorTeste(t, "ok")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Fechar()
	if !strings.HasPrefix(s.base, "http://127.0.0.1:") || len(s.chave) < 32 {
		t.Fatalf("endereço/chave: %s", s.base)
	}
	args := strings.Join(argumentos(s.cfg, 1, "k"), " ")
	for _, a := range []string{"--host 127.0.0.1", "--no-webui", "--log-disable", "--offline", "-ngl 0", "--reasoning off"} {
		if !strings.Contains(args, a) {
			t.Errorf("argumento ausente: %s", a)
		}
	}
	// Sem chave → 401.
	r, err := http.Post(s.base+"/v1/chat/completions", "application/json", strings.NewReader("{}"))
	if err != nil || r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("sem chave: %v %v", err, r)
	}
	r.Body.Close()
	ctx := context.Background()
	res, err := s.Gerar(ctx, Pedido{Sistema: "s", Usuario: "olá", Schema: json.RawMessage(`{"type":"object"}`), MaxTokens: 50})
	if err != nil || !bytes.Contains(res.Conteudo, []byte(`"formato":"json_schema"`)) || res.Truncada || res.TokensSaida != 7 {
		t.Fatalf("%s %+v %v", res.Conteudo, res, err)
	}
	res, _ = s.Gerar(ctx, Pedido{Usuario: "x", MaxTokens: 1, Historico: []Mensagem{{"assistant", "a"}, {"user", "corrija"}}})
	if !res.Truncada || !bytes.Contains(res.Conteudo, []byte(`"mensagens":4`)) {
		t.Fatalf("truncada/histórico: %s", res.Conteudo)
	}
	if n, err := s.ContarTokens(ctx, "um dois três"); err != nil || n != 3 {
		t.Fatalf("tokens: %d %v", n, err)
	}
	s.Fechar()
	if s.Vivo() {
		t.Fatal("processo continua vivo")
	}
	if _, err := s.Gerar(ctx, Pedido{Usuario: "x"}); err == nil {
		t.Fatal("pedido após fechar")
	}
}

// Cobre: T15
func TestSidecarFalhas(t *testing.T) {
	if _, err := servidorTeste(t, "morre"); !eCod(err, erros.ModeloTravado) {
		t.Fatalf("morte na carga: %v", err)
	}
	s, err := servidorTeste(t, "morre-no-pedido")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Fechar()
	if _, err := s.Gerar(context.Background(), Pedido{Usuario: "x"}); !eCod(err, erros.ModeloTravado) {
		t.Fatalf("morte no pedido: %v", err)
	}
	if _, err := IniciarServidor(context.Background(), ConfigServidor{Executavel: filepath.Join(t.TempDir(), "nao-existe")}); !eCod(err, erros.ModeloIncompativel) {
		t.Fatalf("executável ausente: %v", err)
	}
}

func TestHardwareDetectado(t *testing.T) {
	h, err := DetectarHardware()
	if err != nil || h.RAMTotal == 0 || h.RAMLivre == 0 || h.NucleosLogicos == 0 || h.NucleosFisicos == 0 {
		t.Fatalf("%+v %v", h, err)
	}
	if len(h.GPUs) != 0 {
		t.Fatal("GPU sondada sem consentimento (D-08)")
	}
}

func TestLerDispositivos(t *testing.T) {
	saida := []byte("Available devices:\n  Vulkan0: AMD Radeon RX 6600 XT (8176 MiB, 7900 MiB free)\n  Vulkan1: Intel UHD (2048 MiB, 1900 MiB free)\nlixo\n")
	gs := lerDispositivos(saida)
	if len(gs) != 2 || gs[0].Backend != "Vulkan" || gs[0].Nome != "AMD Radeon RX 6600 XT" || gs[0].VRAMLivre != 7900*mib {
		t.Fatalf("%+v", gs)
	}
}
