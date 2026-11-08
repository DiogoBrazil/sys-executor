package app

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

func servico(t *testing.T, sel SeletorPDF, ler func(string) ([]byte, error)) *TarefaService {
	t.Helper()
	g := workflow.Novo(workflow.Opcoes{Sessoes: session.NovoGerenciador(t.TempDir())})
	t.Cleanup(g.Encerrar)
	s := NovaTarefaService(g, sel)
	if ler != nil {
		s.lerArquivo = ler
	}
	return s
}

// Nenhuma resposta leva a causa técnica à interface.
func TestRespostaSemCausaTecnica(t *testing.T) {
	s := servico(t,
		func() (string, error) { return `C:\autos\0021.000000-2026-00.pdf`, nil },
		func(string) ([]byte, error) { return nil, errors.New("open C:\\autos\\SEGREDO: acesso negado") })
	s.Iniciar()
	s.EscolherTipo("PADS")
	s.EscolherEntrada("UPLOAD")
	r := s.SelecionarPDF()
	if r.Erro == nil || r.Erro.Codigo != "ARQ-001" {
		t.Fatalf("%+v", r.Erro)
	}
	b, _ := json.Marshal(r)
	for _, proibido := range []string{"SEGREDO", "0021.000000", `C:\\autos`, "acesso negado"} {
		if strings.Contains(string(b), proibido) {
			t.Fatalf("resposta expõe %q: %s", proibido, b)
		}
	}
	if r.Visao == nil || r.Visao.Estado != workflow.RecebendoUpload {
		t.Fatalf("falha de leitura deve manter a tarefa aguardando: %+v", r.Visao)
	}
}

func TestSelecionarPDFCanceladoEOrdem(t *testing.T) {
	chamado := false
	s := servico(t, func() (string, error) { chamado = true; return "", nil }, nil)
	if r := s.SelecionarPDF(); r.Erro == nil || r.Erro.Codigo != "TAR-002" || chamado {
		t.Fatalf("sem tarefa em upload, o diálogo não deve abrir: %+v", r.Erro)
	}
	s.Iniciar()
	s.EscolherTipo("SR")
	s.EscolherEntrada("UPLOAD")
	if r := s.SelecionarPDF(); r.Erro != nil || r.Visao.Estado != workflow.RecebendoUpload || !chamado {
		t.Fatalf("cancelar o diálogo não é erro: %+v", r)
	}
}

func TestFluxoUploadPeloServico(t *testing.T) {
	s := servico(t,
		func() (string, error) { return "x.pdf", nil },
		func(string) ([]byte, error) { return []byte("%PDF-1.4\n%%EOF"), nil })
	s.Iniciar()
	s.EscolherTipo("PADS")
	s.EscolherEntrada("UPLOAD")
	if r := s.SelecionarPDF(); r.Erro != nil {
		t.Fatalf("%+v", r.Erro)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v := s.Atual().Visao; v != nil && v.Parada {
			if v.Erro == nil || v.Erro.Codigo != "TAR-004" {
				t.Fatalf("%+v", v)
			}
			s.Cancelar()
			if r := s.Descartar(); r.Erro != nil || s.Atual().Visao != nil {
				t.Fatal("descartar deveria voltar ao início")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("tarefa não terminou")
}

func TestPreferenciasService(t *testing.T) {
	l, _ := prefs.Abrir(filepath.Join(t.TempDir(), "p.json"))
	s := NovaPreferenciasService(l)
	if s.Obter().Preferencias.PesquisaWeb {
		t.Fatal("T19: deve começar desligada")
	}
	if r := s.DefinirPesquisaWeb(true); r.Erro != nil || !r.Preferencias.PesquisaWeb {
		t.Fatalf("%+v", r)
	}
}
