//go:build windows

package win

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Job é um Windows Job Object com KILL_ON_JOB_CLOSE: os processos atribuídos
// morrem quando o handle fecha, inclusive se o app for morto à força (plano
// §9; F0-02). Opcionalmente limita a memória privada do conjunto.
type Job struct{ h windows.Handle }

// StatusFalhaRapida é o código de saída de um processo abortado ao estourar o
// limite de memória do Job (0xC0000409, STATUS_STACK_BUFFER_OVERRUN/fail-fast).
const StatusFalhaRapida = 0xC0000409

// NovoJob cria o Job; limiteBytes = 0 não limita memória.
func NovoJob(limiteBytes uint64) (*Job, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if limiteBytes > 0 {
		info.BasicLimitInformation.LimitFlags |= windows.JOB_OBJECT_LIMIT_JOB_MEMORY
		info.JobMemoryLimit = uintptr(limiteBytes)
	}
	if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		windows.CloseHandle(h)
		return nil, err
	}
	return &Job{h: h}, nil
}

// Atribuir coloca o processo no Job.
func (j *Job) Atribuir(pid int) error {
	if j == nil || j.h == 0 {
		return errors.New("job fechado")
	}
	p, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(p)
	return windows.AssignProcessToJobObject(j.h, p)
}

// PicoMemoria devolve o pico de memória privada do Job, em bytes. Pesos
// mapeados por mmap não entram nesta conta (F0-02).
func (j *Job) PicoMemoria() uint64 {
	if j == nil || j.h == 0 {
		return 0
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	if err := windows.QueryInformationJobObject(j.h, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil); err != nil {
		return 0
	}
	return uint64(info.PeakJobMemoryUsed)
}

// Fechar fecha o handle; os processos do Job são encerrados.
func (j *Job) Fechar() error {
	if j == nil || j.h == 0 {
		return nil
	}
	err := windows.CloseHandle(j.h)
	j.h = 0
	return err
}

type memoryStatusEx struct {
	Length, MemoryLoad                                 uint32
	TotalPhys, AvailPhys, TotalPageFile, AvailPageFile uint64
	TotalVirtual, AvailVirtual, AvailExtendedVirtual   uint64
}

var procGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

// Memoria devolve a RAM física total e a disponível, em bytes.
func Memoria() (total, livre uint64, err error) {
	var m memoryStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	r, _, e := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m)))
	if r == 0 {
		return 0, 0, e
	}
	return m.TotalPhys, m.AvailPhys, nil
}
