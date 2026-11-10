package app

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/credencial"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
)

const chaveSentinela = "AIzaSyCHAVE_SENTINELA_app_0123456789"

var modelosTeste = []inference.ModeloGemini{
	{ID: "gemini-3.6-flash", Nome: "Gemini 3.6 Flash", Estagio: inference.EstagioEstavel, Pensamento: true},
	{ID: "gemini-3.8-flash-preview", Nome: "Prévia", Estagio: inference.EstagioPrevia},
}

type motorUsoFalso struct {
	fechado bool
}

func (m *motorUsoFalso) Gerar(context.Context, inference.Pedido) (inference.Resposta, error) {
	return inference.Resposta{}, nil
}
func (m *motorUsoFalso) ContarTokens(_ context.Context, s string) (int, error) { return len(s), nil }
func (m *motorUsoFalso) Contexto() int                                         { return 32768 }
func (m *motorUsoFalso) Uso() inference.Uso {
	return inference.Uso{Modelo: "gemini-3.6-flash", Chamadas: 3, TokensEntrada: 900}
}
func (m *motorUsoFalso) Fechar() { m.fechado = true }

type cenario struct {
	cred      *credencial.LojaMemoria
	loja      *prefs.Loja
	fonte     *FonteGemini
	serv      *IAService
	chaves    []string // chaves recebidas pelas funções de rede
	listarErr error
	testarErr error
	emitidos  []VisaoIA
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()
	c := &cenario{cred: &credencial.LojaMemoria{}}
	c.loja, _ = prefs.Abrir(filepath.Join(t.TempDir(), "p.json"))
	c.fonte = &FonteGemini{Credenciais: c.cred, Loja: c.loja,
		Listar: func(_ context.Context, cfg inference.ConfigGemini) ([]inference.ModeloGemini, error) {
			c.chaves = append(c.chaves, string(cfg.Chave))
			return modelosTeste, c.listarErr
		},
		Testar: func(_ context.Context, cfg inference.ConfigGemini) error {
			c.chaves = append(c.chaves, string(cfg.Chave))
			return c.testarErr
		},
		NovoMotor: func(_ context.Context, cfg inference.ConfigGemini) (MotorComUso, error) {
			c.chaves = append(c.chaves, string(cfg.Chave))
			return &motorUsoFalso{}, nil
		},
	}
	c.serv = &IAService{Fonte: c.fonte, Emitir: func(v *VisaoIA) { c.emitidos = append(c.emitidos, *v) }}
	return c
}

func semChave(t *testing.T, vs ...VisaoIA) {
	t.Helper()
	for _, v := range vs {
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), chaveSentinela) || strings.Contains(string(b), "SENTINELA") {
			t.Fatalf("a chave apareceu na visão enviada à interface: %s", b)
		}
	}
}

// Regra: TRV-SEG-009
func TestSalvarChaveValidaEGuarda(t *testing.T) {
	c := novoCenario(t)
	if v := c.serv.Estado(); v.ChaveCadastrada || v.Status != StatusSemChave || v.URLChaves != URLChavesAIStudio {
		t.Fatalf("estado inicial: %+v", v)
	}
	v := c.serv.SalvarChave("  " + chaveSentinela + "\n")
	if !v.ChaveCadastrada || v.Status != StatusValida || len(v.Modelos) != 2 || v.Erro != nil || v.Ocupado {
		t.Fatalf("após salvar: %+v", v)
	}
	if got, _ := c.cred.Ler(); string(got) != chaveSentinela {
		t.Fatal("chave não guardada (ou guardada com espaços)")
	}
	if c.chaves[0] != chaveSentinela {
		t.Fatal("validação deve usar a chave informada")
	}
	semChave(t, append(c.emitidos, v, c.serv.Estado())...)
}

func TestChaveRecusadaNaoEhGuardada(t *testing.T) {
	c := novoCenario(t)
	c.listarErr = erros.Novo(erros.IAChaveInvalida)
	v := c.serv.SalvarChave(chaveSentinela)
	if v.ChaveCadastrada || v.Erro == nil || v.Erro.Codigo != "IA-002" {
		t.Fatalf("%+v", v)
	}
	if c.cred.Existe() {
		t.Fatal("chave recusada foi guardada")
	}
	c.listarErr = erros.Novo(erros.IASemConexao)
	if v := c.serv.SalvarChave(chaveSentinela); c.cred.Existe() || v.Erro.Codigo != "IA-007" {
		t.Fatalf("sem conexão não valida nem guarda: %+v", v)
	}
	if v := c.serv.SalvarChave("curta"); v.Erro == nil || v.Erro.Codigo != "IA-002" || len(c.chaves) != 2 {
		t.Fatalf("formato inválido recusado sem rede: %+v", v)
	}
	semChave(t, c.emitidos...)
}

func TestRemoverEVerificar(t *testing.T) {
	c := novoCenario(t)
	c.serv.SalvarChave(chaveSentinela)
	c.listarErr = erros.Novo(erros.IAChaveInvalida)
	if v := c.serv.Verificar(); v.Status != StatusInvalida || v.Erro.Codigo != "IA-002" {
		t.Fatalf("chave revogada: %+v", v)
	}
	c.listarErr = erros.Novo(erros.IASemConexao)
	if v := c.serv.Verificar(); v.Status != StatusSemConexao {
		t.Fatalf("sem conexão: %+v", v)
	}
	v := c.serv.RemoverChave()
	if v.ChaveCadastrada || v.Status != StatusSemChave || len(v.Modelos) != 0 || c.cred.Existe() {
		t.Fatalf("remover: %+v", v)
	}
	if v := c.serv.Verificar(); v.Erro == nil || v.Erro.Codigo != "IA-001" {
		t.Fatalf("sem chave: %+v", v)
	}
}

// Regra: TRV-IA-011
func TestSelecionarETestarModelo(t *testing.T) {
	c := novoCenario(t)
	c.serv.SalvarChave(chaveSentinela)
	if v := c.serv.SelecionarModelo("gemini-inexistente"); v.Erro == nil || v.ModeloSelecionado != "" {
		t.Fatalf("modelo fora da lista: %+v", v)
	}
	v := c.serv.SelecionarModelo("gemini-3.6-flash")
	if v.ModeloSelecionado != "gemini-3.6-flash" || c.loja.Obter().ModeloGemini != "gemini-3.6-flash" {
		t.Fatalf("seleção não persistiu: %+v", v)
	}
	v = c.serv.TestarModelo("gemini-3.6-flash")
	if v.Modelos[0].Teste != "ok" {
		t.Fatalf("teste: %+v", v.Modelos[0])
	}
	c.testarErr = erros.Novo(erros.IACota)
	v = c.serv.TestarModelo("gemini-3.8-flash-preview")
	if v.Modelos[1].Teste != "falhou" || v.Modelos[1].TesteErro.Codigo != "IA-005" {
		t.Fatalf("teste com cota esgotada: %+v", v.Modelos[1])
	}
	semChave(t, c.emitidos...)
}

func TestFonteGemini(t *testing.T) {
	c := novoCenario(t)
	if _, err := c.fonte.Modelo(); !errors.Is(err, erros.Novo(erros.IAChaveAusente)) {
		t.Fatalf("sem chave: %v", err)
	}
	c.serv.SalvarChave(chaveSentinela)
	if _, err := c.fonte.Modelo(); !errors.Is(err, erros.Novo(erros.IAModeloIndisponivel)) {
		t.Fatalf("sem modelo: %v", err)
	}
	c.serv.SelecionarModelo("gemini-3.6-flash")
	if m, err := c.fonte.Modelo(); err != nil || m != "gemini-3.6-flash" {
		t.Fatalf("modelo: %q %v", m, err)
	}
	var atividades []string
	m, fechar, err := c.fonte.Abrir(context.Background(), func(s string) { atividades = append(atividades, s) }, nil)
	if err != nil || m == nil || len(atividades) != 1 {
		t.Fatalf("abrir: %v %v", err, atividades)
	}
	if c.chaves[len(c.chaves)-1] != chaveSentinela {
		t.Fatal("o motor deve receber a chave guardada")
	}
	v := c.serv.Estado()
	if v.Uso == nil || v.Uso.Chamadas != 3 || v.Uso.TokensEntrada != 900 {
		t.Fatalf("uso da última tarefa: %+v", v.Uso)
	}
	fechar()
	if !m.(*motorUsoFalso).fechado {
		t.Fatal("fechar deve descartar o motor")
	}
}
