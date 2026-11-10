// Pacote chatgpt conecta a conta ChatGPT do usuário pelo fluxo oficial
// Sign in with ChatGPT (V4 §48.3, TRV-SEG-014), para usar o plano do
// usuário em requisições elegíveis à Responses API.
//
//   - OAuth 2.0 Authorization Code com PKCE S256, state e nonce; retorno só
//     em 127.0.0.1 (porta aleatória), um único GET conferido pelo Host;
//   - registro dinâmico (dynamic_agent_client → client_id emitido),
//     identificado como "sys-executor";
//   - ID token validado (assinatura JWKS, emissor, destinatário, validade,
//     nonce); a permissão de uso do plano (chatgpt.tokens.use.direct) é
//     exigida para gerar texto;
//   - credencial (registro, identidade, refresh token, ID token) cifrada com
//     DPAPI; o access token fica só em memória; renovação em série, com o
//     refresh novo gravado antes de ser usado;
//   - nenhum token vai à interface, a URLs de páginas, a erros ou logs.
//
// Comportamentos confirmados na PoC de 09/11/2026 (docs/planning/
// RELATORIO_POC_CHATGPT.md): o relogin omite client_id no retorno; conta
// sem plano elegível recebe invalid_grant na troca do code; a revogação
// invalida na hora o refresh e o access token.
package chatgpt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/DiogoBrazil/sys-executor/internal/credencial"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
)

// Constantes do fluxo (documentação SIWC e discovery, 09/11/2026).
const (
	EmissorOpenAI     = "https://auth.openai.com"
	Recurso           = "https://api.openai.com/v1"
	ClienteDinamico   = "dynamic_agent_client"
	NomeAplicativo    = "sys-executor"
	EscopoPlano       = "chatgpt.tokens.use.direct"
	escopos           = "openid profile email offline_access resource.invoke " + EscopoPlano
	caminhoRetorno    = "/auth/callback"
	prazoLogin        = 5 * time.Minute
	margemRenovacao   = 2 * time.Minute
	versaoRegistro    = 1
	ArquivoCredencial = "chatgpt.dpapi"
)

// EntropiaCredencial vincula a credencial cifrada a este uso (DPAPI).
var EntropiaCredencial = []byte("sys-executor/chatgpt-siwc/v1")

// registro é o que fica cifrado no disco (nunca o access token).
type registro struct {
	Versao       int       `json:"versao"`
	HostID       string    `json:"host_id"`
	ClientID     string    `json:"client_id"`
	Sub          string    `json:"sub"`
	Email        string    `json:"email"`
	Escopos      []string  `json:"escopos"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	Salvo        time.Time `json:"salvo"`
}

func (r *registro) temEscopo(e string) bool {
	for _, s := range r.Escopos {
		if s == e {
			return true
		}
	}
	return false
}

// Config da conta.
type Config struct {
	Loja     credencial.Loja
	AbrirURL func(string) error // navegador padrão do sistema (nunca a WebView)
	// Somente testes: emissor falso e relógio.
	Emissor string
	agora   func() time.Time
}

// Conta é a conexão com o ChatGPT deste computador (uma por vez).
type Conta struct {
	cfg Config

	mu        sync.Mutex
	carregado bool
	reg       *registro
	acesso    string
	expira    time.Time
	renovadoA time.Time
	entrando  bool

	muRenovar sync.Mutex

	muProv sync.Mutex
	prov   *oidc.Provider // discovery e JWKS em cache (chaves renovadas pelo go-oidc ao surgir kid novo)
}

// provedor devolve o provedor OIDC (discovery uma vez por execução).
func (c *Conta) provedor(ctx context.Context, cli *http.Client) (*oidc.Provider, error) {
	c.muProv.Lock()
	defer c.muProv.Unlock()
	if c.prov != nil {
		return c.prov, nil
	}
	// O cliente restrito fica associado ao provedor para buscar o JWKS depois.
	p, err := oidc.NewProvider(oidc.ClientContext(context.WithoutCancel(ctx), cli), c.emissor())
	if err != nil {
		return nil, err
	}
	c.prov = p
	return p, nil
}

// Nova cria a conta (a credencial é lida na primeira consulta).
func Nova(cfg Config) *Conta { return &Conta{cfg: cfg} }

func (c *Conta) emissor() string {
	if c.cfg.Emissor != "" {
		return c.cfg.Emissor
	}
	return EmissorOpenAI
}

func (c *Conta) agora() time.Time {
	if c.cfg.agora != nil {
		return c.cfg.agora()
	}
	return time.Now()
}

// Estado da conta para a interface (sem nenhum token).
type Estado struct {
	Conectada       bool   `json:"conectada"`
	Email           string `json:"email"`
	PlanoAutorizado bool   `json:"planoAutorizado"`
	Reconectar      bool   `json:"reconectar"` // registro existe, mas a sessão expirou ou foi revogada
	Entrando        bool   `json:"entrando"`
}

// Estado lê a credencial (uma vez) e resume.
func (c *Conta) Estado() (Estado, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.carregarLocked(); err != nil {
		return Estado{Entrando: c.entrando}, err
	}
	e := Estado{Entrando: c.entrando}
	if c.reg != nil {
		e.Email = c.reg.Email
		e.Conectada = c.reg.RefreshToken != ""
		e.Reconectar = c.reg.RefreshToken == ""
		e.PlanoAutorizado = e.Conectada && c.reg.temEscopo(EscopoPlano)
	}
	return e, nil
}

func (c *Conta) carregarLocked() error {
	if c.carregado {
		return nil
	}
	if c.cfg.Loja == nil || !c.cfg.Loja.Existe() {
		c.carregado = true
		return nil
	}
	b, err := c.cfg.Loja.Ler()
	if errors.Is(err, credencial.ErrAusente) {
		c.carregado = true
		return nil
	}
	if err != nil {
		return erros.Com(erros.IACredencial, err)
	}
	defer credencial.Zerar(b)
	var r registro
	if err := json.Unmarshal(b, &r); err != nil || r.Versao != versaoRegistro {
		// Ilegível: tratado como ausente (o usuário conecta de novo).
		c.carregado = true
		return nil
	}
	c.reg, c.carregado = &r, true
	return nil
}

func (c *Conta) gravarLocked(r *registro) error {
	r.Versao, r.Salvo = versaoRegistro, c.agora().UTC()
	b, err := json.Marshal(r)
	if err != nil {
		return erros.Com(erros.InternoInesperado, err)
	}
	defer credencial.Zerar(b)
	if err := c.cfg.Loja.Guardar(b); err != nil {
		return erros.Com(erros.IACredencial, err)
	}
	c.reg = r
	return nil
}

// ---- tokens (inference.FonteToken) ----

var _ inference.FonteToken = (*Conta)(nil)

// Token devolve o access token válido, renovando perto do vencimento.
func (c *Conta) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	if err := c.carregarLocked(); err != nil {
		c.mu.Unlock()
		return "", err
	}
	if c.reg == nil {
		c.mu.Unlock()
		return "", erros.Novo(erros.IAContaDesconectada)
	}
	if c.acesso != "" && c.agora().Add(margemRenovacao).Before(c.expira) {
		t, plano := c.acesso, c.reg.temEscopo(EscopoPlano)
		c.mu.Unlock()
		if !plano {
			return "", erros.Novo(erros.IAPlanoNaoAutorizado)
		}
		return t, nil
	}
	c.mu.Unlock()
	return c.Renovar(ctx)
}

// Renovar troca o refresh token (com rotação). Renovações são feitas em
// série; uma renovação recente é reaproveitada por quem chega depois.
func (c *Conta) Renovar(ctx context.Context) (string, error) {
	c.muRenovar.Lock()
	defer c.muRenovar.Unlock()
	c.mu.Lock()
	if err := c.carregarLocked(); err != nil {
		c.mu.Unlock()
		return "", err
	}
	if c.reg == nil {
		c.mu.Unlock()
		return "", erros.Novo(erros.IAContaDesconectada)
	}
	if c.acesso != "" && c.agora().Sub(c.renovadoA) < 10*time.Second && c.agora().Before(c.expira) {
		t, plano := c.acesso, c.reg.temEscopo(EscopoPlano)
		c.mu.Unlock()
		if !plano {
			return "", erros.Novo(erros.IAPlanoNaoAutorizado)
		}
		return t, nil
	}
	if c.reg.RefreshToken == "" || c.reg.ClientID == "" {
		c.mu.Unlock()
		return "", erros.Novo(erros.IASessaoExpirada)
	}
	reg := *c.reg
	c.mu.Unlock()

	cli, err := inference.NovoClienteRestrito(c.emissor(), nil)
	if err != nil {
		return "", erros.Com(erros.InternoInesperado, err)
	}
	cli.Timeout = 30 * time.Second
	tok, err := c.postarToken(ctx, cli, url.Values{
		"grant_type": {"refresh_token"}, "client_id": {reg.ClientID},
		"refresh_token": {reg.RefreshToken}, "resource": {Recurso},
	})
	if err != nil {
		var eo *erroOAuth
		if errors.As(err, &eo) && refreshTerminal(eo.Codigo) {
			// Sessão encerrada: apaga só os tokens; registro e identidade
			// ficam para a reconexão com o mesmo client_id.
			c.mu.Lock()
			reg.RefreshToken, reg.IDToken = "", ""
			c.acesso, c.expira = "", time.Time{}
			gerr := c.gravarLocked(&reg)
			c.mu.Unlock()
			if gerr != nil {
				return "", gerr
			}
			return "", erros.Com(erros.IASessaoExpirada, err)
		}
		if errors.As(err, &eo) {
			return "", erros.Com(erros.IAIndisponivel, err) // 4xx não terminal: credencial preservada
		}
		return "", err // rede/5xx: credencial preservada
	}
	if tok.IDToken != "" {
		if _, err := c.validarIDToken(ctx, cli, tok.IDToken, reg.ClientID, ""); err != nil {
			return "", erros.Com(erros.IASessaoExpirada, err)
		}
		reg.IDToken = tok.IDToken
	}
	if tok.RefreshToken != "" {
		reg.RefreshToken = tok.RefreshToken
	}
	if s := strings.Fields(tok.Scope); len(s) > 0 {
		reg.Escopos = s
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// O refresh novo é gravado antes de o access token ser usado.
	if err := c.gravarLocked(&reg); err != nil {
		return "", err
	}
	c.acesso = tok.AccessToken
	c.expira = c.agora().Add(time.Duration(tok.ExpiresIn) * time.Second)
	c.renovadoA = c.agora()
	if !reg.temEscopo(EscopoPlano) {
		return "", erros.Novo(erros.IAPlanoNaoAutorizado)
	}
	return c.acesso, nil
}

func refreshTerminal(cod string) bool {
	switch cod {
	case "invalid_grant", "invalid_refresh_token", "token_expired", "refresh_token_expired",
		"refresh_token_invalidated", "refresh_token_reused", "invalid_client":
		return true
	}
	return false
}

// ---- login ----

// Entrar executa o login no navegador do sistema e só volta ao concluir,
// falhar, vencer o prazo (5 min) ou ser cancelado (ctx).
func (c *Conta) Entrar(ctx context.Context) error {
	c.mu.Lock()
	if c.entrando {
		c.mu.Unlock()
		return erros.Com(erros.IALoginNaoConcluido, errors.New("já há uma entrada em andamento"))
	}
	if err := c.carregarLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	c.entrando = true
	var anterior *registro
	if c.reg != nil {
		r := *c.reg
		anterior = &r
	}
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.entrando = false
		c.mu.Unlock()
	}()
	return c.entrar(ctx, anterior)
}

func aleatorio(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func desafioPKCE(v string) string {
	h := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func novoHostID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type retorno struct{ code, state, clientID, scope, erro string }

func (c *Conta) entrar(ctx context.Context, anterior *registro) error {
	cli, err := inference.NovoClienteRestrito(c.emissor(), nil)
	if err != nil {
		return erros.Com(erros.InternoInesperado, err)
	}
	cli.Timeout = 30 * time.Second
	prov, err := c.provedor(ctx, cli)
	if err != nil {
		return erros.Com(erros.IASemConexao, fmt.Errorf("discovery: %w", err))
	}
	endp := prov.Endpoint()
	if !mesmaOrigem(endp.AuthURL, c.emissor()) || !mesmaOrigem(endp.TokenURL, c.emissor()) {
		return erros.Com(erros.InternoInesperado, errors.New("endpoints fora do emissor"))
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return erros.Com(erros.InternoInesperado, fmt.Errorf("servidor de retorno: %w", err))
	}
	defer ln.Close()
	end := ln.Addr().String()
	redirect := "http://" + end + caminhoRetorno

	relogin := anterior != nil && anterior.ClientID != ""
	hostID := novoHostID()
	if anterior != nil && anterior.HostID != "" {
		hostID = anterior.HostID
	}
	state, nonce, verif := aleatorio(32), aleatorio(32), aleatorio(48)
	q := url.Values{}
	if relogin {
		q.Set("client_id", anterior.ClientID)
		if anterior.IDToken != "" {
			q.Set("id_token_hint", anterior.IDToken)
		}
		if anterior.Email != "" {
			q.Set("login_hint", anterior.Email)
		}
	} else {
		q.Set("client_id", ClienteDinamico)
		q.Set("agent_name_hint", NomeAplicativo)
	}
	q.Set("ext_agent_host_id", hostID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirect)
	q.Set("scope", escopos)
	q.Set("resource", Recurso)
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge_method", "S256")
	q.Set("code_challenge", desafioPKCE(verif))

	ch := make(chan retorno, 1)
	var uma sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc(caminhoRetorno, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Host != end {
			http.Error(w, "requisição recusada", http.StatusBadRequest)
			return
		}
		v := r.URL.Query()
		recebido := false
		uma.Do(func() {
			recebido = true
			ch <- retorno{code: v.Get("code"), state: v.Get("state"), clientID: v.Get("client_id"), scope: v.Get("scope"), erro: v.Get("error")}
		})
		if !recebido {
			http.Error(w, "retorno já processado", http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		fmt.Fprintf(w, "<!doctype html><meta charset=utf-8><title>%s</title><p>Retorno recebido pelo %s. Pode fechar esta janela e voltar ao aplicativo.</p>",
			html.EscapeString(NomeAplicativo), html.EscapeString(NomeAplicativo))
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	defer srv.Close()

	if c.cfg.AbrirURL == nil {
		return erros.Com(erros.InternoInesperado, errors.New("sem navegador"))
	}
	// A URL (com id_token_hint e login_hint) só vai ao navegador do sistema;
	// nunca à interface, a erros ou a logs.
	if err := c.cfg.AbrirURL(endp.AuthURL + "?" + q.Encode()); err != nil {
		return erros.Com(erros.InternoInesperado, fmt.Errorf("abrir o navegador: %w", err))
	}

	espera, cancelar := context.WithTimeout(ctx, prazoLogin)
	defer cancelar()
	var rt retorno
	select {
	case rt = <-ch:
	case <-espera.Done():
		return erros.Com(erros.IALoginNaoConcluido, errors.New("retorno não recebido (prazo ou cancelamento)"))
	}
	if rt.state != state {
		return erros.Com(erros.IALoginNaoConcluido, errors.New("state divergente: retorno recusado"))
	}
	if rt.erro != "" {
		return erros.Com(erros.IALoginNaoConcluido, fmt.Errorf("autorização: %s", rt.erro))
	}
	clientID := ""
	switch {
	case relogin && (rt.clientID == "" || rt.clientID == anterior.ClientID):
		clientID = anterior.ClientID
	case relogin:
		return erros.Com(erros.IALoginNaoConcluido, errors.New("client_id do retorno difere do registrado"))
	case rt.clientID == "":
		return erros.Com(erros.IALoginNaoConcluido, errors.New("registro incompleto: retorno sem client_id"))
	default:
		clientID = rt.clientID
	}
	if rt.code == "" {
		return erros.Com(erros.IALoginNaoConcluido, errors.New("retorno sem code"))
	}

	tok, err := c.postarToken(ctx, cli, url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {rt.code},
		"code_verifier": {verif}, "redirect_uri": {redirect}, "resource": {Recurso},
	})
	if err != nil {
		var eo *erroOAuth
		if errors.As(err, &eo) && eo.Codigo == "invalid_grant" {
			// Visto na PoC: conta sem plano elegível recebe invalid_grant aqui.
			return erros.Com(erros.IALoginNaoConcluido, err)
		}
		if errors.As(err, &eo) {
			return erros.Com(erros.IALoginNaoConcluido, err)
		}
		return err
	}
	idt, err := c.validarIDToken(ctx, cli, tok.IDToken, clientID, nonce)
	if err != nil {
		return erros.Com(erros.IALoginNaoConcluido, err)
	}
	var claims struct {
		Email string `json:"email"`
	}
	_ = idt.Claims(&claims)
	if relogin && anterior.Sub != "" && idt.Subject != anterior.Sub {
		return erros.Com(erros.IALoginNaoConcluido, errors.New("conta diferente da conectada: desconecte antes de trocar de conta"))
	}
	esc := strings.Fields(tok.Scope)
	if len(esc) == 0 {
		esc = strings.Fields(rt.scope)
	}
	novo := &registro{HostID: hostID, ClientID: clientID, Sub: idt.Subject, Email: claims.Email,
		Escopos: esc, RefreshToken: tok.RefreshToken, IDToken: tok.IDToken}
	if novo.RefreshToken == "" {
		return erros.Com(erros.IALoginNaoConcluido, errors.New("sem refresh token (offline_access não concedido)"))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.gravarLocked(novo); err != nil {
		return err
	}
	c.acesso = tok.AccessToken
	c.expira = c.agora().Add(time.Duration(tok.ExpiresIn) * time.Second)
	c.renovadoA = c.agora()
	if !novo.temEscopo(EscopoPlano) {
		// A conta fica conectada, mas o provedor não gera texto (doc SIWC).
		return erros.Novo(erros.IAPlanoNaoAutorizado)
	}
	return nil
}

func mesmaOrigem(u, emissor string) bool {
	a, err1 := url.Parse(u)
	b, err2 := url.Parse(emissor)
	return err1 == nil && err2 == nil && a.Scheme == b.Scheme && strings.EqualFold(a.Host, b.Host)
}

func (c *Conta) validarIDToken(ctx context.Context, cli *http.Client, raw, clientID, nonce string) (*oidc.IDToken, error) {
	if raw == "" {
		return nil, errors.New("resposta sem id_token")
	}
	cctx := oidc.ClientContext(ctx, cli)
	prov, err := c.provedor(ctx, cli)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	idt, err := prov.Verifier(&oidc.Config{ClientID: clientID, Now: c.agora}).Verify(cctx, raw)
	if err != nil {
		return nil, fmt.Errorf("id_token inválido: %w", err)
	}
	if nonce != "" && idt.Nonce != nonce {
		return nil, errors.New("nonce do id_token divergente")
	}
	if idt.Subject == "" {
		return nil, errors.New("id_token sem sub")
	}
	return idt, nil
}

// ---- desconectar ----

// Desconectar revoga a sessão no provedor (quando possível) e apaga a
// credencial local. Devolve confirmada=false se a revogação não foi
// confirmada (o usuário deve desconectar também nas configurações do
// ChatGPT). Falha de rede não impede apagar a credencial local.
func (c *Conta) Desconectar(ctx context.Context) (confirmada bool, err error) {
	c.muRenovar.Lock()
	defer c.muRenovar.Unlock()
	c.mu.Lock()
	if err := c.carregarLocked(); err != nil {
		c.mu.Unlock()
		return false, err
	}
	var reg *registro
	if c.reg != nil {
		r := *c.reg
		reg = &r
	}
	c.mu.Unlock()
	if reg != nil && reg.RefreshToken != "" && reg.ClientID != "" {
		confirmada = c.revogar(ctx, reg)
	} else {
		confirmada = true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reg, c.acesso, c.expira = nil, "", time.Time{}
	if c.cfg.Loja != nil {
		if e := c.cfg.Loja.Remover(); e != nil {
			return confirmada, erros.Com(erros.IACredencial, e)
		}
	}
	return confirmada, nil
}

func (c *Conta) revogar(ctx context.Context, reg *registro) bool {
	cli, err := inference.NovoClienteRestrito(c.emissor(), nil)
	if err != nil {
		return false
	}
	cli.Timeout = 20 * time.Second
	prov, err := c.provedor(ctx, cli)
	if err != nil {
		return false
	}
	var meta struct {
		Revogacao string `json:"revocation_endpoint"`
	}
	if prov.Claims(&meta) != nil || meta.Revogacao == "" || !mesmaOrigem(meta.Revogacao, c.emissor()) {
		return false
	}
	form := url.Values{"token": {reg.RefreshToken}, "token_type_hint": {"refresh_token"}, "client_id": {reg.ClientID}}
	for tentativa := 0; tentativa < 3; tentativa++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, meta.Revogacao, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := cli.Do(req)
		if err == nil {
			io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
			if resp.StatusCode < 500 {
				return false
			}
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Duration(tentativa+1) * time.Second):
		}
	}
	return false
}

// ---- troca de tokens ----

type respostaToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// erroOAuth guarda só o código e a descrição devolvidos pelo emissor.
type erroOAuth struct {
	Status    int
	Codigo    string
	Descricao string
}

func (e *erroOAuth) Error() string {
	return fmt.Sprintf("oauth %d %s: %s", e.Status, e.Codigo, e.Descricao)
}

func (c *Conta) postarToken(ctx context.Context, cli *http.Client, form url.Values) (respostaToken, error) {
	var t respostaToken
	prov, err := c.provedor(ctx, cli)
	if err != nil {
		return t, erros.Com(erros.IASemConexao, fmt.Errorf("discovery: %w", err))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, prov.Endpoint().TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return t, erros.Com(erros.InternoInesperado, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return t, context.Canceled
		}
		return t, erros.Com(erros.IASemConexao, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		eo := &erroOAuth{Status: resp.StatusCode}
		var v struct {
			Error            any    `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if json.Unmarshal(b, &v) == nil {
			switch x := v.Error.(type) {
			case string:
				eo.Codigo, eo.Descricao = x, v.ErrorDescription
			case map[string]any:
				eo.Codigo, _ = x["code"].(string)
				eo.Descricao, _ = x["message"].(string)
			}
		}
		if resp.StatusCode >= 500 {
			return t, erros.Com(erros.IAIndisponivel, eo)
		}
		return t, eo
	}
	if err := json.Unmarshal(b, &t); err != nil || t.AccessToken == "" {
		return t, erros.Com(erros.IAIndisponivel, errors.New("resposta de token ilegível"))
	}
	return t, nil
}
