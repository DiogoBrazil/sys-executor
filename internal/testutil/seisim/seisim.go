// Package seisim é um simulador FICTÍCIO do SEI para testes automatizados do
// cliente. Reproduz só a estrutura mapeada na F0-08 (nomes de campos, iframes,
// ações, modal de reautenticação, botão btnGerar, download por ifrDownload).
// Algumas páginas tentam, de propósito, disparar ações de escrita (imagem,
// fetch e link) para provar que a guarda do cliente as bloqueia.
package seisim

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"
)

// Simulador do SEI.
type Simulador struct {
	Usuario, Senha, OTP string
	ExigirOTP           bool
	ProcessoNormal      string
	ProcessoSigiloso    string
	PDF                 []byte

	mu       sync.Mutex
	acoes    map[string]int // requisições recebidas por ação (acao=, acao_ajax= ou caminho)
	logado   bool
	liberado bool
}

// Novo cria o simulador com dados fictícios.
func Novo(pdf []byte) *Simulador {
	return &Simulador{
		Usuario: "usuario.teste", Senha: "senha-certa", OTP: "123456", ExigirOTP: true,
		ProcessoNormal: "0000.000001/2026-01", ProcessoSigiloso: "0000.000002/2026-02",
		PDF: pdf, acoes: map[string]int{},
	}
}

// Recebidas devolve quantas vezes a ação chegou ao servidor.
func (s *Simulador) Recebidas(acao string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.acoes[acao]
}

func pagina(w http.ResponseWriter, titulo, corpo string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title></head><body>%s</body></html>`, html.EscapeString(titulo), corpo)
}

func (s *Simulador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	acao := r.URL.Query().Get("acao")
	if a := r.URL.Query().Get("acao_ajax"); a != "" {
		acao = "ajax:" + a
	}
	chave := acao
	if chave == "" {
		chave = r.URL.Path
	}
	s.mu.Lock()
	s.acoes[chave]++
	logado := s.logado
	s.mu.Unlock()

	switch r.URL.Path {
	case "/sip/login.php":
		s.login(w, r)
		return
	case "/sei/inicializar.php":
		http.Redirect(w, r, "/sei/controlador.php?acao=procedimento_controlar&infra_hash=h1", http.StatusFound)
		return
	case "/sei/controlador.php", "/sei/controlador_ajax.php":
	default:
		http.NotFound(w, r)
		return
	}
	if !logado {
		http.Redirect(w, r, "/sip/login.php", http.StatusFound)
		return
	}
	switch acao {
	case "procedimento_controlar":
		pagina(w, "SEI - Controle de Processos", `
<form method="post" action="controlador.php?acao=protocolo_pesquisa_rapida&infra_hash=h2">
  <input type="text" name="txtPesquisaRapida"> <img title="Pesquisa Rápida" alt="Pesquisa Rápida">
</form>
<h1>Controle de Processos</h1>
<a href="controlador.php?acao=procedimento_excluir&infra_hash=x">Excluir</a>
<img src="controlador.php?acao=procedimento_concluir&infra_hash=x" alt="">
<script>fetch('controlador.php?acao=documento_assinar&infra_hash=x', {method: 'POST'}).catch(() => {});</script>
<img src="http://127.0.0.2:9/rastreador.png" alt="">`)
	case "protocolo_pesquisa_rapida":
		p := strings.TrimSpace(r.PostFormValue("txtPesquisaRapida"))
		switch p {
		case s.ProcessoNormal:
			s.mu.Lock()
			s.liberado = true
			s.mu.Unlock()
			http.Redirect(w, r, "/sei/controlador.php?acao=procedimento_trabalhar&id_procedimento=1&infra_hash=h3", http.StatusFound)
		case s.ProcessoSigiloso:
			s.mu.Lock()
			s.liberado = false
			s.mu.Unlock()
			http.Redirect(w, r, "/sei/controlador.php?acao=procedimento_trabalhar&id_procedimento=2&infra_hash=h3", http.StatusFound)
		default:
			pagina(w, "SEI - Pesquisa", `<div class="infraExcecao">Processo não encontrado.</div>
<form method="post" action="controlador.php?acao=protocolo_pesquisa_rapida&infra_hash=h2"><input type="text" name="txtPesquisaRapida"></form>`)
		}
	case "procedimento_trabalhar":
		s.mu.Lock()
		liberado := s.liberado || r.URL.Query().Get("acesso") == "1"
		s.mu.Unlock()
		if !liberado {
			pagina(w, "SEI - 0000.000002/2026-02", `<iframe name="modal-frame" id="modal-frame" src="controlador.php?acao=usuario_validar_acesso&id_procedimento=2&infra_hash=h4" style="width:600px;height:300px"></iframe>`)
			return
		}
		pagina(w, "SEI - 0000.000001/2026-01", `
<iframe name="ifrArvore" src="controlador.php?acao=procedimento_visualizar&infra_hash=h5" style="width:300px;height:500px"></iframe>
<iframe name="ifrConteudoVisualizacao" src="controlador.php?acao=arvore_visualizar&infra_hash=h6" style="width:800px;height:500px"></iframe>`)
	case "usuario_validar_acesso":
		if r.Method == http.MethodPost {
			if r.PostFormValue("pwdSenha") != s.Senha {
				pagina(w, "SEI - Identificação de Acesso", `<p>Senha inválida.</p>`+formValidar(s.Usuario))
				return
			}
			s.mu.Lock()
			s.liberado = true
			s.mu.Unlock()
			pagina(w, "SEI", `<script>window.top.location.href = 'controlador.php?acao=procedimento_trabalhar&acesso=1&id_procedimento=2&infra_hash=h7';</script>`)
			return
		}
		pagina(w, "SEI - Identificação de Acesso", formValidar(s.Usuario))
	case "procedimento_visualizar":
		pagina(w, "SEI - Árvore Montar", `<p>Documentos do processo (fictício)</p>`)
	case "arvore_visualizar":
		pagina(w, "SEI - Visualizar Árvore", `
<a href="controlador.php?acao=procedimento_concluir&infra_hash=x"><img title="Concluir Processo" alt="Concluir Processo"></a>
<a href="controlador.php?acao=procedimento_gerar_pdf&acao_origem=arvore_visualizar&arvore=1&infra_hash=h8" tabindex="451" target="ifrVisualizacao" onmouseover="return true" onmouseout="return true"><img src="data:image/gif;base64,R0lGODlhAQABAAAAACw=" alt="Gerar Arquivo PDF do Processo" data-title-pro="Gerar Arquivo PDF do Processo" width="24" height="24"></a>
<iframe name="ifrVisualizacao" src="controlador.php?acao=arvore_processar_html&infra_hash=h9" style="width:780px;height:400px"></iframe>`)
	case "arvore_processar_html":
		pagina(w, "SEI", `<p>Processo aberto (fictício).</p>`)
	case "procedimento_gerar_pdf":
		if r.Method == http.MethodPost {
			if r.PostFormValue("rdoTipo") != "T" || r.PostFormValue("hdnFlagGerar") != "1" {
				pagina(w, "SEI - Gerar Arquivo PDF do Processo", `<p>Opção inesperada.</p>`)
				return
			}
			pagina(w, "SEI - Gerar Arquivo PDF do Processo", `<p>Gerando…</p>
<iframe name="ifrDownload" src="controlador.php?acao=exibir_arquivo&nome_arquivo=x&nome_download=SEI_0000.000001_2026_01.pdf&infra_hash=h10" style="display:none"></iframe>`)
			return
		}
		pagina(w, "SEI - Gerar Arquivo PDF do Processo", `
<h2>Gerar Arquivo PDF do Processo</h2>
<form id="frmProcedimentoPdf" method="post" action="controlador.php?acao=procedimento_gerar_pdf&arvore=1&infra_hash=h8">
  <button type="button" accesskey="G" name="btnGerar" value="Gerar" onclick="gerar();" class="infraButton">Gerar</button>
  <button type="button" name="btnFechar">Fechar</button>
  <input type="radio" name="rdoTipo" id="optTodos" value="T" checked> <label for="optTodos">Todos os documentos disponíveis</label>
  <input type="radio" name="rdoTipo" id="optExceto" value="E"> <label for="optExceto">Todos exceto selecionados</label>
  <input type="radio" name="rdoTipo" id="optApenas" value="A"> <label for="optApenas">Apenas selecionados</label>
  <input type="hidden" name="hdnFlagGerar" id="hdnFlagGerar" value="">
  <input type="hidden" name="hdnDocumentosApenas" id="hdnDocumentosApenas" value="">
</form>
<iframe name="ifrDownload" style="display:none"></iframe>
<script>
// Espelha a função gerar() real do SEI/RO (informada pelo usuário).
function gerar() {
  if (!document.getElementById('optTodos').checked && !document.getElementById('optExceto').checked && !document.getElementById('optApenas').checked) {
    alert('Nenhuma opção selecionada.');
    return;
  }
  if (document.getElementById('optApenas').checked && document.getElementById('hdnDocumentosApenas').value == '') {
    alert('Nenhum documento selecionado.');
    return;
  }
  if (parent.parent.parent.infraExibirAviso) parent.parent.parent.infraExibirAviso();
  document.getElementById('hdnFlagGerar').value = '1';
  document.getElementById('frmProcedimentoPdf').submit();
}
</script>`)
	case "exibir_arquivo":
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="SEI_0000.000001_2026_01.pdf"`)
		_, _ = w.Write(s.PDF)
	default:
		// Qualquer outra ação (escrita) "executaria" algo no SEI real.
		pagina(w, "SEI", `<p>Ação executada.</p>`)
	}
}

func formValidar(usuario string) string {
	return `<h2>Identificação de Acesso</h2>
<form method="post" action="controlador.php?acao=usuario_validar_acesso&id_procedimento=2&infra_hash=h4">
  <label>Usuário:</label><input type="text" name="txtUsuario" value="` + html.EscapeString(usuario) + `" readonly>
  <label>Senha:</label><input type="password" name="pwdSenha"><input type="text" name="pwdSenha" style="display:none">
  <input type="hidden" name="hdnInfraTipoPagina" value="2">
  <button type="submit" name="sbmAcessar">Acessar</button>
</form>`
}

func (s *Simulador) login(w http.ResponseWriter, r *http.Request) {
	formLogin := func(msg string) {
		pagina(w, "SEI / RO", msg+`
<form method="post" action="/sip/login.php?sigla_orgao_sistema=RO&sigla_sistema=SEI">
  <input type="text" name="txtUsuario"> <input type="password" name="pwdSenha"> <input type="text" name="pwdSenha" style="display:none">
  <select name="selOrgao"><option value="0">Selecione</option><option value="10">PM</option><option value="11">SESDEC</option></select>
  <input type="hidden" name="hdnAcao" value="2">
  <button type="submit" name="sbmLogin">ACESSAR</button>
  <a href="#">Entrar com gov.br</a> <a href="#">Autenticação em dois fatores</a>
</form>`)
	}
	formOTP := func(msg string) {
		pagina(w, "SEI / RO", msg+`
<form method="post" action="/sip/login.php?sigla_orgao_sistema=RO&sigla_sistema=SEI">
  <p>Informe o código de 6 números gerado pelo aplicativo de autenticação em 2 fatores:</p>
  <input type="text" name="txtCodigoAcesso"> <input type="checkbox" name="chkLiberar"> Não usar 2FA neste dispositivo e navegador
  <input type="hidden" name="hdnAcao" value="3">
  <button type="submit" name="sbmValidar">Validar</button> <button type="submit" name="sbmDesativar" formaction="/sip/login.php?desativar=1">Desativar 2FA</button>
</form>`)
	}
	if r.Method != http.MethodPost {
		formLogin("")
		return
	}
	entrar := func() {
		s.mu.Lock()
		s.logado = true
		s.mu.Unlock()
		http.Redirect(w, r, "/sei/inicializar.php?id_login=1", http.StatusFound)
	}
	switch r.PostFormValue("hdnAcao") {
	case "2":
		if r.PostFormValue("txtUsuario") != s.Usuario || r.PostFormValue("pwdSenha") != s.Senha || r.PostFormValue("selOrgao") != "10" {
			formLogin(`<div class="infraExcecao">Usuário ou senha inválida.</div>`)
			return
		}
		if s.ExigirOTP {
			formOTP("")
			return
		}
		entrar()
	case "3":
		if r.URL.Query().Get("desativar") != "" || r.PostFormValue("chkLiberar") != "" {
			s.mu.Lock()
			s.acoes["2fa_alterado"]++
			s.mu.Unlock()
		}
		if r.PostFormValue("txtCodigoAcesso") != s.OTP {
			formOTP(`<div class="infraExcecao">Código inválido.</div>`)
			return
		}
		entrar()
	default:
		formLogin("")
	}
}
