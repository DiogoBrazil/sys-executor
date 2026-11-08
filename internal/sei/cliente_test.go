package sei

import (
	"context"
	"errors"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/seisim"
)

var pdfFicticio = []byte("%PDF-1.4\n% autos fictícios\n%%EOF\n")

func navegadorDeTeste(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		if p, err := LocalizarNavegador(); err == nil {
			return p
		}
	}
	for _, n := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	t.Skip("nenhum navegador Chromium disponível")
	return ""
}

// usuarioFalso responde como o usuário responderia na tela do app.
type usuarioFalso struct {
	mu       sync.Mutex
	fases    []Fase
	login    []Login
	senhas   []string // devolvidas em sequência nas credenciais
	alertas  []erros.Codigo
	reautent int
	processo string
}

func (u *usuarioFalso) Fase(f Fase) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.fases) == 0 || u.fases[len(u.fases)-1] != f {
		u.fases = append(u.fases, f)
	}
}

func (u *usuarioFalso) anotar(a *erros.Erro) {
	if a != nil {
		u.alertas = append(u.alertas, a.Codigo)
	}
}

func (u *usuarioFalso) Credenciais(ctx context.Context, alerta *erros.Erro) (Login, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.anotar(alerta)
	senha := u.senhas[min(len(u.login), len(u.senhas)-1)]
	l := Login{Usuario: "usuario.teste", Senha: session.NovoSegredo([]byte(senha)), Orgao: "PM", Processo: u.processo}
	u.login = append(u.login, l)
	return l, nil
}

func (u *usuarioFalso) CodigoOTP(ctx context.Context, alerta *erros.Erro) (*session.Segredo, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.anotar(alerta)
	return session.NovoSegredo([]byte("123456")), nil
}

func (u *usuarioFalso) SenhaReautenticacao(ctx context.Context, alerta *erros.Erro) (*session.Segredo, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.anotar(alerta)
	u.reautent++
	return session.NovoSegredo([]byte("senha-certa")), nil
}

func cenario(t *testing.T, processo string, senhas ...string) (*seisim.Simulador, *usuarioFalso, []byte, error, string) {
	t.Helper()
	nav := navegadorDeTeste(t)
	sim := seisim.Novo(pdfFicticio)
	srv := httptest.NewServer(sim)
	t.Cleanup(srv.Close)
	u := &usuarioFalso{senhas: senhas, processo: processo}
	pasta := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	b, err := ObterPDF(ctx, Config{
		Navegador: nav, URLLogin: srv.URL + "/sip/login.php?sigla_orgao_sistema=RO&sigla_sistema=SEI",
		PastaTrabalho: pasta, PermitirHTTP: true, TempoTela: 20 * time.Second, TempoGeracao: 30 * time.Second,
	}, u)
	return sim, u, b, err, pasta
}

func semEscrita(t *testing.T, sim *seisim.Simulador) {
	t.Helper()
	for _, a := range []string{"procedimento_excluir", "procedimento_concluir", "documento_assinar", "2fa_alterado"} {
		if n := sim.Recebidas(a); n != 0 {
			t.Errorf("SEG13-07: a ação %q chegou ao SEI %d vez(es)", a, n)
		}
	}
}

func TestObterPDFComOTP(t *testing.T) {
	sim, u, b, err, pasta := cenario(t, "0000.000001/2026-01", "senha-certa")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(pdfFicticio) {
		t.Fatalf("PDF diferente: %q", b)
	}
	esperado := []Fase{FaseConectando, FaseLogin, FaseOTP, FaseLocalizando, FaseGerando, FaseBaixando}
	if len(u.fases) != len(esperado) {
		t.Fatalf("fases: %v", u.fases)
	}
	for i := range esperado {
		if u.fases[i] != esperado[i] {
			t.Fatalf("fases: %v", u.fases)
		}
	}
	if !u.login[0].Senha.Vazio() {
		t.Error("a senha deve ser zerada depois do uso")
	}
	if sim.Recebidas("exibir_arquivo") != 1 || u.reautent != 0 {
		t.Errorf("download=%d reautent=%d", sim.Recebidas("exibir_arquivo"), u.reautent)
	}
	semEscrita(t, sim)
	if es, _ := os.ReadDir(pasta + "/baixados"); len(es) != 0 {
		t.Errorf("o PDF em claro deve ser apagado após a leitura: %v", es)
	}
}

func TestObterPDFSigilosoPedeSenhaDeNovo(t *testing.T) {
	sim, u, b, err, _ := cenario(t, "0000.000002/2026-02", "senha-certa")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(pdfFicticio) || u.reautent != 1 {
		t.Fatalf("reautent=%d", u.reautent)
	}
	semEscrita(t, sim)
}

func TestSenhaErradaPedeDeNovo(t *testing.T) {
	sim, u, _, err, _ := cenario(t, "0000.000001/2026-01", "senha-errada", "senha-certa")
	if err != nil {
		t.Fatal(err)
	}
	if len(u.login) != 2 || len(u.alertas) == 0 || u.alertas[0] != erros.SEICredencial {
		t.Fatalf("logins=%d alertas=%v", len(u.login), u.alertas)
	}
	semEscrita(t, sim)
}

func TestProcessoNaoEncontrado(t *testing.T) {
	_, _, _, err, _ := cenario(t, "9999.999999/2026-99", "senha-certa")
	if !errors.Is(err, erros.Novo(erros.SEIProcessoNaoAchado)) {
		t.Fatalf("esperado SEI-008, veio %v", err)
	}
}

// T03: sem conexão → REDE-001 (o app oferece o envio do PDF).
func TestSemConexao(t *testing.T) {
	nav := navegadorDeTeste(t)
	// Porta alta recém-liberada: conexão recusada (a porta 1 é "insegura" para o Chromium).
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endereco := "http://" + l.Addr().String() + "/sip/login.php"
	l.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err = ObterPDF(ctx, Config{
		Navegador: nav, URLLogin: endereco, PastaTrabalho: t.TempDir(),
		PermitirHTTP: true, TempoTela: 10 * time.Second,
	}, &usuarioFalso{senhas: []string{"x"}})
	if !errors.Is(err, erros.Novo(erros.RedeSemConexao)) {
		t.Fatalf("esperado REDE-001, veio %v", err)
	}
}

// Tela fora do mapeamento: SEI-005 com o resumo estrutural da tela (diagnóstico).
func TestTelaDesconhecidaTrazResumo(t *testing.T) {
	nav := navegadorDeTeste(t)
	sim := seisim.Novo(pdfFicticio)
	sim.TelaDesconhecida = true
	srv := httptest.NewServer(sim)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := ObterPDF(ctx, Config{
		Navegador: nav, URLLogin: srv.URL + "/sip/login.php", PastaTrabalho: t.TempDir(),
		PermitirHTTP: true, TempoTela: 3 * time.Second,
	}, &usuarioFalso{senhas: []string{"senha-certa"}, processo: "x"})
	if !errors.Is(err, erros.Novo(erros.SEIInterfaceAlterada)) {
		t.Fatalf("esperado SEI-005, veio %v", err)
	}
	msg := err.Error()
	for _, esperado := range []string{"Aviso de manutenção ####", "txtOutroCampo", "Prosseguir"} {
		if !strings.Contains(msg, esperado) {
			t.Errorf("resumo sem %q: %s", esperado, msg)
		}
	}
	if strings.Contains(msg, "senha-certa") {
		t.Fatal("o resumo não pode conter valores digitados")
	}
}
