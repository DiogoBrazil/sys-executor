package sei

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// URLLoginPadrao é a entrada do SEI/RO (mapeada na F0-08).
const URLLoginPadrao = "https://sei.sistemas.ro.gov.br/sip/login.php?sigla_orgao_sistema=RO&sigla_sistema=SEI"

// Fase informa à tarefa em que ponto da obtenção o cliente está.
type Fase int

const (
	FaseConectando Fase = iota
	FaseLogin
	FaseOTP
	FaseLocalizando
	FaseReautenticacao
	FaseGerando
	FaseBaixando
)

// Login são os dados digitados pelo usuário na tela do app (V4 §33).
type Login struct {
	Usuario  string
	Senha    *session.Segredo
	Orgao    string
	Processo string
}

// Interlocutor é como o cliente pede dados ao usuário e informa a fase.
// Os métodos bloqueiam até a resposta ou o cancelamento do contexto.
type Interlocutor interface {
	Fase(f Fase)
	Credenciais(ctx context.Context, alerta *erros.Erro) (Login, error)
	CodigoOTP(ctx context.Context, alerta *erros.Erro) (*session.Segredo, error)
	SenhaReautenticacao(ctx context.Context, alerta *erros.Erro) (*session.Segredo, error)
}

// Config do cliente.
type Config struct {
	Navegador     string // caminho do msedge.exe (ou Chrome, em testes)
	URLLogin      string
	PastaTrabalho string // pasta protegida da sessão (perfil e download)
	Visivel       bool   // janela visível (diagnóstico); padrão: sem janela
	PermitirHTTP  bool   // só para o simulador de testes
	AcoesAjax     []string
	// Tempos máximos (zero = padrão).
	TempoTela    time.Duration // espera por cada tela (padrão 90 s)
	TempoGeracao time.Duration // geração + download do PDF (padrão 20 min)
}

// LocalizarNavegador procura o Microsoft Edge instalado (ou SYSEXEC_NAVEGADOR).
func LocalizarNavegador() (string, error) {
	if p := os.Getenv("SYSEXEC_NAVEGADOR"); p != "" {
		return p, nil
	}
	for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
		if base == "" {
			continue
		}
		p := filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", erros.Com(erros.SEIIndisponivel, errors.New("Microsoft Edge não encontrado"))
}

// ObterPDF autentica, localiza o processo, gera o PDF consolidado com todos
// os documentos e devolve os bytes. Só usa ações de consulta.
func ObterPDF(ctx context.Context, cfg Config, in Interlocutor) ([]byte, error) {
	if cfg.URLLogin == "" {
		cfg.URLLogin = URLLoginPadrao
	}
	if cfg.TempoTela == 0 {
		cfg.TempoTela = 90 * time.Second
	}
	if cfg.TempoGeracao == 0 {
		cfg.TempoGeracao = 20 * time.Minute
	}
	u, err := url.Parse(cfg.URLLogin)
	if err != nil {
		return nil, erros.Com(erros.SEIIndisponivel, err)
	}
	acoesAjax := cfg.AcoesAjax
	if acoesAjax == nil {
		acoesAjax = AcoesAjax
	}
	guarda := NovaGuarda(u.Host, cfg.PermitirHTTP, AcoesLeitura, acoesAjax)
	perfil := filepath.Join(cfg.PastaTrabalho, "navegador")
	baixados := filepath.Join(cfg.PastaTrabalho, "baixados")
	for _, d := range []string{perfil, baixados} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, erros.Com(erros.SessaoCriacao, err)
		}
	}

	opts := []chromedp.ExecAllocatorOption{
		chromedp.ExecPath(cfg.Navegador),
		chromedp.UserDataDir(perfil),
		chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-crash-reporter", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-features", "msEdgeSignIn,EdgeCollections,msShopping,Translate,AutofillServerCommunication"),
		chromedp.Flag("password-store", "basic"),
		chromedp.WindowSize(1280, 900),
	}
	if !cfg.Visivel {
		opts = append(opts, chromedp.Flag("headless", "new"), chromedp.Flag("hide-scrollbars", true), chromedp.Flag("mute-audio", true))
	}
	actx, cancelAloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAloc()
	bctx, cancelNav := chromedp.NewContext(actx)
	defer cancelNav()

	s := &sessaoNavegador{ctx: bctx, guarda: guarda, baixados: baixados, downloads: make(chan string, 1)}
	in.Fase(FaseConectando)
	if err := s.preparar(); err != nil {
		return nil, erros.Com(erros.SEIIndisponivel, err)
	}
	if err := s.navegar(cfg.URLLogin); err != nil {
		return nil, err
	}
	if _, err := s.esperar(cfg.TempoTela, func(e estadoTela) bool { return e.Login }); err != nil {
		return nil, erros.Com(erros.SEIInterfaceAlterada, err)
	}

	// Login (com novas tentativas se o SEI recusar).
	var alerta *erros.Erro
	var processo string
	for tentativa := 0; ; tentativa++ {
		in.Fase(FaseLogin)
		l, err := in.Credenciais(ctx, alerta)
		if err != nil {
			return nil, err
		}
		processo = strings.TrimSpace(l.Processo)
		err = l.Senha.Usar(func(senha []byte) error {
			if err := s.preencher("txtUsuario", l.Usuario); err != nil {
				return err
			}
			if err := s.preencher("pwdSenha", string(senha)); err != nil {
				return err
			}
			if l.Orgao != "" {
				if err := s.preencher("selOrgao", l.Orgao); err != nil {
					return err
				}
			}
			return s.clicar("ACESSAR")
		})
		l.Senha.Zerar()
		if err != nil {
			return nil, erros.Com(erros.SEIInterfaceAlterada, err)
		}
		e, err := s.esperar(cfg.TempoTela, func(e estadoTela) bool {
			return e.OTP || e.Pesquisa || e.CredencialInvalida
		})
		if err != nil {
			return nil, erros.Com(erros.SEIInterfaceAlterada, err)
		}
		if e.CredencialInvalida && !e.OTP && !e.Pesquisa {
			alerta = erros.Novo(erros.SEICredencial)
			if tentativa >= 2 {
				return nil, alerta
			}
			continue
		}
		break
	}

	// Autenticação em dois fatores, se o SEI pedir.
	alerta = nil
	for tentativa := 0; ; tentativa++ {
		e, err := s.estado()
		if err != nil {
			return nil, erros.Com(erros.SEIInterfaceAlterada, err)
		}
		if !e.OTP {
			break
		}
		in.Fase(FaseOTP)
		cod, err := in.CodigoOTP(ctx, alerta)
		if err != nil {
			return nil, err
		}
		err = cod.Usar(func(c []byte) error {
			if err := s.preencher("txtCodigoAcesso", string(c)); err != nil {
				return err
			}
			return s.clicar("Validar") // nunca "Desativar 2FA" nem "Não usar 2FA neste dispositivo"
		})
		cod.Zerar()
		if err != nil {
			return nil, erros.Com(erros.SEIInterfaceAlterada, err)
		}
		e, err = s.esperar(cfg.TempoTela, func(e estadoTela) bool { return e.Pesquisa || e.CodigoInvalido })
		if err != nil {
			return nil, erros.Com(erros.SEIInterfaceAlterada, err)
		}
		if e.Pesquisa {
			break
		}
		alerta = erros.Novo(erros.SEIOTP)
		if tentativa >= 2 {
			return nil, alerta
		}
	}

	// Pesquisa rápida do processo.
	in.Fase(FaseLocalizando)
	if err := s.preencher("txtPesquisaRapida", processo); err != nil {
		return nil, erros.Com(erros.SEIInterfaceAlterada, err)
	}
	if err := s.avaliarBool(jsSubmeterPesquisa); err != nil {
		return nil, erros.Com(erros.SEIInterfaceAlterada, err)
	}
	e, err := s.esperar(cfg.TempoTela, func(e estadoTela) bool {
		return (e.ProcessoAberto && e.IconePDF) || e.Reautenticacao || e.NaoEncontrado || e.AcessoNegado
	})
	if err != nil {
		return nil, erros.Com(erros.SEIInterfaceAlterada, err)
	}

	// Processo sigiloso: o SEI pede a senha de novo (V4 §13: pedir ao usuário).
	alerta = nil
	for tentativa := 0; e.Reautenticacao && !e.ProcessoAberto; tentativa++ {
		in.Fase(FaseReautenticacao)
		senha, err := in.SenhaReautenticacao(ctx, alerta)
		if err != nil {
			return nil, err
		}
		err = senha.Usar(func(b []byte) error {
			if err := s.preencher("pwdSenha", string(b)); err != nil {
				return err
			}
			return s.clicar("Acessar")
		})
		senha.Zerar()
		if err != nil {
			return nil, erros.Com(erros.SEIInterfaceAlterada, err)
		}
		e, err = s.esperar(cfg.TempoTela, func(e estadoTela) bool {
			return (e.ProcessoAberto && e.IconePDF) || e.CredencialInvalida || e.AcessoNegado
		})
		if err != nil {
			return nil, erros.Com(erros.SEIInterfaceAlterada, err)
		}
		if e.ProcessoAberto && e.IconePDF {
			break
		}
		if e.AcessoNegado || tentativa >= 2 {
			return nil, erros.Novo(erros.SEIAcessoNegado)
		}
		alerta = erros.Novo(erros.SEICredencial)
	}
	switch {
	case e.NaoEncontrado:
		return nil, erros.Novo(erros.SEIProcessoNaoAchado)
	case e.AcessoNegado:
		return nil, erros.Novo(erros.SEIAcessoNegado)
	}

	// Gerar Arquivo PDF do Processo → Todos os documentos disponíveis → Gerar.
	in.Fase(FaseGerando)
	if err := s.clicar("Gerar Arquivo PDF do Processo"); err != nil {
		return nil, erros.Com(erros.SEIInterfaceAlterada, err)
	}
	if _, err := s.esperar(cfg.TempoTela, func(e estadoTela) bool { return e.FormGerar }); err != nil {
		return nil, erros.Com(erros.SEIInterfaceAlterada, err)
	}
	if err := s.avaliarBool(jsTodosDocumentos); err != nil {
		return nil, erros.Com(erros.SEIInterfaceAlterada, err)
	}
	if err := s.avaliarBool(jsGerar); err != nil {
		return nil, erros.Com(erros.SEIInterfaceAlterada, err)
	}
	return s.aguardarDownload(ctx, cfg.TempoGeracao, in)
}

// ---- navegador ----

type sessaoNavegador struct {
	ctx       context.Context
	guarda    *Guarda
	baixados  string
	downloads chan string // caminho do arquivo concluído ("" = cancelado)

	mu       sync.Mutex
	iniciado bool
	dialogo  string // texto do último diálogo JavaScript (alertas do SEI)
}

var reDigito = regexp.MustCompile(`[0-9]`)

// ultimoDialogo devolve e consome o texto do último diálogo exibido pela página.
func (s *sessaoNavegador) ultimoDialogo() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.dialogo
	s.dialogo = ""
	return d
}

// preparar inicia o navegador, liga a guarda, os diálogos e os downloads.
func (s *sessaoNavegador) preparar() error {
	if err := chromedp.Do(s.ctx); err != nil {
		return err
	}
	ligar := func(tctx context.Context) {
		chromedp.Listen(tctx, func(ev any) {
			switch e := ev.(type) {
			case *fetch.EventRequestPaused:
				go s.decidir(tctx, e)
			case *page.EventJavascriptDialogOpening:
				// Alertas do SEI são confirmados (só têm "OK"); confirmações
				// são recusadas: o cliente nunca aceita ação proposta pela página.
				// O texto é guardado para reconhecer a resposta (ex.: senha inválida).
				s.mu.Lock()
				s.dialogo = e.Message
				s.mu.Unlock()
				aceitar := e.Type == page.DialogTypeAlert
				go func() {
					_, _ = chromedp.Call(tctx, page.HandleJavaScriptDialog, page.HandleJavaScriptDialogParams{Accept: aceitar})
				}()
			}
		})
	}
	ligar(s.ctx)
	chromedp.ListenBrowser(s.ctx, func(ev any) {
		switch e := ev.(type) {
		case *target.EventTargetCreated:
			if e.TargetInfo.Type == "page" && e.TargetInfo.OpenerID != "" {
				// Janela aberta pela página: mesma guarda.
				go func() {
					tctx, _ := chromedp.NewContext(s.ctx, chromedp.WithTargetID(e.TargetInfo.TargetID))
					ligar(tctx)
					_, _ = chromedp.Call(tctx, fetch.Enable, fetch.EnableParams{})
					_, _ = chromedp.Call(tctx, page.Enable, page.EnableParams{})
				}()
			}
		case *browser.EventDownloadProgress:
			switch e.State {
			case browser.DownloadProgressStateCompleted:
				s.sinalizar(filepath.Join(s.baixados, e.GUID))
			case browser.DownloadProgressStateCanceled:
				s.sinalizar("")
			}
		case *browser.EventDownloadWillBegin:
			s.mu.Lock()
			s.iniciado = true
			s.mu.Unlock()
		}
	})
	ev := true
	if _, err := chromedp.CallBrowser(s.ctx, browser.SetDownloadBehavior, browser.SetDownloadBehaviorParams{
		Behavior: browser.SetDownloadBehaviorBehaviorAllowAndName, DownloadPath: s.baixados, EventsEnabled: &ev,
	}); err != nil {
		return err
	}
	if _, err := chromedp.Call(s.ctx, fetch.Enable, fetch.EnableParams{}); err != nil {
		return err
	}
	_, err := chromedp.Call(s.ctx, page.Enable, page.EnableParams{})
	return err
}

func (s *sessaoNavegador) sinalizar(p string) {
	select {
	case s.downloads <- p:
	default:
	}
}

func (s *sessaoNavegador) decidir(tctx context.Context, e *fetch.EventRequestPaused) {
	u, err := url.Parse(e.Request.URL)
	if err == nil && s.guarda.Permitir(u, e.ResourceType) {
		_, _ = chromedp.Call(tctx, fetch.ContinueRequest, fetch.ContinueRequestParams{RequestID: e.RequestID})
		return
	}
	_, _ = chromedp.Call(tctx, fetch.FailRequest, fetch.FailRequestParams{RequestID: e.RequestID, ErrorReason: network.ErrorReasonBlockedByClient})
}

// navegar abre o endereço; falha de rede vira REDE-001 (T03).
func (s *sessaoNavegador) navegar(endereco string) error {
	r, err := chromedp.Call(s.ctx, page.Navigate, page.NavigateParams{URL: endereco})
	if err != nil {
		return erros.Com(erros.SEIIndisponivel, err)
	}
	if r.ErrorText != "" {
		if semRede(r.ErrorText) {
			return erros.Com(erros.RedeSemConexao, errors.New(r.ErrorText))
		}
		return erros.Com(erros.SEIIndisponivel, errors.New(r.ErrorText))
	}
	return nil
}

func semRede(t string) bool {
	for _, c := range []string{"ERR_INTERNET_DISCONNECTED", "ERR_NAME_NOT_RESOLVED", "ERR_NAME_RESOLUTION_FAILED",
		"ERR_CONNECTION_REFUSED", "ERR_CONNECTION_TIMED_OUT", "ERR_ADDRESS_UNREACHABLE", "ERR_NETWORK_CHANGED", "ERR_PROXY_CONNECTION_FAILED"} {
		if strings.Contains(t, c) {
			return true
		}
	}
	return false
}

var (
	reDlgCredencial = regexp.MustCompile(`(usuario|senha).{0,40}(invalid|incorret)|(invalid|incorret).{0,40}(usuario|senha)`)
	reDlgCodigo     = regexp.MustCompile(`codigo.{0,40}(invalid|incorret|expirad)`)
)

func (s *sessaoNavegador) estado() (estadoTela, error) {
	var e estadoTela
	raw, err := chromedp.Run(s.ctx, chromedp.Evaluate[string](jsEstado))
	if err != nil {
		return e, err
	}
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return e, err
	}
	// Respostas do SEI exibidas em caixa de diálogo, e não na página.
	s.mu.Lock()
	d := texto.SemAcentos(s.dialogo)
	s.mu.Unlock()
	if d != "" {
		e.CredencialInvalida = e.CredencialInvalida || reDlgCredencial.MatchString(d)
		e.CodigoInvalido = e.CodigoInvalido || reDlgCodigo.MatchString(d)
	}
	return e, nil
}

// resumoTela descreve a tela atual para o diagnóstico (sem valores; dígitos mascarados).
func (s *sessaoNavegador) resumoTela() string {
	raw, err := chromedp.Run(s.ctx, chromedp.Evaluate[string](jsResumo))
	if err != nil {
		raw = "indisponível: " + err.Error()
	}
	s.mu.Lock()
	d := s.dialogo
	s.mu.Unlock()
	r := "tela atual: " + raw
	if d != "" {
		r += " | último aviso da página: " + d
	}
	if len(r) > 900 {
		r = r[:900] + "…"
	}
	return reDigito.ReplaceAllString(r, "#")
}

// esperar consulta a tela até a condição ou o tempo máximo.
func (s *sessaoNavegador) esperar(max time.Duration, cond func(estadoTela) bool) (estadoTela, error) {
	fim := time.Now().Add(max)
	for {
		e, err := s.estado()
		if err == nil && cond(e) {
			return e, nil
		}
		if s.ctx.Err() != nil {
			return e, s.ctx.Err()
		}
		if time.Now().After(fim) {
			return e, errors.New("tela esperada não apareceu; " + s.resumoTela())
		}
		select {
		case <-s.ctx.Done():
			return e, s.ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
}

func (s *sessaoNavegador) avaliarBool(js string) error {
	ok, err := chromedp.Run(s.ctx, chromedp.Evaluate[bool](js))
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("elemento não encontrado")
	}
	return nil
}

// preencher digita o valor no campo como um teclado (keydown/char/keyup):
// o campo de senha do SEI/RO ignora valores definidos por script ("Informe a
// Senha", visto no 1º teste real). Se a digitação não chegar ao campo, cai no
// preenchimento por script. Listas de seleção são sempre por script.
func (s *sessaoNavegador) preencher(nome, valor string) error {
	s.ultimoDialogo()
	if nome != "selOrgao" {
		n, _ := json.Marshal(nome)
		if s.avaliarBool(fmt.Sprintf(jsFocar, n)) == nil {
			if _, err := chromedp.Run(s.ctx, chromedp.KeyEvent(valor)); err == nil {
				tam, err := chromedp.Run(s.ctx, chromedp.Evaluate[int](fmt.Sprintf(jsTamanho, n)))
				if err == nil && tam == len([]rune(valor)) {
					return nil
				}
			}
		}
	}
	return s.preencherPorScript(nome, valor)
}

func (s *sessaoNavegador) preencherPorScript(nome, valor string) error {
	n, _ := json.Marshal(nome)
	v, _ := json.Marshal(valor)
	return s.avaliarBool(fmt.Sprintf(jsPreencher, n, v))
}

func (s *sessaoNavegador) clicar(rotulo string) error {
	s.ultimoDialogo()
	r, _ := json.Marshal(rotulo)
	return s.avaliarBool(fmt.Sprintf(jsClicar, r))
}

// aguardarDownload espera o PDF, lê, apaga o arquivo em claro e devolve os bytes.
func (s *sessaoNavegador) aguardarDownload(ctx context.Context, max time.Duration, in Interlocutor) ([]byte, error) {
	limite := time.After(max)
	tique := time.NewTicker(time.Second)
	defer tique.Stop()
	avisou := false
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-limite:
			return nil, erros.Novo(erros.SEIGeracaoFalhou)
		case <-tique.C:
			s.mu.Lock()
			ini := s.iniciado
			s.mu.Unlock()
			if ini && !avisou {
				in.Fase(FaseBaixando)
				avisou = true
			}
		case p := <-s.downloads:
			if !avisou {
				in.Fase(FaseBaixando)
			}
			if p == "" {
				return nil, erros.Novo(erros.SEIDownloadIncompleto)
			}
			b, err := os.ReadFile(p)
			_ = session.ApagarArquivo(p)
			if err != nil {
				return nil, erros.Com(erros.SEIDownloadIncompleto, err)
			}
			if !strings.HasPrefix(string(b[:min(len(b), 8)]), "%PDF-") {
				return nil, erros.Novo(erros.SEIDownloadIncompleto)
			}
			return b, nil
		}
	}
}
