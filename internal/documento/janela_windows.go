//go:build windows

package documento

import (
	"os/exec"
	"syscall"
)

const nomeTypst = "typst.exe"

// ocultarJanela evita que o Typst abra uma janela de console.
func ocultarJanela(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
