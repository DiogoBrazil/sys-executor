//go:build !windows

package credencial

// Fora do Windows não há DPAPI: a loja em arquivo recusa gravar (a chave
// nunca fica em texto puro). Use LojaMemoria em testes e desenvolvimento.
func proteger(_, _ []byte) ([]byte, error) { return nil, ErrSemProtecao }

func desproteger(_, _ []byte) ([]byte, error) { return nil, ErrSemProtecao }

func protegerDiretorio(string) error { return nil }
