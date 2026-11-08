//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/DiogoBrazil/sys-executor/internal/platform/win"
)

// excluirDoWER impede relatórios/dumps do Windows Error Reporting deste
// executável para o usuário atual (plano §13; SEG-WER-01). Melhor esforço.
func excluirDoWER() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = win.ExcluirDoWER(filepath.Base(exe))
}

// prepararFaxina desvincula a cópia de faxina do processo atual.
func prepararFaxina(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008 | 0x00000200 | 0x08000000, // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW
		HideWindow:    true,
	}
}
