//go:build windows

package credencial

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/DiogoBrazil/sys-executor/internal/platform/win"
)

// proteger cifra com a DPAPI no escopo do usuário atual, sem interface.
func proteger(dado, ent []byte) ([]byte, error) {
	in := blob(dado)
	e := blob(ent)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, &e, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

// desproteger decifra; falha se o arquivo for de outra conta ou estiver alterado.
func desproteger(cif, ent []byte) ([]byte, error) {
	in := blob(cif)
	e := blob(ent)
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, &e, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	claro := unsafe.Slice(out.Data, out.Size)
	cp := append([]byte(nil), claro...)
	zerar(claro)
	return cp, nil
}

func blob(b []byte) windows.DataBlob {
	if len(b) == 0 {
		return windows.DataBlob{}
	}
	return windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
}

func protegerDiretorio(dir string) error { return win.ProtegerDiretorio(dir) }
