//go:build windows

package win

import (
	"errors"

	"golang.org/x/sys/windows"
)

const stillActive = 259 // STILL_ACTIVE

// InicioProcesso devolve o instante de criação do processo (FILETIME em
// unidades de 100 ns). Junto com o PID, identifica o processo sem confundi-lo
// com outro que tenha reaproveitado o mesmo PID.
func InicioProcesso(pid int) (int64, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(h)
	var criacao, saida, kernel, usuario windows.Filetime
	if err := windows.GetProcessTimes(h, &criacao, &saida, &kernel, &usuario); err != nil {
		return 0, err
	}
	return int64(criacao.HighDateTime)<<32 | int64(criacao.LowDateTime), nil
}

// ProcessoVivo informa se o processo (pid, inicio) ainda está em execução.
// Na dúvida por falta de permissão, responde "vivo" (não apaga o que pode
// pertencer a outra instância).
func ProcessoVivo(pid int, inicio int64) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer windows.CloseHandle(h)
	var codigo uint32
	if err := windows.GetExitCodeProcess(h, &codigo); err != nil || codigo != stillActive {
		return false
	}
	var criacao, saida, kernel, usuario windows.Filetime
	if err := windows.GetProcessTimes(h, &criacao, &saida, &kernel, &usuario); err != nil {
		return true
	}
	return int64(criacao.HighDateTime)<<32|int64(criacao.LowDateTime) == inicio
}
