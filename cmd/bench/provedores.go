package main

// Provedores do bench (plano D-11 §4.9 e etapa 8):
//
//   - gemini: Gemini API com a chave de desenvolvimento (SYSEXEC_GEMINI_API_KEY);
//   - chatgpt: ChatGPT pelo Sign in with ChatGPT, com a MESMA conta e o MESMO
//     motor do aplicativo (internal/chatgpt + inference.MotorChatGPT). A
//     credencial de desenvolvimento fica em arquivo 0600 fora do repositório
//     (decisão do usuário de 10/11/2026), porque fora do Windows não há DPAPI;
//   - falso: sem rede; responde com eco do pedido. Serve à métrica de
//     vazamento e à comparação das entradas sem gastar cota.
//
// Nos três, o corpo de cada requisição passa pela barreira de privacidade
// (camada 2, quando há pseudonimização) e depois pela métrica de vazamento.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/chatgpt"
	"github.com/DiogoBrazil/sys-executor/internal/credencial"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
)

const (
	provGemini  = "gemini"
	provChatGPT = "chatgpt"
	provFalso   = "falso"
)

// ---- credencial de desenvolvimento do ChatGPT ----

// lojaDev guarda a credencial do ChatGPT do bench em arquivo com permissão
// 0600 (pasta 0700). Só para desenvolvimento no WSL, com dados sintéticos; o
// aplicativo usa DPAPI (V4 §48.3). Nunca dentro do repositório.
type lojaDev struct{ caminho string }

func caminhoCredencialDev() (string, error) {
	if p := os.Getenv("SYSEXEC_CHATGPT_DEV"); p != "" {
		return p, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sys-executor", "chatgpt-dev.json"), nil
}

func (l lojaDev) Guardar(b []byte) error {
	dir := filepath.Dir(l.caminho)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".chatgpt-dev-*")
	if err != nil {
		return err
	}
	nome := tmp.Name()
	if err = tmp.Chmod(0o600); err == nil {
		if _, err = tmp.Write(b); err == nil {
			err = tmp.Sync()
		}
	}
	if e := tmp.Close(); err == nil {
		err = e
	}
	if err == nil {
		err = os.Rename(nome, l.caminho)
	}
	if err != nil {
		os.Remove(nome)
	}
	return err
}

func (l lojaDev) Ler() ([]byte, error) {
	b, err := os.ReadFile(l.caminho)
	if errors.Is(err, os.ErrNotExist) {
		return nil, credencial.ErrAusente
	}
	return b, err
}

func (l lojaDev) Remover() error {
	if err := os.Remove(l.caminho); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (l lojaDev) Existe() bool {
	_, err := os.Stat(l.caminho)
	return err == nil
}

func contaDev() (*chatgpt.Conta, error) {
	p, err := caminhoCredencialDev()
	if err != nil {
		return nil, err
	}
	return chatgpt.Nova(chatgpt.Config{Loja: lojaDev{caminho: p}, AbrirURL: abrirNavegador}), nil
}

// abrirNavegador abre o navegador padrão (no WSL, o do Windows). A URL de
// autorização não contém segredo (só state, nonce e o desafio PKCE); se não
// abrir sozinho, ela é mostrada para copiar.
func abrirNavegador(u string) error {
	var cmd *exec.Cmd
	switch {
	case runtime.GOOS == "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case os.Getenv("WSL_DISTRO_NAME") != "":
		cmd = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", u) // explorer.exe perde a URL
		cmd.Dir = "/mnt/c/"
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Abra no navegador para entrar:\n"+u)
		return nil
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func conectarChatGPT(ctx context.Context) error {
	c, err := contaDev()
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Abrindo o navegador para o Sign in with ChatGPT (prazo de 5 minutos)…")
	if err := c.Entrar(ctx); err != nil {
		return err
	}
	return estadoChatGPT()
}

func desconectarChatGPT(ctx context.Context) error {
	c, err := contaDev()
	if err != nil {
		return err
	}
	ok, err := c.Desconectar(ctx)
	if err != nil {
		return err
	}
	if ok {
		fmt.Fprintln(os.Stderr, "Desconectado: sessão revogada no provedor e credencial de desenvolvimento apagada.")
	} else {
		fmt.Fprintln(os.Stderr, "Credencial apagada; a revogação no provedor não foi confirmada (desconecte também nas configurações do ChatGPT).")
	}
	return nil
}

// estadoChatGPT mostra o estado sem token nem e-mail (dado pessoal).
func estadoChatGPT() error {
	c, err := contaDev()
	if err != nil {
		return err
	}
	e, err := c.Estado()
	if err != nil {
		return err
	}
	sim := map[bool]string{true: "sim", false: "não"}
	fmt.Fprintf(os.Stderr, "ChatGPT (bench): conectada %s · permissão de uso do plano %s · reconectar %s\n",
		sim[e.Conectada], sim[e.PlanoAutorizado], sim[e.Reconectar])
	return nil
}

// ---- motores ----

// novoMotor cria o motor do provedor, com o gancho de inspeção do corpo.
func novoMotor(ctx context.Context, cfg config, inspecionar func([]byte) error) (inference.Motor, func(), error) {
	switch cfg.provedor {
	case provGemini:
		chave := os.Getenv("SYSEXEC_GEMINI_API_KEY")
		if chave == "" {
			return nil, nil, errors.New("defina SYSEXEC_GEMINI_API_KEY (chave de desenvolvimento; nunca em argumento)")
		}
		g, err := inference.NovoMotorGemini(ctx, inference.ConfigGemini{Chave: []byte(chave), Modelo: cfg.modelo,
			Contexto: cfg.ctx, Pensamento: cfg.pensamento, Inspecionar: inspecionar})
		if err != nil {
			return nil, nil, err
		}
		return g, g.Fechar, nil
	case provChatGPT:
		c, err := contaDev()
		if err != nil {
			return nil, nil, err
		}
		e, err := c.Estado()
		if err != nil {
			return nil, nil, err
		}
		if !e.Conectada {
			return nil, nil, errors.New("conta ChatGPT do bench não conectada: rode antes `bench -conectar-chatgpt`")
		}
		m, err := inference.NovoMotorChatGPT(inference.ConfigChatGPT{Modelo: cfg.modelo, Tokens: c, Contexto: cfg.ctx,
			Esforco: cfg.pensamento, Inspecionar: inspecionar})
		if err != nil {
			return nil, nil, err
		}
		return m, m.Fechar, nil
	case provFalso:
		return &motorFalso{inspecionar: inspecionar, ctx: cfg.ctx}, func() {}, nil
	}
	return nil, nil, fmt.Errorf("provedor desconhecido %q (gemini, chatgpt ou falso)", cfg.provedor)
}

// listarModelos mostra os modelos disponíveis para a chave ou a conta.
func listarModelos(ctx context.Context, provedor string) error {
	switch provedor {
	case provGemini:
		chave := os.Getenv("SYSEXEC_GEMINI_API_KEY")
		if chave == "" {
			return errors.New("defina SYSEXEC_GEMINI_API_KEY")
		}
		ms, err := inference.ListarModelos(ctx, inference.ConfigGemini{Chave: []byte(chave)})
		if err != nil {
			return err
		}
		for _, m := range ms {
			fmt.Printf("%s\t%s\t%s\tentrada %d\n", m.ID, m.Estagio, m.Nome, m.EntradaMax)
		}
		return nil
	case provChatGPT:
		c, err := contaDev()
		if err != nil {
			return err
		}
		ms, err := inference.ListarModelosChatGPT(ctx, inference.ConfigChatGPT{Tokens: c})
		if err != nil {
			return err
		}
		for _, m := range ms {
			fmt.Printf("%s\t%s\tcontexto %d\traciocínio %s (padrão %s)\n", m.ID, m.Nome, m.Contexto,
				strings.Join(m.Raciocinio, ","), m.RaciocinioPadrao)
		}
		return nil
	}
	return fmt.Errorf("-modelos vale para gemini ou chatgpt")
}

// ecoDaEvidencia imita um modelo que cita a prova: devolve o texto do
// primeiro documento (<<<DOC …>>> … <<<FIM>>>); sem documento, o início da
// mensagem. Copiar o cabeçalho "DADOS ESTRUTURADOS" (como fazia a primeira
// versão do eco) gerava no histórico uma revisão que nenhum modelo real
// provoca.
func ecoDaEvidencia(u string) string {
	if i := strings.Index(u, "<<<DOC "); i >= 0 {
		resto := u[i:]
		if j := strings.Index(resto, ">>>\n"); j >= 0 {
			resto = resto[j+4:]
			if k := strings.Index(resto, "\n<<<FIM>>>"); k >= 0 {
				return resto[:k]
			}
			return resto
		}
	}
	return u
}

// motorFalso não usa rede. Monta um corpo JSON com todos os campos do pedido
// (como um provedor faria), passa-o pelo mesmo gancho de inspeção e responde
// com um eco do início da mensagem do usuário. O eco não segue o schema: o
// validador recusa e o executor reenvia com o histórico, que traz a resposta
// já RESTAURADA (com os nomes reais). Assim a métrica cobre também o caminho
// do reenvio, o mais sujeito a vazamento.
type motorFalso struct {
	inspecionar func([]byte) error
	ctx         int
}

func (m *motorFalso) Contexto() int {
	if m.ctx > 0 {
		return m.ctx
	}
	return inference.ContextoPadrao
}

func (m *motorFalso) ContarTokens(ctx context.Context, t string) (int, error) {
	return policy.ContadorEstimado{}.ContarTokens(ctx, t)
}

func (m *motorFalso) Gerar(_ context.Context, p inference.Pedido) (inference.Resposta, error) {
	corpo, err := json.Marshal(struct {
		Sistema   string               `json:"system"`
		Usuario   string               `json:"user"`
		Historico []inference.Mensagem `json:"history"`
		Schema    json.RawMessage      `json:"schema,omitempty"`
	}{p.Sistema, p.Usuario, p.Historico, p.Schema})
	if err != nil {
		return inference.Resposta{}, err
	}
	if m.inspecionar != nil {
		if err := m.inspecionar(corpo); err != nil {
			return inference.Resposta{}, err
		}
	}
	eco := []rune(ecoDaEvidencia(p.Usuario))
	if len(eco) > 400 {
		eco = eco[:400]
	}
	resp, _ := json.Marshal(map[string]string{"eco": string(eco)})
	return inference.Resposta{Conteudo: resp, TokensEntrada: inference.EstimarTokens(string(corpo)),
		TokensSaida: inference.EstimarTokens(string(resp)), Duracao: time.Millisecond}, nil
}
