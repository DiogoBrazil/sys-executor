//go:build windows

package win

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// SEG-DIR-01: a pasta protegida tem DACL protegida com uma única ACE (o usuário atual).
func TestProtegerDiretorioSomenteUsuario(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessao")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ProtegerDiretorio(dir); err != nil {
		t.Fatal(err)
	}
	sid, err := SIDUsuarioAtual()
	if err != nil {
		t.Fatal(err)
	}
	for _, alvo := range []string{dir, criarArquivo(t, dir)} {
		d, err := LerACL(alvo)
		if err != nil {
			t.Fatal(err)
		}
		if alvo == dir && !d.Protegida {
			t.Errorf("%s: DACL não protegida contra herança", alvo)
		}
		if len(d.Entradas) != 1 {
			t.Errorf("%s: esperado 1 ACE, veio %d: %+v", alvo, len(d.Entradas), d.Entradas)
		}
		for _, e := range d.Entradas {
			if e.Tipo != 0 || e.SID != sid.String() {
				t.Errorf("%s: ACE de outra conta ou de negação: %+v (usuário %s)", alvo, e, sid)
			}
			if e.Mascara&fileAllAccess != fileAllAccess {
				t.Errorf("%s: usuário sem controle total: %+v", alvo, e)
			}
		}
	}
}

func criarArquivo(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "x.bin")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProcessoVivo(t *testing.T) {
	ini, err := InicioProcesso(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if !ProcessoVivo(os.Getpid(), ini) {
		t.Fatal("o próprio processo deveria estar vivo")
	}
	if ProcessoVivo(os.Getpid(), ini+1) {
		t.Fatal("início diferente (PID reaproveitado) deveria contar como morto")
	}
	cmd := exec.Command("cmd.exe", "/c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if ProcessoVivo(cmd.Process.Pid, 0) {
		t.Fatal("processo encerrado deveria contar como morto")
	}
}

// SEG-WER-01 / P-10: a exclusão do WER por usuário funciona e é reversível.
func TestExclusaoWER(t *testing.T) {
	const nome = "sys-executor-teste-wer.exe"
	if err := ExcluirDoWER(nome); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = RemoverExclusaoWER(nome) })
	ok, err := ExclusaoWERAtiva(nome)
	if err != nil || !ok {
		t.Fatalf("exclusão não encontrada no registro: ok=%v err=%v", ok, err)
	}
	if err := RemoverExclusaoWER(nome); err != nil {
		t.Fatal(err)
	}
	if ok, _ := ExclusaoWERAtiva(nome); ok {
		t.Fatal("exclusão deveria ter sido removida")
	}
}
