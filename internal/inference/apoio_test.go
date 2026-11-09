package inference

import (
	"os"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/testutil/ggufsint"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/llamafalso"
)

// Servidor falso e GGUF sintético: internal/testutil/{llamafalso,ggufsint}.
func TestMain(m *testing.M) {
	if llamafalso.Executar() {
		return
	}
	os.Exit(m.Run())
}

func ggufSintetico(arq string, chat bool, pooling uint32, extra int) []byte {
	return ggufsint.Montar(arq, chat, pooling, extra)
}
