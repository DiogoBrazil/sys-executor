//go:build windows

package ocr

import (
	"os/exec"
	"syscall"
)

const nomeExecutavel = "tesseract.exe"

// ocultarJanela evita que o Tesseract abra uma janela de console.
func ocultarJanela(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
