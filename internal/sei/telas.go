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
// Campo por name OU id, preferindo o visível. No SEI/RO a senha usa o
// componente MaskedPassword: o campo visível tem só id="pwdSenha" (sem name)
// e repassa os caracteres a um campo oculto name="pwdSenha".
const __campo = nome => {
  const todos = [];
  for (const d of __docs) {
    const porId = d.getElementById(nome);
    if (porId) todos.push(porId);
    todos.push(...d.getElementsByName(nome));
  }
  const els = [...new Set(todos)].filter(e => (e.tagName === 'INPUT' || e.tagName === 'SELECT') && e.type !== 'hidden');
  return els.find(__visivel) || els[0] || null;
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
  // Preenche o campo visível; os scripts da página (ex.: MaskedPassword)
  // repassam o valor aos campos ocultos ao receber os eventos.
  c.focus();
  c.value = valor;
  for (const t of ['input', 'change', 'keyup']) c.dispatchEvent(new Event(t, { bubbles: true }));
  return true;
})(%s, %s)`)

// jsFocar põe o foco no campo (pelo name) e o esvazia, para a digitação real.
var jsFocar = script(`((nome) => {
  const c = __campo(nome);
  if (!c) return false;
  c.focus();
  c.value = '';
  c.dispatchEvent(new Event('input', { bubbles: true }));
  return c.ownerDocument.activeElement === c;
})(%s)`)

// jsTamanho devolve só o tamanho do valor do campo (a senha nunca é lida de volta).
var jsTamanho = script(`((nome) => {
  const c = __campo(nome);
  return c ? c.value.length : -1;
})(%s)`)

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

// jsTodosDocumentos garante "Todos os documentos disponíveis" (já vem
// marcada por padrão no SEI/RO). Procura pelo rótulo e pelo id optTodos, que
// é o que a função gerar() da página consulta.
var jsTodosDocumentos = script(`(() => {
  for (const d of __docs) {
    const porId = d.getElementById('optTodos');
    if (porId && porId.type === 'radio') { if (!porId.checked) porId.click(); return porId.checked; }
    for (const r of d.querySelectorAll('input[type=radio]')) {
      const lab = (r.id && d.querySelector('label[for="' + r.id + '"]')) || r.closest('label') || r.nextElementSibling;
      const t = __norm(lab ? lab.innerText || lab.textContent : '');
      if (t.startsWith('todos os documentos disponiveis')) { if (!r.checked) r.click(); return r.checked; }
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

// jsResumo descreve a tela atual para diagnóstico: títulos, ações, campos
// (tipo, visibilidade, trava, TAMANHO do valor e eventos tratados), botões e o
// código das funções chamadas pelos botões (código público do SEI). Nenhum
// valor digitado nem texto da página.
var jsResumo = script(`(() => {
  const acao = d => { try { return new URL(d.location.href).searchParams.get('acao') || d.location.pathname; } catch (e) { return '?'; } };
  const eventos = el => [...el.attributes].filter(a => a.name.startsWith('on')).map(a => a.name + '=' + a.value.slice(0, 80));
  const funcoes = new Set();
  const docs = __docs.map(d => ({
    titulo: (d.title || '').slice(0, 60),
    acao: acao(d),
    campos: [...d.querySelectorAll('input,select,textarea')].filter(e => e.type !== 'hidden').slice(0, 12).map(e => {
      const st = d.defaultView ? d.defaultView.getComputedStyle(e) : null;
      return { nome: e.name || e.id || '?', id: e.id, tipo: e.type, visivel: __visivel(e), oculto: st ? (st.display === 'none' || st.visibility === 'hidden' || st.opacity === '0') : null,
        trava: e.readOnly, desab: e.disabled, tamanho: (e.value || '').length, foco: d.activeElement === e, eventos: eventos(e) };
    }),
    botoes: [...d.querySelectorAll('button,input[type=submit],input[type=button]')].slice(0, 6).map(b => {
      (b.getAttribute('onclick') || '').replace(/([A-Za-z_$][\w$]*)\s*\(/g, (m, f) => { funcoes.add(f); return m; });
      return { rotulo: (b.innerText || b.value || '').trim().slice(0, 25), tipo: b.type, eventos: eventos(b) };
    }),
    formularios: [...d.querySelectorAll('form')].map(f => ({ id: f.id, eventos: eventos(f) })),
  }));
  const codigo = {};
  for (const f of funcoes) {
    if (['return', 'if', 'function'].includes(f)) continue;
    for (const d of __docs) { try { const fn = d.defaultView[f]; if (typeof fn === 'function') { codigo[f] = String(fn).slice(0, 700); break; } } catch (e) {} }
  }
  return JSON.stringify({ docs, codigo });
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
