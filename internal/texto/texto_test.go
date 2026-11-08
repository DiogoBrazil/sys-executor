package texto

import "testing"

func TestNormalizacao(t *testing.T) {
	if SemAcentos("TERMO DE INQUIRIÇÃO") != "termo de inquiricao" {
		t.Fatal(SemAcentos("TERMO DE INQUIRIÇÃO"))
	}
	s := "Câmera: ÁUDIO e vídeo"
	a := Alinhado(s)
	if len(a) != len([]rune(s)) || string(a) != "camera: audio e video" {
		t.Fatalf("%q", string(a))
	}
	if got := Termos("Art. 16, XVI — prevenção"); len(got) != 4 || got[3] != "prevencao" {
		t.Fatalf("%q", got)
	}
}
