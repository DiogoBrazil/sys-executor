package autos

import "testing"

func TestLerCarimbos(t *testing.T) {
	txt := "corpo do documento\nPADS Memorando Disciplinar 12 (0012345678) SEI 0000.000001/2026-01 / pg. 3\n"
	cs := LerCarimbos(txt)
	if len(cs) != 1 || cs[0].Nome != "PADS Memorando Disciplinar 12" || cs[0].SEI != "0012345678" || cs[0].Processo != "0000.000001/2026-01" || cs[0].PaginaPG != 3 {
		t.Fatalf("%+v", cs)
	}
	// Sem parênteses e vários carimbos na mesma linha (documento trazido de outro processo).
	txt = "PADS CAPA 0012345670 SEI 0000.000001/2026-01 / pg. 1\n" +
		"Anexo - Termo 7 (0099999999) SEI 0000.000009/2025-09 / pg. 14 Anexo - Termo (0012345680) SEI 0000.000001/2026-01 / pg. 20"
	cs = LerCarimbos(txt)
	if len(cs) != 3 || cs[0].Nome != "PADS CAPA" || cs[0].SEI != "0012345670" || cs[1].Processo != "0000.000009/2025-09" ||
		cs[2].Nome != "Anexo - Termo" || cs[2].PaginaPG != 20 {
		t.Fatalf("%+v", cs)
	}
	if SemCarimbos("texto\nPADS CAPA 0012345670 SEI 0000.000001/2026-01 / pg. 1") != "texto" {
		t.Fatal("SemCarimbos")
	}
	// Página escaneada: só o carimbo na camada de texto não é "texto útil".
	if TextoUtil(SemCarimbos("Termo de Inquirição de testemunha 4 (0012345681) SEI 0000.000001/2026-01 / pg. 9")) {
		t.Fatal("carimbo sozinho não pode evitar o OCR")
	}
}
