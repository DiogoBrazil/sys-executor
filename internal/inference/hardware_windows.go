//go:build windows

package inference

import (
	"unsafe"

	"github.com/DiogoBrazil/sys-executor/internal/platform/win"
	"golang.org/x/sys/windows"
)

func memoriaSistema() (total, livre uint64, err error) { return win.Memoria() }

var procGetLogicalProcessorInformation = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetLogicalProcessorInformation")

// nucleosFisicos conta as entradas RelationProcessorCore.
func nucleosFisicos() int {
	// SYSTEM_LOGICAL_PROCESSOR_INFORMATION no x64: ProcessorMask (8),
	// Relationship (4) + preenchimento (4), união (16) = 32 bytes.
	const tam = 32
	var n uint32
	procGetLogicalProcessorInformation.Call(0, uintptr(unsafe.Pointer(&n)))
	if n == 0 {
		return 0
	}
	buf := make([]byte, n)
	r, _, _ := procGetLogicalProcessorInformation.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return 0
	}
	nucleos := 0
	for off := 0; off+tam <= int(n); off += tam {
		if *(*uint32)(unsafe.Pointer(&buf[off+8])) == 0 { // RelationProcessorCore
			nucleos++
		}
	}
	return nucleos
}
