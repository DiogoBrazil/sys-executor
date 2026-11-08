//go:build linux

package inference

import (
	"os/exec"
	"syscall"
)

type protecao interface {
	pico() uint64
	fechar()
	estourouMemoria(err error) bool
}

type protecaoNula struct{}

func (protecaoNula) pico() uint64               { return 0 }
func (protecaoNula) fechar()                    {}
func (protecaoNula) estourouMemoria(error) bool { return false }

// prepararProcesso (Linux, desenvolvimento): o sidecar morre com o processo
// pai (F0-02 viu um llama-server órfão sem isso).
func prepararProcesso(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}

func proteger(*exec.Cmd, uint64) (protecao, error) { return protecaoNula{}, nil }
