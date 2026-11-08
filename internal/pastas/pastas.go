// Package pastas centraliza os locais em disco usados pelo aplicativo.
//
// Tudo fica sob %LOCALAPPDATA%\sys-executor (não sob o perfil móvel, para não
// ser sincronizado com servidores da rede):
//
//	trabalho\      sessões efêmeras das tarefas (apagadas ao final e na próxima abertura)
//	webview\       dados do WebView2 (apagados a cada abertura e encerramento)
//	preferencias.json  preferências persistentes mínimas (V4 §29.3)
package pastas

import (
	"os"
	"path/filepath"
)

const nomeApp = "sys-executor"

// Pastas são os locais resolvidos a partir de uma base.
type Pastas struct {
	Base string
}

// Padrao resolve a base: %LOCALAPPDATA%\sys-executor no Windows; o diretório
// de cache do usuário nos demais sistemas (desenvolvimento). A variável
// SYSEXEC_BASE substitui a base (testes automatizados).
func Padrao() (Pastas, error) {
	if b := os.Getenv("SYSEXEC_BASE"); b != "" {
		return Pastas{Base: b}, nil
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		return Pastas{Base: filepath.Join(la, nomeApp)}, nil
	}
	c, err := os.UserCacheDir()
	if err != nil {
		return Pastas{}, err
	}
	return Pastas{Base: filepath.Join(c, nomeApp)}, nil
}

func (p Pastas) Trabalho() string     { return filepath.Join(p.Base, "trabalho") }
func (p Pastas) WebView() string      { return filepath.Join(p.Base, "webview") }
func (p Pastas) Preferencias() string { return filepath.Join(p.Base, "preferencias.json") }

// Modelos guarda os GGUF instalados (persistentes; nenhum dado de processo).
func (p Pastas) Modelos() string { return filepath.Join(p.Base, "modelos") }
