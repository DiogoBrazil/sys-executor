//go:build !windows

package session

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Equivalentes para desenvolvimento fora do Windows (o produto é Windows x64).

func protegerDiretorio(dir string) error { return os.Chmod(dir, 0o700) }

// inicioProcesso lê o campo 22 (starttime) de /proc/<pid>/stat no Linux.
// Sem /proc, devolve 0 e a verificação usa só a existência do PID.
func inicioProcesso(pid int) (int64, error) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if _, e := os.Stat("/proc/self"); e != nil {
				return 0, nil
			}
		}
		return 0, err
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return 0, errors.New("stat inválido")
	}
	campos := strings.Fields(s[i+1:])
	if len(campos) < 20 {
		return 0, errors.New("stat inválido")
	}
	return strconv.ParseInt(campos[19], 10, 64) // campo 22 = índice 19 após ")"
}

func processoVivo(pid int, inicio int64) bool {
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return false
	}
	atual, err := inicioProcesso(pid)
	if err != nil {
		return false
	}
	return inicio == 0 || atual == inicio
}
