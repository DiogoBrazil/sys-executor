//go:build windows

package inference

import (
	"errors"
	"os/exec"
	"syscall"

	"github.com/DiogoBrazil/sys-executor/internal/platform/win"
	"golang.org/x/sys/windows"
)

type protecao interface {
	pico() uint64
	fechar()
	estourouMemoria(err error) bool
}

func prepararProcesso(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

type protecaoJob struct {
	job    *win.Job
	limite uint64
}

// proteger coloca o sidecar num Job com KILL_ON_JOB_CLOSE (e limite de
// memória, se houver).
func proteger(cmd *exec.Cmd, limite uint64) (protecao, error) {
	j, err := win.NovoJob(limite)
	if err != nil {
		return nil, err
	}
	if err := j.Atribuir(cmd.Process.Pid); err != nil {
		j.Fechar()
		return nil, err
	}
	return &protecaoJob{job: j, limite: limite}, nil
}

func (p *protecaoJob) pico() uint64 { return p.job.PicoMemoria() }
func (p *protecaoJob) fechar()      { p.job.Fechar() }

// estourouMemoria: com limite no Job, o abortamento 0xC0000409 é falta de
// memória (F0-02; D-09.2).
func (p *protecaoJob) estourouMemoria(err error) bool {
	var ee *exec.ExitError
	if p.limite == 0 || !errors.As(err, &ee) {
		return false
	}
	return uint32(ee.ExitCode()) == win.StatusFalhaRapida
}
