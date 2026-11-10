package credencial

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const sentinela = "AIzaSy-SENTINELA-DPAPI-0123456789"

// Regra: TRV-SEG-009
func TestLojaArquivo(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "credenciais")
	l := NovaLojaArquivo(dir)
	if l.Existe() {
		t.Fatal("não deveria existir")
	}
	if _, err := l.Ler(); !errors.Is(err, ErrAusente) {
		t.Fatalf("esperado ErrAusente: %v", err)
	}
	err := l.Guardar([]byte(sentinela))
	if runtime.GOOS != "windows" {
		if !errors.Is(err, ErrSemProtecao) {
			t.Fatalf("fora do Windows não pode gravar sem proteção: %v", err)
		}
		if _, e := os.Stat(filepath.Join(dir, arquivoGemini)); e == nil {
			t.Fatal("arquivo gravado sem proteção")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	cru, err := os.ReadFile(filepath.Join(dir, arquivoGemini))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cru), sentinela) || strings.Contains(string(cru), "SENTINELA") {
		t.Fatal("a chave está em texto puro no disco")
	}
	got, err := l.Ler()
	if err != nil || string(got) != sentinela {
		t.Fatalf("ida e volta: %q %v", got, err)
	}
	// Arquivo adulterado não decifra.
	cru[len(cru)/2] ^= 0xFF
	os.WriteFile(filepath.Join(dir, arquivoGemini), cru, 0o600)
	if _, err := l.Ler(); err == nil {
		t.Fatal("arquivo adulterado foi aceito")
	}
	if err := l.Remover(); err != nil || l.Existe() {
		t.Fatalf("remover: %v", err)
	}
	if err := l.Remover(); err != nil {
		t.Fatal("remover de novo não é erro")
	}
}

func TestLojaMemoria(t *testing.T) {
	var l LojaMemoria
	if err := l.Guardar([]byte("  ")); err == nil {
		t.Fatal("chave vazia aceita")
	}
	if err := l.Guardar([]byte(sentinela)); err != nil {
		t.Fatal(err)
	}
	a, _ := l.Ler()
	Zerar(a)
	b, _ := l.Ler()
	if string(b) != sentinela {
		t.Fatal("Ler deve devolver cópia")
	}
	l.Remover()
	if l.Existe() {
		t.Fatal("removida")
	}
}

// V4 §48.3: a credencial do ChatGPT fica na mesma pasta protegida, com nome
// e entropia próprios; uma não decifra a outra (job go-windows do CI).
// Regra: TRV-SEG-009
func TestLojasNomeadasIsoladas(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "credenciais")
	g := NovaLojaArquivo(dir)
	c := NovaLojaNomeada(dir, "chatgpt.dpapi", []byte("sys-executor/chatgpt/v1"))
	if c.caminho() == g.caminho() {
		t.Fatal("as lojas não podem compartilhar o arquivo")
	}
	for _, ruim := range []string{"", "../x.dpapi", `a\b`} {
		func() {
			defer func() { _ = recover() }()
			NovaLojaNomeada(dir, ruim, []byte("e"))
			if ruim == "" || ruim == "../x.dpapi" {
				t.Errorf("nome %q deveria ser recusado", ruim)
			}
		}()
	}
	if runtime.GOOS != "windows" {
		if err := c.Guardar([]byte(sentinela)); !errors.Is(err, ErrSemProtecao) {
			t.Fatalf("fora do Windows: %v", err)
		}
		return
	}
	if err := g.Guardar([]byte("chave-gemini")); err != nil {
		t.Fatal(err)
	}
	if err := c.Guardar([]byte(sentinela)); err != nil {
		t.Fatal(err)
	}
	// Com a entropia da outra loja, o arquivo não decifra.
	cruzada := NovaLojaNomeada(dir, "chatgpt.dpapi", entropiaGemini)
	if _, err := cruzada.Ler(); err == nil {
		t.Fatal("entropia diferente decifrou o segredo")
	}
	if v, err := g.Ler(); err != nil || string(v) != "chave-gemini" {
		t.Fatalf("loja Gemini afetada: %q %v", v, err)
	}
}
