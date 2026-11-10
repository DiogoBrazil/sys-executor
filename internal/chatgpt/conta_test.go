package chatgpt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"

	"github.com/DiogoBrazil/sys-executor/internal/credencial"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// emissorFalso imita auth.openai.com (discovery, JWKS, token, revogação) e,
// pelo AbrirURL, o navegador do usuário.
type emissorFalso struct {
	t      *testing.T
	srv    *httptest.Server
	chave  *rsa.PrivateKey
	mu     sync.Mutex
	authz  []url.Values // parâmetros de cada autorização aberta
	codes  map[string]url.Values
	posts  []url.Values // corpos do endpoint de token
	revog  []url.Values
	refres string // refresh token válido no momento
	gen    int

	// comportamento do "navegador"/emissor
	sub          string
	semPlano     bool
	negar        bool
	stateErrado  bool
	nonceErrado  bool
	invalidGrant bool
	semClientID  bool
	refreshMorto bool
	duplicar     bool
}

func novoEmissor(t *testing.T) *emissorFalso {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	e := &emissorFalso{t: t, chave: k, codes: map[string]url.Values{}, sub: "user-sub-1"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		u := e.srv.URL
		json.NewEncoder(w).Encode(map[string]any{
			"issuer": u, "authorization_endpoint": u + "/api/accounts/authorize", "token_endpoint": u + "/api/accounts/oauth/token",
			"revocation_endpoint": u + "/api/accounts/oauth/revoke", "jwks_uri": u + "/.well-known/jwks.json",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &k.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig"}}})
	})
	mux.HandleFunc("/api/accounts/oauth/token", e.token)
	mux.HandleFunc("/api/accounts/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		e.mu.Lock()
		e.revog = append(e.revog, r.PostForm)
		e.refres = ""
		e.mu.Unlock()
	})
	e.srv = httptest.NewServer(mux)
	t.Cleanup(e.srv.Close)
	return e
}

func (e *emissorFalso) idToken(aud, nonce string) string {
	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: e.chave, KeyID: "k1"}}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		e.t.Fatal(err)
	}
	agora := time.Now().Unix()
	claims := map[string]any{"iss": e.srv.URL, "aud": []string{aud}, "sub": e.sub, "email": "fulano@exemplo.invalid",
		"iat": agora, "exp": agora + 3600}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	b, _ := json.Marshal(claims)
	j, err := sig.Sign(b)
	if err != nil {
		e.t.Fatal(err)
	}
	s, _ := j.CompactSerialize()
	return s
}

func (e *emissorFalso) escopos() string {
	if e.semPlano {
		return "openid profile email offline_access resource.invoke"
	}
	return "openid profile email offline_access resource.invoke chatgpt.tokens.use.direct"
}

func (e *emissorFalso) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	f := r.PostForm
	e.mu.Lock()
	defer e.mu.Unlock()
	e.posts = append(e.posts, f)
	falhar := func(cod string) {
		w.WriteHeader(400)
		fmt.Fprintf(w, `{"error":%q}`, cod)
	}
	if f.Get("resource") != Recurso {
		falhar("invalid_target")
		return
	}
	switch f.Get("grant_type") {
	case "authorization_code":
		a, ok := e.codes[f.Get("code")]
		delete(e.codes, f.Get("code"))
		h := sha256.Sum256([]byte(f.Get("code_verifier")))
		if !ok || e.invalidGrant || base64.RawURLEncoding.EncodeToString(h[:]) != a.Get("code_challenge") || f.Get("redirect_uri") != a.Get("redirect_uri") {
			falhar("invalid_grant")
			return
		}
		nonce := a.Get("nonce")
		if e.nonceErrado {
			nonce = "outro"
		}
		e.gen++
		e.refres = fmt.Sprintf("refresh-%d", e.gen)
		json.NewEncoder(w).Encode(map[string]any{"access_token": fmt.Sprintf("access-%d", e.gen), "refresh_token": e.refres,
			"id_token": e.idToken(f.Get("client_id"), nonce), "expires_in": 3600, "scope": e.escopos(), "token_type": "Bearer"})
	case "refresh_token":
		if e.refreshMorto || f.Get("refresh_token") != e.refres || f.Get("scope") != "" {
			falhar("invalid_grant")
			return
		}
		e.gen++
		e.refres = fmt.Sprintf("refresh-%d", e.gen)
		json.NewEncoder(w).Encode(map[string]any{"access_token": fmt.Sprintf("access-%d", e.gen), "refresh_token": e.refres,
			"id_token": e.idToken(f.Get("client_id"), ""), "expires_in": 3600, "scope": e.escopos()})
	default:
		falhar("unsupported_grant_type")
	}
}

// abrir faz o papel do navegador: recebe a URL de autorização e chama o
// retorno em 127.0.0.1 como o emissor faria.
func (e *emissorFalso) abrir(u string) error {
	pu, err := url.Parse(u)
	if err != nil {
		return err
	}
	q := pu.Query()
	e.mu.Lock()
	e.authz = append(e.authz, q)
	code := fmt.Sprintf("code-%d", len(e.authz))
	e.codes[code] = q
	e.mu.Unlock()
	cb := url.Values{"state": {q.Get("state")}}
	switch {
	case e.negar:
		cb.Set("error", "access_denied")
	default:
		cb.Set("code", code)
		cb.Set("scope", e.escopos())
		if q.Get("client_id") == ClienteDinamico && !e.semClientID {
			cb.Set("client_id", "oaiapp_teste")
		}
	}
	if e.stateErrado {
		cb.Set("state", "forjado")
	}
	go func() {
		alvo := q.Get("redirect_uri") + "?" + cb.Encode()
		resp, err := http.Get(alvo)
		if err == nil {
			resp.Body.Close()
		}
		if e.duplicar {
			if r2, err := http.Get(alvo); err == nil {
				e.mu.Lock()
				e.authz = append(e.authz, url.Values{"_segundo_status": {fmt.Sprint(r2.StatusCode)}})
				e.mu.Unlock()
				r2.Body.Close()
			}
		}
	}()
	return nil
}

func (e *emissorFalso) conta(loja credencial.Loja) *Conta {
	return Nova(Config{Loja: loja, AbrirURL: e.abrir, Emissor: e.srv.URL})
}

func codigoDe(err error) erros.Codigo {
	var x *erros.Erro
	if errors.As(err, &x) {
		return x.Codigo
	}
	return ""
}

// Login completo: registro dinâmico, PKCE, state, nonce, ID token pela JWKS,
// escopos; credencial sem access token; nenhum token na interface.
// Regra: TRV-SEG-014
// Regra: TRV-SEG-015
func TestEntrarConectaComPlano(t *testing.T) {
	e := novoEmissor(t)
	loja := &credencial.LojaMemoria{}
	c := e.conta(loja)
	if err := c.Entrar(context.Background()); err != nil {
		t.Fatal(err)
	}
	q := e.authz[0]
	for k, v := range map[string]string{"client_id": ClienteDinamico, "agent_name_hint": "sys-executor", "response_type": "code",
		"code_challenge_method": "S256", "resource": Recurso} {
		if q.Get(k) != v {
			t.Errorf("%s=%q, esperado %q", k, q.Get(k), v)
		}
	}
	if !strings.Contains(q.Get("scope"), EscopoPlano) || !strings.HasPrefix(q.Get("redirect_uri"), "http://127.0.0.1:") ||
		!strings.HasPrefix(q.Get("ext_agent_host_id"), "urn:uuid:") || q.Get("state") == "" || q.Get("nonce") == "" {
		t.Fatalf("autorização: %v", q)
	}
	st, _ := c.Estado()
	if !st.Conectada || !st.PlanoAutorizado || st.Email != "fulano@exemplo.invalid" || st.Entrando {
		t.Fatalf("estado: %+v", st)
	}
	b, _ := loja.Ler()
	if strings.Contains(string(b), "access-") || !strings.Contains(string(b), "oaiapp_teste") || !strings.Contains(string(b), "refresh-1") {
		t.Fatalf("credencial gravada: %s", b)
	}
	tok, err := c.Token(context.Background())
	if err != nil || tok != "access-1" || len(e.posts) != 1 {
		t.Fatalf("token: %q %v (posts %d)", tok, err, len(e.posts))
	}
	if js, _ := json.Marshal(st); strings.Contains(string(js), "access") || strings.Contains(string(js), "refresh") {
		t.Fatal("estado para a interface não pode ter tokens")
	}
}

// Relogin com o client_id salvo e id_token_hint, sem agent_name_hint; o
// retorno sem client_id é aceito; outra conta é recusada sem trocar a
// credencial.
func TestReloginMesmaContaEOutraConta(t *testing.T) {
	e := novoEmissor(t)
	loja := &credencial.LojaMemoria{}
	c := e.conta(loja)
	if err := c.Entrar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.Entrar(context.Background()); err != nil {
		t.Fatal(err)
	}
	q := e.authz[1]
	if q.Get("client_id") != "oaiapp_teste" || q.Get("agent_name_hint") != "" || q.Get("id_token_hint") == "" ||
		q.Get("ext_agent_host_id") != e.authz[0].Get("ext_agent_host_id") {
		t.Fatalf("relogin: %v", q)
	}
	antes, _ := loja.Ler()
	e.sub = "outra-pessoa"
	if err := c.Entrar(context.Background()); codigoDe(err) != erros.IALoginNaoConcluido || !strings.Contains(err.Error(), "conta diferente") {
		t.Fatalf("outra conta: %v", err)
	}
	if depois, _ := loja.Ler(); string(depois) != string(antes) {
		t.Fatal("credencial trocada por outra conta")
	}
}

// Falhas do login: nada é gravado (exceto conta sem permissão do plano,
// que fica conectada sem gerar texto, como manda a documentação).
// Regra: TRV-SEG-015
func TestEntrarFalhas(t *testing.T) {
	casos := []struct {
		nome   string
		ajuste func(*emissorFalso)
		codigo erros.Codigo
		grava  bool
	}{
		{"negado", func(e *emissorFalso) { e.negar = true }, erros.IALoginNaoConcluido, false},
		{"state forjado", func(e *emissorFalso) { e.stateErrado = true }, erros.IALoginNaoConcluido, false},
		{"nonce errado", func(e *emissorFalso) { e.nonceErrado = true }, erros.IALoginNaoConcluido, false},
		{"conta sem plano (invalid_grant)", func(e *emissorFalso) { e.invalidGrant = true }, erros.IALoginNaoConcluido, false},
		{"registro sem client_id", func(e *emissorFalso) { e.semClientID = true }, erros.IALoginNaoConcluido, false},
		{"sem permissão do plano", func(e *emissorFalso) { e.semPlano = true }, erros.IAPlanoNaoAutorizado, true},
	}
	for _, cs := range casos {
		t.Run(cs.nome, func(t *testing.T) {
			e := novoEmissor(t)
			cs.ajuste(e)
			loja := &credencial.LojaMemoria{}
			c := e.conta(loja)
			err := c.Entrar(context.Background())
			if codigoDe(err) != cs.codigo || loja.Existe() != cs.grava {
				t.Fatalf("%v (gravou=%v)", err, loja.Existe())
			}
			st, _ := c.Estado()
			if cs.grava && (!st.Conectada || st.PlanoAutorizado) {
				t.Fatalf("sem plano: %+v", st)
			}
			if cs.grava {
				if _, err := c.Renovar(context.Background()); codigoDe(err) != erros.IAPlanoNaoAutorizado {
					t.Fatalf("sem plano não gera token utilizável: %v", err)
				}
			}
		})
	}
}

// O retorno é aceito uma única vez; o segundo recebe 409.
func TestRetornoDuplicadoRecusado(t *testing.T) {
	e := novoEmissor(t)
	e.duplicar = true
	c := e.conta(&credencial.LojaMemoria{})
	if err := c.Entrar(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		e.mu.Lock()
		n := len(e.authz)
		e.mu.Unlock()
		if n == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.authz) != 2 || e.authz[1].Get("_segundo_status") == "200" {
		t.Fatalf("segundo retorno deveria ser recusado: %v", e.authz)
	}
}

// Cancelar a entrada encerra a espera (IA-016) e o servidor de retorno.
func TestEntrarCancelado(t *testing.T) {
	e := novoEmissor(t)
	c := Nova(Config{Loja: &credencial.LojaMemoria{}, AbrirURL: func(string) error { return nil }, Emissor: e.srv.URL})
	ctx, cancelar := context.WithCancel(context.Background())
	go func() { time.Sleep(100 * time.Millisecond); cancelar() }()
	if err := c.Entrar(ctx); codigoDe(err) != erros.IALoginNaoConcluido {
		t.Fatalf("%v", err)
	}
}

// Renovação: perto do vencimento, Token renova; rotação gravada; duas
// renovações simultâneas fazem um único POST; refresh morto apaga só os
// tokens (client_id e identidade ficam) e pede reconexão.
// Regra: TRV-SEG-014
func TestRenovacao(t *testing.T) {
	e := novoEmissor(t)
	loja := &credencial.LojaMemoria{}
	agora := time.Now()
	c := Nova(Config{Loja: loja, AbrirURL: e.abrir, Emissor: e.srv.URL, agora: func() time.Time { return agora }})
	if err := c.Entrar(context.Background()); err != nil {
		t.Fatal(err)
	}
	agora = agora.Add(59 * time.Minute) // dentro da margem de renovação
	var wg sync.WaitGroup
	toks := make([]string, 2)
	for i := range toks {
		wg.Add(1)
		go func(i int) { defer wg.Done(); toks[i], _ = c.Token(context.Background()) }(i)
	}
	wg.Wait()
	refreshes := 0
	for _, p := range e.posts {
		if p.Get("grant_type") == "refresh_token" {
			refreshes++
		}
	}
	if refreshes != 1 || toks[0] != "access-2" || toks[1] != "access-2" {
		t.Fatalf("renovação concorrente: %d POSTs, tokens %v", refreshes, toks)
	}
	if b, _ := loja.Ler(); !strings.Contains(string(b), "refresh-2") {
		t.Fatalf("rotação não gravada: %s", b)
	}
	e.refreshMorto = true
	agora = agora.Add(2 * time.Hour)
	if _, err := c.Token(context.Background()); codigoDe(err) != erros.IASessaoExpirada {
		t.Fatalf("refresh morto: %v", err)
	}
	st, _ := c.Estado()
	b, _ := loja.Ler()
	if st.Conectada || !st.Reconectar || !strings.Contains(string(b), "oaiapp_teste") || strings.Contains(string(b), "refresh-2") {
		t.Fatalf("depois do refresh morto: %+v %s", st, b)
	}
}

// Desconectar revoga o refresh token e apaga a credencial local.
func TestDesconectar(t *testing.T) {
	e := novoEmissor(t)
	loja := &credencial.LojaMemoria{}
	c := e.conta(loja)
	if err := c.Entrar(context.Background()); err != nil {
		t.Fatal(err)
	}
	ok, err := c.Desconectar(context.Background())
	if err != nil || !ok || len(e.revog) != 1 || e.revog[0].Get("token") != "refresh-1" || e.revog[0].Get("token_type_hint") != "refresh_token" {
		t.Fatalf("revogação: %v %v %v", ok, err, e.revog)
	}
	st, _ := c.Estado()
	if loja.Existe() || st.Conectada || st.Reconectar {
		t.Fatalf("credencial não apagada: %+v", st)
	}
	if _, err := c.Token(context.Background()); codigoDe(err) != erros.IAContaDesconectada {
		t.Fatalf("depois de desconectar: %v", err)
	}
}
