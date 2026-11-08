//go:build windows

package session

import "github.com/DiogoBrazil/sys-executor/internal/platform/win"

func protegerDiretorio(dir string) error      { return win.ProtegerDiretorio(dir) }
func inicioProcesso(pid int) (int64, error)   { return win.InicioProcesso(pid) }
func processoVivo(pid int, inicio int64) bool { return win.ProcessoVivo(pid, inicio) }
