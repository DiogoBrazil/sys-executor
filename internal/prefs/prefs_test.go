package prefs

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPadraoPrimeiraInstalacao(t *testing.T) {
	l, err := Abrir(filepath.Join(t.TempDir(), "p.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := l.Obter()
	if p.PesquisaWeb {
		t.Fatal("T19: pesquisa web deve vir desligada na primeira instalação")
	}
	if p.GPU.Estado != GPUDesconhecida || p.GPU.SondaEmAndamento {
		t.Fatalf("GPU deve começar desconhecida (CPU por padrão): %+v", p.GPU)
	}
}

func TestAtualizarPersisteEValida(t *testing.T) {
	arq := filepath.Join(t.TempDir(), "sub", "p.json")
	l, _ := Abrir(arq)
	if _, err := l.Atualizar(func(p *Preferencias) { p.PesquisaWeb = true; p.ModeloManual = "qwen3.5-4b-q4_k_m" }); err != nil {
		t.Fatal(err)
	}
	l2, _ := Abrir(arq)
	if got := l2.Obter(); !got.PesquisaWeb || got.ModeloManual != "qwen3.5-4b-q4_k_m" {
		t.Fatalf("não persistiu: %+v", got)
	}
	for _, f := range []func(*Preferencias){
		func(p *Preferencias) { p.PastaDownload = "relativa" },
		func(p *Preferencias) { p.ModeloManual = "../../etc" },
		func(p *Preferencias) { p.GPU.Estado = "turbo" },
	} {
		antes := l.Obter()
		if _, err := l.Atualizar(f); err == nil {
			t.Fatal("valor inválido deveria ser rejeitado")
		}
		if !reflect.DeepEqual(antes, l.Obter()) {
			t.Fatal("falha de validação não pode alterar o estado")
		}
	}
	if es, _ := os.ReadDir(filepath.Dir(arq)); len(es) != 1 {
		t.Fatalf("deve restar só o arquivo final, sem temporários: %v", es)
	}
}

// Formato fechado: nada além das chaves conhecidas pode ser carregado.
func TestChaveDesconhecidaOuCorrompidoViraPadrao(t *testing.T) {
	dir := t.TempDir()
	for nome, conteudo := range map[string]string{
		"extra.json":    `{"versao":1,"pesquisa_web":true,"processo":"0021.000000/2026-00","gpu":{"estado":"estavel"}}`,
		"quebrado.json": `{"versao":1,`,
		"versao.json":   `{"versao":99}`,
	} {
		arq := filepath.Join(dir, nome)
		_ = os.WriteFile(arq, []byte(conteudo), 0o600)
		l, err := Abrir(arq)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(l.Obter(), Padrao()) {
			t.Errorf("%s: esperado padrão, veio %+v", nome, l.Obter())
		}
	}
}

// Nenhum campo livre além dos previstos (proteção contra uso indevido futuro).
func TestCamposFechados(t *testing.T) {
	permitidos := map[string]bool{"Versao": true, "PesquisaWeb": true, "PastaDownload": true, "ModeloManual": true, "GPU": true}
	tp := reflect.TypeOf(Preferencias{})
	for i := 0; i < tp.NumField(); i++ {
		if !permitidos[tp.Field(i).Name] {
			t.Errorf("campo novo %q exige revisão (V4 §29.3: preferências não guardam dados de tarefa)", tp.Field(i).Name)
		}
	}
}

// D-09.1: sentinela da sonda de GPU presente na abertura → GPU instável.
func TestSentinelaGPUMarcaInstavel(t *testing.T) {
	arq := filepath.Join(t.TempDir(), "p.json")
	l, _ := Abrir(arq)
	if _, err := l.Atualizar(func(p *Preferencias) { p.GPU.SondaEmAndamento = true }); err != nil {
		t.Fatal(err)
	}
	// "queda" durante a sonda: o app reabre sem ter limpado a sentinela.
	l2, err := Abrir(arq)
	if err != nil {
		t.Fatal(err)
	}
	if g := l2.Obter().GPU; g.Estado != GPUInstavel || g.SondaEmAndamento {
		t.Fatalf("esperado instável sem sentinela, veio %+v", g)
	}
	b, _ := os.ReadFile(arq)
	if !strings.Contains(string(b), `"instavel"`) {
		t.Fatal("estado instável deve ser persistido")
	}
}
