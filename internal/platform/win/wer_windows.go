//go:build windows

package win

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Exclusão do Windows Error Reporting por usuário (plano §13, "Dumps";
// pendência P-10). Impede que uma falha do app gere relatórios/dumps de
// memória com conteúdo dos autos. Não exige administrador.

var (
	modwer                        = windows.NewLazySystemDLL("wer.dll")
	procWerAddExcludedApplication = modwer.NewProc("WerAddExcludedApplication")
	procWerRemoveExcludedApp      = modwer.NewProc("WerRemoveExcludedApplication")
)

const chaveExclusoesWER = `Software\Microsoft\Windows\Windows Error Reporting\ExcludedApplications`

func chamarWER(proc *windows.LazyProc, nomeExe string) error {
	if err := proc.Find(); err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(nomeExe)
	if err != nil {
		return err
	}
	// HRESULT WerXxxExcludedApplication(PCWSTR pwzExeName, BOOL bAllUsers)
	hr, _, _ := proc.Call(uintptr(unsafe.Pointer(p)), 0)
	if int32(hr) < 0 {
		return fmt.Errorf("HRESULT 0x%08X", uint32(hr))
	}
	return nil
}

// ExcluirDoWER exclui nomeExe (só o nome do arquivo, ex. "sys-executor.exe")
// do WER para o usuário atual.
func ExcluirDoWER(nomeExe string) error {
	return chamarWER(procWerAddExcludedApplication, nomeExe)
}

// RemoverExclusaoWER desfaz ExcluirDoWER (usado em testes e na desinstalação).
func RemoverExclusaoWER(nomeExe string) error {
	return chamarWER(procWerRemoveExcludedApp, nomeExe)
}

// ExclusaoWERAtiva confere no registro do usuário se a exclusão está presente.
func ExclusaoWERAtiva(nomeExe string) (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, chaveExclusoesWER, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue(nomeExe)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == 1, nil
}
