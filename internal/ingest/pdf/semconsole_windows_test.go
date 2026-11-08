//go:build windows

package pdf

import (
	"context"
	"os"
	"syscall"
	"testing"
)

// O app roda sem console (-H windowsgui): stdout/stderr são identificadores
// inválidos. O pdfium em WebAssembly precisa iniciar mesmo assim (defeito
// encontrado no teste do usuário em 08/11/2026).
func TestIniciaSemConsole(t *testing.T) {
	out, errOut := os.Stdout, os.Stderr
	os.Stdout = os.NewFile(uintptr(syscall.InvalidHandle), "/dev/stdout")
	os.Stderr = os.NewFile(uintptr(syscall.InvalidHandle), "/dev/stderr")
	defer func() { os.Stdout, os.Stderr = out, errOut }()

	b, err := os.ReadFile("../../../testdata/autos/coordenadas.pdf")
	if err != nil {
		t.Fatal(err)
	}
	ext := NovoExtrator()
	defer ext.Fechar()
	d, err := ext.Abrir(context.Background(), b)
	if err != nil {
		t.Fatalf("sem console o pdfium deve iniciar: %v", err)
	}
	d.Fechar()
}
