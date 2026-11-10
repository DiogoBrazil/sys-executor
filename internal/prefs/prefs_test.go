package prefs

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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
	if p.ModeloGemini != "" {
		t.Fatalf("nenhum modelo escolhido na primeira instalação: %q", p.ModeloGemini)
	}
}

func TestAtualizarPersisteEValida(t *testing.T) {
	arq := filepath.Join(t.TempDir(), "sub", "p.json")
	l, _ := Abrir(arq)
	if _, err := l.Atualizar(func(p *Preferencias) { p.PesquisaWeb = true; p.ModeloGemini = "gemini-3.6-flash" }); err != nil {
		t.Fatal(err)
	}
	l2, _ := Abrir(arq)
	if got := l2.Obter(); !got.PesquisaWeb || got.ModeloGemini != "gemini-3.6-flash" {
		t.Fatalf("não persistiu: %+v", got)
	}
	for _, f := range []func(*Preferencias){
		func(p *Preferencias) { p.PastaDownload = "relativa" },
		func(p *Preferencias) { p.ModeloGemini = "../../etc" },
		func(p *Preferencias) { p.ModeloGemini = "qwen3.5-4b" },
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
		"extra.json":    `{"versao":2,"pesquisa_web":true,"processo":"0021.000000/2026-00"}`,
		"extra-v1.json": `{"versao":1,"pesquisa_web":true,"processo":"0021.000000/2026-00","gpu":{"estado":"estavel"}}`,
		"chave.json":    `{"versao":2,"modelo_gemini":"gemini-x","chave_api":"AIza..."}`,
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
	// ProvedorIA e ModeloChatGPT (v3, V4 §48.1): configuração da IA, revisados.
	permitidos := map[string]bool{"Versao": true, "PesquisaWeb": true, "PastaDownload": true, "ModeloGemini": true,
		"ProvedorIA": true, "ModeloChatGPT": true}
	tp := reflect.TypeOf(Preferencias{})
	for i := 0; i < tp.NumField(); i++ {
		if !permitidos[tp.Field(i).Name] {
			t.Errorf("campo novo %q exige revisão (V4 §29.3: preferências não guardam dados de tarefa)", tp.Field(i).Name)
		}
	}
}

// D-10: o formato da versão 1 (inferência local) é migrado, preservando
// pesquisa web e pasta de download e descartando modelo local e GPU.
func TestMigraVersao1(t *testing.T) {
	arq := filepath.Join(t.TempDir(), "p.json")
	pasta := filepath.Join(t.TempDir(), "downloads")
	v1 := `{"versao":1,"pesquisa_web":true,"pasta_download":` + strconv.Quote(pasta) +
		`,"modelo_manual":"qwen3.5-4b-q4km","gpu":{"estado":"instavel","sonda_em_andamento":false}}`
	if err := os.WriteFile(arq, []byte(v1), 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := Abrir(arq)
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Obter(); got.Versao != versaoAtual || !got.PesquisaWeb || got.PastaDownload != pasta || got.ModeloGemini != "" {
		t.Fatalf("migração: %+v", got)
	}
	b, _ := os.ReadFile(arq)
	if strings.Contains(string(b), "gpu") || strings.Contains(string(b), "modelo_manual") {
		t.Fatalf("campos da inferência local devem sumir do arquivo: %s", b)
	}
}

// V4 §48.1: o formato da versão 2 (só Gemini) é migrado sem perder nada;
// quem já escolheu um modelo Gemini fica com a Gemini API como provedor.
func TestMigraVersao2(t *testing.T) {
	dir := t.TempDir()
	pasta := filepath.Join(dir, "downloads")
	casos := []struct {
		v2       string
		provedor string
		modelo   string
	}{
		{`{"versao":2,"pesquisa_web":true,"pasta_download":` + strconv.Quote(pasta) + `,"modelo_gemini":"gemini-3.6-flash"}`, ProvedorGemini, "gemini-3.6-flash"},
		{`{"versao":2,"pesquisa_web":false,"pasta_download":"","modelo_gemini":""}`, "", ""},
	}
	for i, c := range casos {
		arq := filepath.Join(dir, strconv.Itoa(i)+".json")
		if err := os.WriteFile(arq, []byte(c.v2), 0o600); err != nil {
			t.Fatal(err)
		}
		l, err := Abrir(arq)
		if err != nil {
			t.Fatal(err)
		}
		got := l.Obter()
		if got.Versao != versaoAtual || got.ProvedorIA != c.provedor || got.ModeloGemini != c.modelo || got.ModeloChatGPT != "" {
			t.Fatalf("caso %d: %+v", i, got)
		}
		if i == 0 && (!got.PesquisaWeb || got.PastaDownload != pasta) {
			t.Fatalf("v2 perdeu campos: %+v", got)
		}
		b, _ := os.ReadFile(arq)
		if !strings.Contains(string(b), `"versao": 3`) && !strings.Contains(string(b), `"versao":3`) {
			t.Fatalf("arquivo não regravado na v3: %s", b)
		}
	}
}

func TestProvedorInvalidoVoltaAoPadrao(t *testing.T) {
	arq := filepath.Join(t.TempDir(), "p.json")
	_ = os.WriteFile(arq, []byte(`{"versao":3,"pesquisa_web":false,"pasta_download":"","provedor_ia":"OUTRO","modelo_gemini":"","modelo_chatgpt":""}`), 0o600)
	l, _ := Abrir(arq)
	if !reflect.DeepEqual(l.Obter(), Padrao()) {
		t.Fatalf("%+v", l.Obter())
	}
	p := Padrao()
	p.ProvedorIA, p.ModeloChatGPT = ProvedorChatGPT, "gpt-6.1-sol"
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	p.ModeloChatGPT = "../../x"
	if p.Validar() == nil {
		t.Fatal("slug inválido deveria ser recusado")
	}
}
