//go:build windows

package win

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// fileAllAccess = FILE_ALL_ACCESS (STANDARD_RIGHTS_REQUIRED | SYNCHRONIZE | 0x1FF).
// Usar o direito específico (e não GENERIC_ALL) evita que o Windows desmembre
// a ACE herdável em duas.
const fileAllAccess = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0x1FF

// SIDUsuarioAtual devolve o SID da conta que executa o processo.
func SIDUsuarioAtual() (*windows.SID, error) {
	tu, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("token do usuário: %w", err)
	}
	return tu.User.Sid.Copy()
}

// ProtegerDiretorio substitui a DACL de dir por uma DACL **protegida** (sem
// herança do pai) com uma única entrada: controle total para o usuário atual,
// herdável por arquivos e subpastas. Administradores e SYSTEM não recebem
// acesso explícito (V4 §§29.2, 33; teste SEG-DIR-01).
func ProtegerDiretorio(dir string) error {
	sid, err := SIDUsuarioAtual()
	if err != nil {
		return err
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: fileAllAccess,
		AccessMode:        windows.SET_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}}, nil)
	if err != nil {
		return fmt.Errorf("montar ACL: %w", err)
	}
	info := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION | windows.OWNER_SECURITY_INFORMATION)
	if err := windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, info, sid, nil, acl, nil); err != nil {
		return fmt.Errorf("aplicar ACL: %w", err)
	}
	return nil
}

// EntradaACL descreve uma ACE da DACL (para verificação em testes).
type EntradaACL struct {
	Tipo    uint8 // ACCESS_ALLOWED_ACE_TYPE = 0, ACCESS_DENIED_ACE_TYPE = 1
	Flags   uint8
	Mascara uint32
	SID     string
}

// DescricaoACL é a DACL de um objeto e se ela está protegida contra herança.
type DescricaoACL struct {
	Protegida bool
	Entradas  []EntradaACL
}

// LerACL lê a DACL de um arquivo ou diretório.
func LerACL(caminho string) (DescricaoACL, error) {
	var out DescricaoACL
	sd, err := windows.GetNamedSecurityInfo(caminho, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return out, err
	}
	ctl, _, err := sd.Control()
	if err != nil {
		return out, err
	}
	out.Protegida = ctl&windows.SE_DACL_PROTECTED != 0
	dacl, _, err := sd.DACL()
	if err != nil {
		return out, err
	}
	if dacl == nil {
		return out, fmt.Errorf("DACL nula (acesso irrestrito)")
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return out, err
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		out.Entradas = append(out.Entradas, EntradaACL{
			Tipo: ace.Header.AceType, Flags: ace.Header.AceFlags, Mascara: uint32(ace.Mask), SID: sid.String(),
		})
	}
	return out, nil
}
