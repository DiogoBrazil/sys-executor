package session

import (
	"bufio"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

const sentinela = "SENTINELA-AUTOS-0021.000000/2026-00"

// Quando o binário de teste é relançado como "filho", cria uma sessão, grava o
// sentinela e fica parado até ser morto (simula queda abrupta do app).
func TestMain(m *testing.M) {
	if raiz := os.Getenv("SYSEXEC_TESTE_FILHO_RAIZ"); raiz != "" {
		s, err := NovoGerenciador(raiz).Criar()
		if err != nil {
			os.Stdout.WriteString("erro " + err.Error() + "\n")
			os.Exit(2)
		}
		if err := s.Gravar("autos.pdf", []byte(sentinela)); err != nil {
			os.Exit(3)
		}
		os.Stdout.WriteString("pronto " + s.Dir() + "\n")
		time.Sleep(10 * time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func arquivosContem(t *testing.T, raiz, alvo string) bool {
	t.Helper()
	achou := false
	_ = filepath.WalkDir(raiz, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if b, e := os.ReadFile(p); e == nil && bytes.Contains(b, []byte(alvo)) {
				achou = true
			}
		}
		return nil
	})
	return achou
}

func TestGravarLerCifradoComNomeOpaco(t *testing.T) {
	raiz := t.TempDir()
	s, err := NovoGerenciador(raiz).Criar()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Encerrar()
	if err := s.Gravar("autos.pdf", []byte(sentinela)); err != nil {
		t.Fatal(err)
	}
	got, err := s.Ler("autos.pdf")
	if err != nil || string(got) != sentinela {
		t.Fatalf("leitura: %q %v", got, err)
	}
	if arquivosContem(t, raiz, sentinela) || arquivosContem(t, raiz, "autos.pdf") {
		t.Fatal("conteúdo ou nome lógico apareceu em claro no disco")
	}
	if _, err := s.Ler("inexistente"); !errors.Is(err, erros.Novo(erros.ArquivoLeitura)) {
		t.Fatalf("esperado ARQ-001, veio %v", err)
	}
}

// T11 (parcial) / TRV-SES-002: encerrar apaga tudo e é idempotente.
func TestEncerrarApagaTudo(t *testing.T) {
	raiz := t.TempDir()
	s, err := NovoGerenciador(raiz).Criar()
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Gravar("a", bytes.Repeat([]byte("x"), 200<<10))
	dir := s.Dir()
	if err := s.Encerrar(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("pasta da sessão ainda existe")
	}
	if err := s.Encerrar(); err != nil {
		t.Fatal("Encerrar deve ser idempotente")
	}
	if err := s.Gravar("b", []byte("y")); err == nil {
		t.Fatal("gravar após encerrar deveria falhar")
	}
}

// T12 / SEG-CRASH-01: processo morto à força → a próxima abertura remove o
// resíduo, sem restaurar nada; sessão viva do próprio processo é preservada.
func TestVarreduraRemoveResiduoDeQueda(t *testing.T) {
	raiz := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^$")
	cmd.Env = append(os.Environ(), "SYSEXEC_TESTE_FILHO_RAIZ="+raiz)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	linha, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || !strings.HasPrefix(linha, "pronto ") {
		_ = cmd.Process.Kill()
		t.Fatalf("filho não ficou pronto: %q %v", linha, err)
	}
	dirFilho := strings.TrimSpace(strings.TrimPrefix(linha, "pronto "))

	// Sessão viva do próprio processo de teste (não pode ser apagada).
	viva, err := NovoGerenciador(raiz).Criar()
	if err != nil {
		t.Fatal(err)
	}
	defer viva.Encerrar()

	// Enquanto o filho vive, a varredura preserva as duas.
	if v := NovoGerenciador(raiz).VarrerResiduos(); v.Ativas != 2 || v.Removidas != 0 {
		t.Fatalf("com filho vivo: %+v", v)
	}
	if arquivosContem(t, raiz, sentinela) {
		t.Fatal("sentinela em claro no disco")
	}

	if err := cmd.Process.Kill(); err != nil { // queda abrupta
		t.Fatal(err)
	}
	_ = cmd.Wait()

	v := NovoGerenciador(raiz).VarrerResiduos()
	if v.Removidas != 1 || v.Ativas != 1 || v.Pendentes != 0 {
		t.Fatalf("após queda: %+v", v)
	}
	if _, err := os.Stat(dirFilho); !os.IsNotExist(err) {
		t.Fatal("resíduo do processo morto não foi removido")
	}
	if _, err := os.Stat(viva.Dir()); err != nil {
		t.Fatal("sessão viva foi apagada indevidamente")
	}
}

func TestVarreduraRemovePastaSemMarcadorELixo(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "semmarcador", "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(raiz, "semmarcador", "sub", "x"), []byte("y"), 0o600)
	_ = os.WriteFile(filepath.Join(raiz, "solto.tmp"), []byte("z"), 0o600)
	v := NovoGerenciador(raiz).VarrerResiduos()
	if v.Removidas != 1 || v.Pendentes != 0 {
		t.Fatalf("%+v", v)
	}
	if es, _ := os.ReadDir(raiz); len(es) != 0 {
		t.Fatalf("restou: %v", es)
	}
	// Raiz inexistente: nada a fazer, sem erro.
	if v := NovoGerenciador(filepath.Join(raiz, "nao-existe")).VarrerResiduos(); v != (Varredura{}) {
		t.Fatalf("%+v", v)
	}
}

func TestSegredoZerar(t *testing.T) {
	b := []byte("senha-sentinela")
	s := NovoSegredo(b)
	var visto string
	_ = s.Usar(func(p []byte) error { visto = string(p); return nil })
	if visto != "senha-sentinela" {
		t.Fatal("Usar não emprestou o conteúdo")
	}
	s.Zerar()
	if !s.Vazio() || !bytes.Equal(b, make([]byte, len(b))) {
		t.Fatalf("bytes não zerados: %q", b)
	}
	s.Zerar()
}

func TestEsvaziarEProtegerPasta(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "webview")
	if err := ProtegerPasta(dir); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(dir, "EBWebView", "Default"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "EBWebView", "Default", "LOG"), []byte("x"), 0o600)
	if !EsvaziarPasta(dir, time.Second) {
		t.Fatal("pasta não ficou vazia")
	}
	if es, _ := os.ReadDir(dir); len(es) != 0 {
		t.Fatalf("restou %v", es)
	}
	if !EsvaziarPasta(filepath.Join(dir, "nao-existe"), 0) {
		t.Fatal("pasta inexistente conta como vazia")
	}
}
