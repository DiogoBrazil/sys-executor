package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestCarregarEmbutido(t *testing.T) {
	c, err := Carregar()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Regras) < 185 {
		t.Fatalf("%d regras; a matriz semeou 185", len(c.Regras))
	}
	r, ok := c.PorID("PADS-REI-003")
	if !ok || !r.Tem(Deterministica) || r.Fluxo != "PADS" || !r.Vigente() {
		t.Fatalf("%+v", r)
	}
	if len(c.Testes()["R2"]) == 0 {
		t.Fatal("R2 deveria estar mapeado")
	}
}

const valida = `
- id: PADS-REI-900
  origem: "§1"
  fluxo: PADS
  rotulo: [APROVADO]
  tipo: [DETERMINISTICA]
  enunciado: "x"
  modulo: "engine"
  testes: ["R1"]
  versao: 1
`

func carregarTexto(arqs map[string]string) error {
	fs := fstest.MapFS{}
	for n, s := range arqs {
		fs[n] = &fstest.MapFile{Data: []byte(s)}
	}
	_, err := CarregarFS(fs)
	return err
}

func TestValidacao(t *testing.T) {
	if err := carregarTexto(map[string]string{"a.yaml": valida}); err != nil {
		t.Fatal(err)
	}
	casos := map[string]string{
		"campo desconhecido": strings.Replace(valida, "versao: 1", "versao: 1\n  inventado: x", 1),
		"fluxo × prefixo":    strings.Replace(valida, "fluxo: PADS", "fluxo: SR", 1),
		"rótulo":             strings.Replace(valida, "[APROVADO]", "[TALVEZ]", 1),
		"tipo":               strings.Replace(valida, "[DETERMINISTICA]", "[MAGICA]", 1),
		"sem testes":         strings.Replace(valida, `["R1"]`, `[]`, 1),
		"teste repetido":     strings.Replace(valida, `["R1"]`, `["R1", "R1"]`, 1),
		"versão":             strings.Replace(valida, "versao: 1", "versao: 0", 1),
		"símbolo":            strings.Replace(valida, "versao: 1", "versao: 1\n  enforced_by: [Reincidencia]", 1),
		"instrução fora IA":  strings.Replace(valida, "versao: 1", "versao: 1\n  instrucao_ia: \"NUNCA x\"", 1),
		"status":             strings.Replace(valida, "versao: 1", "versao: 1\n  status: suspensa", 1),
		"ID":                 strings.Replace(valida, "PADS-REI-900", "PADS-REI-9", 1),
		"interpretação":      strings.Replace(valida, "versao: 1", "versao: 1\n  interpretacao: I-01", 1),
		"etapa":              strings.Replace(valida, "versao: 1", "versao: 1\n  etapas: [inventada]", 1),
		"condição":           strings.Replace(valida, "versao: 1", "versao: 1\n  condicao: [talvez]", 1),
		"IA sem instrução":   strings.Replace(valida, "[DETERMINISTICA]", "[POLITICA_IA]", 1),
	}
	for nome, txt := range casos {
		if err := carregarTexto(map[string]string{"a.yaml": txt}); err == nil {
			t.Errorf("%s: deveria reprovar", nome)
		}
	}
	if err := carregarTexto(map[string]string{"a.yaml": valida, "b.yaml": valida}); err == nil || !strings.Contains(err.Error(), "repetido") {
		t.Errorf("ID repetido entre arquivos: %v", err)
	}
	if err := carregarTexto(map[string]string{"a.yaml": valida + "---\n" + valida}); err == nil {
		t.Error("dois documentos no mesmo arquivo")
	}
}

func TestSelecionar(t *testing.T) {
	c, _ := Carregar()
	ids := func(rs []Regra) string {
		var s []string
		for _, r := range rs {
			s = append(s, r.ID)
		}
		return strings.Join(s, " ")
	}
	nuc := ids(c.Selecionar("PADS", EtapaNucleo, nil))
	for _, id := range []string{"TRV-ESC-001", "TRV-PRV-006", "TRV-PRV-007"} {
		if !strings.Contains(nuc, id) {
			t.Errorf("núcleo sem %s: %s", id, nuc)
		}
	}
	sem := ids(c.Selecionar("PADS", EtapaRedacao, nil))
	com := ids(c.Selecionar("PADS", EtapaRedacao, map[string]bool{CondTemPunicao: true}))
	if strings.Contains(sem, "PADS-DOS-001") || !strings.Contains(com, "PADS-DOS-001") {
		t.Fatalf("condição tem_punicao: %s | %s", sem, com)
	}
	if a := ids(c.Selecionar("SR", EtapaAnalise, nil)); strings.Contains(a, "PADS-") || !strings.Contains(a, "SR-SOL-004") {
		t.Fatalf("SR recebeu regra do PADS: %s", a)
	}
	for _, e := range Etapas {
		if e != EtapaAjuste && e != EtapaValidacao && len(c.Selecionar("PADS", e, nil))+len(c.Selecionar("SR", e, nil)) == 0 {
			t.Errorf("etapa %s sem regras", e)
		}
	}
}
