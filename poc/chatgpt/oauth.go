package main

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
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	emissor        = "https://auth.openai.com"
	urlAutorizar   = "https://auth.openai.com/api/accounts/authorize"
	urlToken       = "https://auth.openai.com/api/accounts/oauth/token"
	urlRevogar     = "https://auth.openai.com/api/accounts/oauth/revoke"
	recurso        = "https://api.openai.com/v1"
	clienteDinamic = "dynamic_agent_client"
	nomeApp        = "sys-executor"
	escopoPlano    = "chatgpt.tokens.use.direct"
	escopos        = "openid profile email offline_access resource.invoke " + escopoPlano
	caminhoRetorno = "/auth/callback"
)

func aleatorio(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func desafioPKCE(verificador string) string {
	h := sha256.Sum256([]byte(verificador))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

type retorno struct {
	code, state, clientID, scope, erro, erroDesc string
}

// login executa o fluxo Authorization Code + PKCE com retorno em loopback.
// relogin=true usa o client_id salvo (e id_token_hint), sem agent_name_hint.
func login(ctx context.Context, c *Credencial, porta int, abrir bool, relogin bool, forcarConsentimento bool, soIdentidade bool) error {
	if err := c.garantirHostID(); err != nil {
		return err
	}
	if relogin && c.ClientID == "" {
		return errors.New("relogin exige um client_id salvo (faça 'login' primeiro)")
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", porta))
	if err != nil {
		return fmt.Errorf("servidor de retorno: %w", err)
	}
	defer ln.Close()
	end := ln.Addr().String() // 127.0.0.1:NNNN
	redirect := "http://" + end + caminhoRetorno

	state, nonce, verif := aleatorio(32), aleatorio(32), aleatorio(48)
	q := url.Values{}
	if relogin {
		q.Set("client_id", c.ClientID)
		if c.IDToken != "" {
			q.Set("id_token_hint", c.IDToken)
		}
		if c.Email != "" {
			q.Set("login_hint", c.Email)
		}
	} else {
		q.Set("client_id", clienteDinamic)
		q.Set("agent_name_hint", nomeApp)
	}
	q.Set("ext_agent_host_id", c.HostID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirect)
	if soIdentidade {
		q.Set("scope", "openid profile email offline_access")
	} else {
		q.Set("scope", escopos)
		q.Set("resource", recurso)
	}
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge_method", "S256")
	q.Set("code_challenge", desafioPKCE(verif))
	if forcarConsentimento {
		q.Set("prompt", "consent")
	}
	urlAuth := urlAutorizar + "?" + q.Encode()

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
			ch <- retorno{code: v.Get("code"), state: v.Get("state"), clientID: v.Get("client_id"),
				scope: v.Get("scope"), erro: v.Get("error"), erroDesc: v.Get("error_description")}
		})
		if !recebido {
			http.Error(w, "retorno já processado", http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		fmt.Fprintf(w, "<!doctype html><meta charset=utf-8><title>%s</title><p>Retorno recebido pelo %s. Pode fechar esta janela.</p>", html.EscapeString(nomeApp), html.EscapeString(nomeApp))
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	defer srv.Close()

	fmt.Printf("Servidor de retorno: %s\n", redirect)
	fmt.Printf("Modo: %s\n", map[bool]string{true: "relogin (client_id salvo)", false: "1º registro (dynamic_agent_client)"}[relogin])
	// id_token_hint e login_hint nunca aparecem em saída (doc: "Redact id_token_hint values from logs").
	exibivel := urlAuth
	if relogin {
		qq := url.Values{}
		for k, v := range q {
			qq[k] = v
		}
		qq.Del("id_token_hint")
		qq.Del("login_hint")
		exibivel = urlAutorizar + "?" + qq.Encode() + "  (dicas de conta omitidas; a aba aberta automaticamente as contém)"
	}
	fmt.Printf("\nAbra no navegador para entrar com o ChatGPT:\n\n%s\n\n", exibivel)
	if abrir {
		abrirNavegador(urlAuth)
	}

	espera, cancelar := context.WithTimeout(ctx, 5*time.Minute)
	defer cancelar()
	var rt retorno
	var recebidoEm time.Time
	select {
	case rt = <-ch:
		recebidoEm = time.Now()
	case <-espera.Done():
		return errors.New("tempo esgotado aguardando o retorno (5 min) ou cancelado")
	}
	if rt.state != state {
		return errors.New("state divergente: retorno recusado (possível CSRF)")
	}
	if rt.erro != "" {
		return fmt.Errorf("autorização negada ou falhou: %s (%s)", rt.erro, rt.erroDesc)
	}
	fmt.Printf("Retorno: code=%v client_id=%q scope=%q\n", rt.code != "", rt.clientID, rt.scope)

	clientID := c.ClientID
	switch {
	case !relogin && rt.clientID == "":
		return errors.New("registro incompleto: o retorno não trouxe o client_id emitido")
	case !relogin:
		clientID = rt.clientID
	case rt.clientID != "" && rt.clientID != c.ClientID:
		return fmt.Errorf("client_id do retorno (%s) difere do salvo (%s): recusado", rt.clientID, c.ClientID)
	}
	if rt.code == "" {
		return errors.New("retorno sem code")
	}

	cli := novoCliente(30*time.Second, hostAuth)
	fmt.Printf("trocando o code (%.1fs após o retorno)…\n", time.Since(recebidoEm).Seconds())
	form := url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {rt.code},
		"code_verifier": {verif}, "redirect_uri": {redirect}, "resource": {recurso},
	}
	if soIdentidade {
		form.Del("resource")
	}
	tok, err := postarToken(ctx, cli, form)
	if err != nil {
		return fmt.Errorf("troca do code: %w", err)
	}
	idt, err := validarIDToken(ctx, cli, tok.IDToken, clientID, nonce)
	if err != nil {
		return err
	}
	var claims map[string]any
	_ = idt.Claims(&claims)
	email, _ := claims["email"].(string)
	if relogin && c.Sub != "" && idt.Subject != c.Sub {
		return errors.New("a conta autenticada é diferente da salva: credenciais NÃO substituídas (desconecte antes de trocar de conta)")
	}
	esc := strings.Fields(tok.Scope)
	if len(esc) == 0 {
		esc = strings.Fields(rt.scope)
	}
	novo := *c
	novo.ClientID, novo.Sub, novo.Email, novo.Escopos = clientID, idt.Subject, email, esc
	novo.IDToken, novo.AccessToken = tok.IDToken, tok.AccessToken
	if tok.RefreshToken != "" {
		novo.RefreshToken = tok.RefreshToken
	}
	novo.Expira = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	novo.CedoRenovar = tok.cedo()
	if err := novo.gravar(); err != nil {
		return err
	}
	*c = novo

	fmt.Println("\n== Login concluído ==")
	fmt.Printf("client_id emitido: %s (prefixo oaiapp_: %v)\n", clientID, strings.HasPrefix(clientID, "oaiapp_"))
	fmt.Printf("conta: sub=%s… email=%s verificado=%v\n", cortar(idt.Subject, 8), mascararEmail(email), claims["email_verified"])
	fmt.Printf("escopos concedidos: %v\n", esc)
	fmt.Printf("permissão do plano (%s): %v\n", escopoPlano, c.temEscopo(escopoPlano))
	fmt.Printf("refresh_token recebido: %v · expires_in=%ds · earliest_refresh_at=%v\n", tok.RefreshToken != "", tok.ExpiresIn, tok.EarliestRefresh)
	fmt.Printf("id_token: alg=%s claims=%v\n", algJWT(tok.IDToken), chaves(claims))
	if aut, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		fmt.Printf("id_token[https://api.openai.com/auth]: %v\n", chaves(aut))
	}
	descreverAccessToken(tok.AccessToken)
	if !c.temEscopo(escopoPlano) {
		fmt.Println("\nATENÇÃO: permissão de uso do plano NÃO concedida → provedor desabilitado para gerar texto.")
	}
	return nil
}

type respostaToken struct {
	AccessToken     string          `json:"access_token"`
	RefreshToken    string          `json:"refresh_token"`
	IDToken         string          `json:"id_token"`
	TokenType       string          `json:"token_type"`
	ExpiresIn       int             `json:"expires_in"`
	Scope           string          `json:"scope"`
	EarliestRefresh json.RawMessage `json:"earliest_refresh_at"`
}

func (t respostaToken) cedo() time.Time {
	var n float64
	if json.Unmarshal(t.EarliestRefresh, &n) == nil && n > 0 {
		return time.Unix(int64(n), 0)
	}
	var s string
	if json.Unmarshal(t.EarliestRefresh, &s) == nil {
		if tm, err := time.Parse(time.RFC3339, s); err == nil {
			return tm
		}
	}
	return time.Time{}
}

// ErroOAuth guarda só o código e a descrição devolvidos (nunca o corpo do pedido).
type ErroOAuth struct {
	Status    int
	Codigo    string
	Descricao string
	Bruto     string
}

func (e *ErroOAuth) Error() string {
	if e.Codigo != "" && e.Descricao != "" {
		return fmt.Sprintf("HTTP %d %s: %s", e.Status, e.Codigo, e.Descricao)
	}
	if e.Codigo != "" {
		return fmt.Sprintf("HTTP %d %s (corpo: %s)", e.Status, e.Codigo, e.Bruto)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Bruto)
}

func postarToken(ctx context.Context, cli *http.Client, form url.Values) (respostaToken, error) {
	var t respostaToken
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlToken, strings.NewReader(form.Encode()))
	if err != nil {
		return t, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return t, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return t, erroOAuth(resp.StatusCode, b)
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return t, fmt.Errorf("resposta de token ilegível: %w", err)
	}
	if t.AccessToken == "" {
		return t, errors.New("resposta de token sem access_token")
	}
	return t, nil
}

func erroOAuth(status int, b []byte) *ErroOAuth {
	e := &ErroOAuth{Status: status}
	var v struct {
		Error            any    `json:"error"`
		ErrorDescription string `json:"error_description"`
		Detail           any    `json:"detail"`
	}
	if json.Unmarshal(b, &v) == nil {
		switch x := v.Error.(type) {
		case string:
			e.Codigo, e.Descricao = x, v.ErrorDescription
		case map[string]any:
			e.Codigo, _ = x["code"].(string)
			e.Descricao, _ = x["message"].(string)
		}
		if e.Codigo == "" && v.Detail != nil {
			e.Descricao = fmt.Sprint(v.Detail)
		}
	}
	e.Bruto = cortar(string(b), 400)
	return e
}

func validarIDToken(ctx context.Context, cli *http.Client, raw, clientID, nonce string) (*oidc.IDToken, error) {
	if raw == "" {
		return nil, errors.New("resposta sem id_token")
	}
	cctx := oidc.ClientContext(ctx, cli)
	prov, err := oidc.NewProvider(cctx, emissor)
	if err != nil {
		return nil, fmt.Errorf("discovery OIDC: %w", err)
	}
	idt, err := prov.Verifier(&oidc.Config{ClientID: clientID}).Verify(cctx, raw)
	if err != nil {
		return nil, fmt.Errorf("id_token inválido: %w", err)
	}
	if nonce != "" && idt.Nonce != nonce {
		return nil, errors.New("nonce do id_token divergente: recusado")
	}
	if idt.Subject == "" {
		return nil, errors.New("id_token sem sub")
	}
	return idt, nil
}

var muRenovar sync.Mutex

// renovar troca o refresh token (com rotação) e grava ANTES de usar o novo.
func renovar(ctx context.Context, c *Credencial) error {
	muRenovar.Lock()
	defer muRenovar.Unlock()
	if c.RefreshToken == "" || c.ClientID == "" {
		return errors.New("sem refresh_token/client_id salvos: faça login")
	}
	cli := novoCliente(30*time.Second, hostAuth)
	antigo := c.RefreshToken
	tok, err := postarToken(ctx, cli, url.Values{
		"grant_type": {"refresh_token"}, "client_id": {c.ClientID},
		"refresh_token": {antigo}, "resource": {recurso},
	})
	if err != nil {
		var eo *ErroOAuth
		if errors.As(err, &eo) && terminalRefresh(eo.Codigo) {
			c.AccessToken, c.RefreshToken, c.Expira = "", "", time.Time{}
			_ = c.gravar() // mantém client_id e host_id
			return fmt.Errorf("refresh terminal (%s): tokens apagados; client_id mantido; refaça o login: %w", eo.Codigo, err)
		}
		return err
	}
	novo := *c
	novo.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		novo.RefreshToken = tok.RefreshToken
	}
	if tok.IDToken != "" {
		if _, err := validarIDToken(ctx, cli, tok.IDToken, c.ClientID, ""); err != nil {
			return fmt.Errorf("id_token do refresh: %w", err)
		}
		novo.IDToken = tok.IDToken
	}
	if s := strings.Fields(tok.Scope); len(s) > 0 {
		novo.Escopos = s
	}
	novo.Expira = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	novo.CedoRenovar = tok.cedo()
	if err := novo.gravar(); err != nil {
		return err
	}
	*c = novo
	fmt.Printf("refresh ok: rotação=%v expires_in=%ds escopos=%v earliest_refresh_at=%s\n",
		tok.RefreshToken != "" && tok.RefreshToken != antigo, tok.ExpiresIn, novo.Escopos, string(tok.EarliestRefresh))
	return nil
}

func terminalRefresh(cod string) bool {
	switch cod {
	case "invalid_grant", "invalid_refresh_token", "token_expired", "refresh_token_expired",
		"refresh_token_invalidated", "refresh_token_reused":
		return true
	}
	return false
}

// tokenValido devolve um access token válido, renovando se faltar < 5 min.
func tokenValido(ctx context.Context, c *Credencial) (string, error) {
	if c.AccessToken != "" && time.Until(c.Expira) > 5*time.Minute {
		return c.AccessToken, nil
	}
	if err := renovar(ctx, c); err != nil {
		return "", err
	}
	return c.AccessToken, nil
}

func revogar(ctx context.Context, c *Credencial) error {
	if c.RefreshToken != "" && c.ClientID != "" {
		cli := novoCliente(30*time.Second, hostAuth)
		form := url.Values{"token": {c.RefreshToken}, "token_type_hint": {"refresh_token"}, "client_id": {c.ClientID}}
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, urlRevogar, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := cli.Do(req)
		if err != nil {
			fmt.Println("revogação não confirmada (rede):", err, "→ desconecte o app em Configurações do ChatGPT")
		} else {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
			resp.Body.Close()
			fmt.Printf("revogação: HTTP %d corpo=%q\n", resp.StatusCode, cortar(string(b), 200))
			// confirmação: o refresh revogado não pode mais renovar
			fmt.Println("refresh revogado reapresentado:", renovarSemGravar(ctx, c))
			if c.AccessToken != "" {
				_, _, err := listarModelos(ctx, c.AccessToken)
				fmt.Println("access token após a revogação (GET /v1/models):", map[bool]string{true: "AINDA ACEITO até expirar", false: fmt.Sprint("recusado: ", err)}[err == nil])
			}
		}
	}
	return removerCredencial()
}

func abrirNavegador(u string) {
	var cmd *exec.Cmd
	switch {
	case runtime.GOOS == "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case os.Getenv("WSL_DISTRO_NAME") != "":
		cmd = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", u) // WSL: explorer.exe perde a URL
		cmd.Dir = "/mnt/c/"
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("(não foi possível abrir o navegador automaticamente; copie a URL acima)")
		return
	}
	go cmd.Wait()
}

// descreverAccessToken mostra só nomes de claims e valores não secretos.
func descreverAccessToken(t string) {
	p := strings.Split(t, ".")
	if len(p) != 3 {
		fmt.Println("access_token: opaco (não é JWT)")
		return
	}
	b, err := base64.RawURLEncoding.DecodeString(p[1])
	if err != nil {
		return
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return
	}
	fmt.Printf("access_token: JWT alg=%s claims=%v aud=%v client_id=%v scope=%v\n", algJWT(t), chaves(m), m["aud"], m["client_id"], m["scope"])
}

func algJWT(t string) string {
	p := strings.Split(t, ".")
	if len(p) < 2 {
		return "?"
	}
	b, err := base64.RawURLEncoding.DecodeString(p[0])
	if err != nil {
		return "?"
	}
	var h struct {
		Alg string `json:"alg"`
	}
	_ = json.Unmarshal(b, &h)
	return h.Alg
}

func chaves(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func cortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// renovarSemGravar só apresenta o refresh token e relata (teste de reuso).
func renovarSemGravar(ctx context.Context, c *Credencial) error {
	cli := novoCliente(30*time.Second, hostAuth)
	_, err := postarToken(ctx, cli, url.Values{
		"grant_type": {"refresh_token"}, "client_id": {c.ClientID},
		"refresh_token": {c.RefreshToken}, "resource": {recurso},
	})
	if err == nil {
		return errors.New("ACEITO (o refresh antigo ainda vale: sem detecção de reuso)")
	}
	return err
}
