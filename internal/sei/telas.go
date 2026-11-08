package sei

// Script de leitura e interação com as telas do SEI. Percorre o documento e
// os iframes de mesma origem (o SEI usa ifrArvore, ifrConteudoVisualizacao,
// ifrVisualizacao, modal-frame). Localiza elementos por name e por rótulo
// visível, nunca por IDs dinâmicos. Devolve só indicadores booleanos: nenhum
// texto da página sai do navegador.
const scriptBase = `
const __docs = [];
(function walk(d) {
  __docs.push(d);
  d.querySelectorAll('iframe,frame').forEach(f => { try { if (f.contentDocument) walk(f.contentDocument); } catch (e) {} });
})(document);
const __norm = s => (s || '').normalize('NFD').replace(/[̀-ͯ]/g, '').replace(/\s+/g, ' ').trim().toLowerCase();
const __visivel = el => { const r = el.getBoundingClientRect(); return r.width > 0 && r.height > 0; };
const __campo = nome => {
  for (const d of __docs) {
    const els = [...d.getElementsByName(nome)].filter(e => (e.tagName === 'INPUT' || e.tagName === 'SELECT') && e.type !== 'hidden');
    const v = els.find(__visivel) || els[0];
    if (v) return v;
  }
  return null;
};
const __clicaveis = () => __docs.flatMap(d => [...d.querySelectorAll('button,input[type=submit],input[type=button],a,img[title],img[alt]')]);
const __rotulo = el => __norm(el.innerText || el.value || el.title || el.alt || el.getAttribute('aria-label'));
const __achar = alvo => {
  const a = __norm(alvo);
  const els = __clicaveis();
  return els.find(e => __rotulo(e) === a) || els.find(e => __rotulo(e).includes(a)) || null;
};
const __textoPagina = () => __norm(__docs.map(d => d.body ? d.body.innerText : '').join(' '));
`

// jsEstado resume a tela atual em indicadores.
var jsEstado = script(`(() => {
  const t = __textoPagina();
  const titulos = __norm(__docs.map(d => d.title).join(' | '));
  return JSON.stringify({
    login: !!__campo('txtUsuario') && !!__campo('pwdSenha') && !!__campo('selOrgao'),
    otp: !!__campo('txtCodigoAcesso'),
    pesquisa: !!__campo('txtPesquisaRapida'),
    reautenticacao: titulos.includes('identificacao de acesso') || __docs.some(d => /acao=usuario_validar_acesso/.test(d.location.href) && d.getElementsByName('pwdSenha').length > 0),
    processoAberto: __docs.some(d => /acao=(arvore_visualizar|procedimento_visualizar)/.test(d.location.href)),
    iconePDF: !!__achar('Gerar Arquivo PDF do Processo'),
    formGerar: __docs.some(d => d.getElementsByName('btnGerar').length > 0),
    credencialInvalida: /(usuario|senha).{0,40}(invalid|incorret)|(invalid|incorret).{0,40}(usuario|senha)/.test(t),
    codigoInvalido: /codigo.{0,40}(invalid|incorret|expirad)/.test(t),
    naoEncontrado: /(processo|protocolo).{0,60}nao (foi )?encontrad|nenhum (registro|resultado) encontrad/.test(t),
    acessoNegado: /acesso negado|sem permissao|nao possui (acesso|permissao)/.test(t),
  });
})()`)

// jsPreencher define o valor de um campo pelo name (o valor vem como JSON).
var jsPreencher = script(`((nome, valor) => {
  const c = __campo(nome);
  if (!c) return false;
  if (c.tagName === 'SELECT') {
    const v = __norm(valor);
    const op = [...c.options].find(o => __norm(o.text) === v || __norm(o.value) === v);
    if (!op) return false;
    c.value = op.value;
    c.dispatchEvent(new Event('change', { bubbles: true }));
    return true;
  }
  // O SEI/RO tem campos homônimos (ex.: pwdSenha como senha e como texto do
  // "mostrar senha"): preenche todos os do mesmo formulário.
  const alvos = [...(c.form ? c.form.elements : [c])].filter(e => e.name === nome && e.tagName === 'INPUT' && e.type !== 'hidden');
  for (const e of (alvos.length ? alvos : [c])) {
    e.focus();
    e.value = valor;
    for (const t of ['input', 'change', 'keyup']) e.dispatchEvent(new Event(t, { bubbles: true }));
  }
  return true;
})(%s, %s)`)

// jsClicar aciona o elemento cujo rótulo (texto, value, title ou alt) é o dado.
// Uma imagem dentro de link aciona o link.
var jsClicar = script(`((alvo) => {
  let el = __achar(alvo);
  if (!el) return false;
  if (el.tagName === 'IMG' && el.closest('a')) el = el.closest('a');
  el.click();
  return true;
})(%s)`)

// jsSubmeterPesquisa envia o formulário da pesquisa rápida.
var jsSubmeterPesquisa = script(`(() => {
  const c = __campo('txtPesquisaRapida');
  if (!c || !c.form) return false;
  if (c.form.requestSubmit) c.form.requestSubmit(); else c.form.submit();
  return true;
})()`)

// jsTodosDocumentos marca "Todos os documentos disponíveis" (já vem marcado
// por padrão no SEI/RO; marcar de novo é inofensivo).
var jsTodosDocumentos = script(`(() => {
  for (const d of __docs) {
    for (const r of d.querySelectorAll('input[type=radio]')) {
      const lab = r.id ? d.querySelector('label[for="' + r.id + '"]') : r.closest('label');
      if (lab && __norm(lab.innerText).startsWith('todos os documentos disponiveis')) { r.click(); return true; }
    }
  }
  return false;
})()`)

// jsGerar aciona o botão "Gerar" (name=btnGerar) da tela de geração do PDF.
var jsGerar = script(`(() => {
  for (const d of __docs) {
    const b = d.getElementsByName('btnGerar')[0];
    if (b) { b.click(); return true; }
  }
  return false;
})()`)

// script isola cada avaliação numa função: declarações não vazam para o
// escopo global da página entre chamadas.
func script(corpo string) string {
	return "(() => {\n" + scriptBase + "\nreturn (" + corpo + ");\n})()"
}

// jsResumo descreve a tela atual para diagnóstico: títulos, ações, nomes de
// campos e rótulos de botões. Nenhum valor digitado ou texto da página.
var jsResumo = script(`(() => {
  const acao = d => { try { return new URL(d.location.href).searchParams.get('acao') || d.location.pathname; } catch (e) { return '?'; } };
  return JSON.stringify(__docs.map(d => ({
    titulo: (d.title || '').slice(0, 60),
    acao: acao(d),
    campos: [...d.querySelectorAll('input,select,textarea')].filter(e => e.type !== 'hidden').map(e => e.name || e.id || '?').slice(0, 12),
    botoes: [...d.querySelectorAll('button,input[type=submit],input[type=button]')].map(b => (b.innerText || b.value || '').trim().slice(0, 25)).filter(Boolean).slice(0, 8),
  })));
})()`)

// estadoTela é o resumo devolvido por jsEstado.
type estadoTela struct {
	Login              bool `json:"login"`
	OTP                bool `json:"otp"`
	Pesquisa           bool `json:"pesquisa"`
	Reautenticacao     bool `json:"reautenticacao"`
	ProcessoAberto     bool `json:"processoAberto"`
	IconePDF           bool `json:"iconePDF"`
	FormGerar          bool `json:"formGerar"`
	CredencialInvalida bool `json:"credencialInvalida"`
	CodigoInvalido     bool `json:"codigoInvalido"`
	NaoEncontrado      bool `json:"naoEncontrado"`
	AcessoNegado       bool `json:"acessoNegado"`
}
