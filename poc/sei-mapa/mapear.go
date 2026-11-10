package main

// Mapeamento da tela "Gerar Arquivo PDF do Processo" com "Apenas
// selecionados" (pré-requisito da etapa 7 do plano D-11, §4.8). Continua
// SOMENTE observação: nada é clicado, digitado ou gravado.
//
// O que se registra, e como:
//   - FORMA dos textos, nunca o texto: cada palavra vira "Aa" (inicial
//     maiúscula), "AA" (caixa alta) ou "a" (minúscula); cada sequência de
//     dígitos vira "#" repetido (só o comprimento). Ficam em claro apenas
//     palavras de uma lista fixa de tipos documentais e rótulos genéricos
//     da interface (TIPOS abaixo), para saber como o título indica o tipo.
//     Nomes de pessoas, unidades e números nunca aparecem;
//   - identificadores de código (name, id, class) com os dígitos mascarados;
//   - código das funções da página ligadas à seleção, com o conteúdo de
//     toda string entre aspas reduzido à forma e os dígitos mascarados;
//   - no POST da geração (acao=procedimento_gerar_pdf), só os NOMES dos
//     campos e a forma dos valores (ex.: "#######,#######" mostra o
//     separador da lista de documentos). POST de qualquer outra tela não é
//     lido (o login tem usuário e senha).

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// tiposDocumentais: palavras que podem aparecer em claro (sem acento, em
// minúsculas). Só tipos de documento e rótulos genéricos; nenhuma é dado
// pessoal.
const tiposDocumentais = `memorando disciplinar portaria termo inquiricao declaracoes declaracao depoimento interrogatorio
qualificacao defesa previa alegacoes finais final despacho oficio certidao notificacao citacao intimacao relatorio capa anexo
midia requerimento procuracao parecer decisao julgamento solucao sindicancia regular processo administrativo sumario pads
documento documentos externo interno comunicacao informacao ata ficha boletim extrato publicacao mandado convocacao ciencia
juntada remessa encaminhamento recibo laudo auto de da do das dos e a o para por em n nº no tipo data numero sei protocolo
titulo selecionar todos apenas selecionados selecionado exceto gerar fechar pdf arquivo disponiveis disponivel lista
assinado assinatura sigiloso restrito publico nivel acesso`

var reTermo = regexp.MustCompile(`[\p{L}]+|[0-9]+`)

var tipos = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(tiposDocumentais) {
		m[w] = true
	}
	return m
}()

var tirarAcento = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i",
	"ó", "o", "ô", "o", "õ", "o", "ú", "u", "ü", "u", "ç", "c")

func semAcento(s string) string { return tirarAcento.Replace(strings.ToLower(s)) }

// forma reduz um texto à sua forma (mesma regra do JavaScript abaixo).
func forma(t string) string {
	t = strings.Join(strings.Fields(t), " ")
	if r := []rune(t); len(r) > 160 {
		t = string(r[:160])
	}
	return reTermo.ReplaceAllStringFunc(t, func(w string) string {
		if w[0] >= '0' && w[0] <= '9' {
			return strings.Repeat("#", min(len(w), 12))
		}
		if tipos[semAcento(w)] {
			return w
		}
		r := []rune(w)
		switch {
		case w == strings.ToUpper(w) && len(r) > 1:
			return "AA"
		case w == strings.ToUpper(w):
			return "A"
		case unicode.IsUpper(r[0]):
			return "Aa"
		}
		return "a"
	})
}

// postDaGeracao imprime nomes e formas dos campos do POST da geração.
func postDaGeracao(e *network.EventRequestWillBeSent) {
	u, err := url.Parse(e.Request.URL)
	if err != nil || u.Query().Get("acao") != "procedimento_gerar_pdf" || e.Request.Method != "POST" {
		return
	}
	var corpo []byte
	for _, p := range e.Request.PostDataEntries {
		if p != nil {
			corpo = append(corpo, p.Bytes...)
		}
	}
	if len(corpo) == 0 {
		fmt.Println("[post-gerar] corpo não disponível (hasPostData =", e.Request.HasPostData, ")")
		return
	}
	q, err := url.ParseQuery(string(corpo))
	if err != nil {
		fmt.Println("[post-gerar] corpo não é formulário urlencoded (multipart?); tamanho", len(corpo))
		return
	}
	var nomes []string
	for k := range q {
		nomes = append(nomes, k)
	}
	sort.Strings(nomes)
	for _, k := range nomes {
		var vs []string
		for _, v := range q[k] {
			vs = append(vs, fmt.Sprintf("%q", forma(v)))
		}
		fmt.Printf("[post-gerar] %s = %s\n", mask(k), strings.Join(vs, " | "))
	}
}

// mapaGerarJS descreve a tela de gerar PDF em todos os quadros da aba.
const mapaGerarJS = `(() => {
  const TIPOS = new Set(` + "`" + tiposDocumentais + "`" + `.split(/\s+/).filter(Boolean));
  const sa = s => s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase();
  const forma = t => (t || '').replace(/\s+/g, ' ').trim().slice(0, 160).replace(/[\p{L}]+|[0-9]+/gu, w => {
    if (/^[0-9]+$/.test(w)) return '#'.repeat(Math.min(w.length, 12));
    if (TIPOS.has(sa(w))) return w;
    if (w === w.toUpperCase()) return w.length > 1 ? 'AA' : 'A';
    if (w[0] === w[0].toUpperCase()) return 'Aa';
    return 'a';
  });
  const ident = s => (s || '').replace(/[0-9]/g, '#').slice(0, 80);
  const literal = /(['"` + "`" + `])((?:\\.|(?!\1)[^\\\n])*)\1/g;
  const codigo = (s, max) => (s || '').replace(literal, (m, q, c) => q + forma(c) + q).replace(/[0-9]/g, '#').slice(0, max);
  const attrs = el => [...el.attributes].map(a => a.name);
  const visivel = el => { try { const cs = el.ownerDocument.defaultView.getComputedStyle(el); return cs.display !== 'none' && cs.visibility !== 'hidden'; } catch (e) { return null; } };
  const resumoCelula = td => ({
    texto: forma(td.innerText),
    elementos: [...td.querySelectorAll('input,a,img,select,button')].slice(0, 6).map(x => {
      let acao = null; try { if (x.href) acao = new URL(x.href).searchParams.get('acao'); } catch (e) {}
      return x.tagName.toLowerCase() + (x.type ? ':' + x.type : '') + (x.name ? ' name=' + ident(x.name) : '') + (x.id ? ' id=' + ident(x.id) : '') +
        (x.value && x.tagName !== 'A' ? ' value=' + forma(x.value) : '') + (acao ? ' acao=' + acao : '') + (x.title ? ' title=' + forma(x.title) : '');
    })
  });
  const handlers = new Set(['gerar']);
  const anotar = el => ['onclick', 'onchange', 'onkeyup', 'onsubmit'].forEach(h => {
    const v = el.getAttribute && el.getAttribute(h);
    if (v) (v.match(/[A-Za-z_$][\w$]*(?=\s*\()/g) || []).forEach(n => handlers.add(n));
  });
  const out = [];
  const descrever = (doc, fm, caminho, acao) => {
    const r = { quadro: caminho, acao, form: fm ? { id: ident(fm.id), metodo: (fm.method || '').toUpperCase(), atributos: attrs(fm) } : null };
    r.radios = [...doc.querySelectorAll('input[type=radio]')].map(x => { anotar(x); return { name: ident(x.name), id: ident(x.id), value: forma(x.value), marcado: x.checked, onclick: codigo(x.getAttribute('onclick'), 300), onchange: codigo(x.getAttribute('onchange'), 300) }; });
    r.ocultos = [...doc.querySelectorAll('input[type=hidden]')].map(x => ident(x.name || x.id) + '=' + forma(x.value).slice(0, 60));
    r.selects = [...doc.querySelectorAll('select')].map(s => { anotar(s); return {
      name: ident(s.name), id: ident(s.id), multiplo: s.multiple, tamanho: s.size, desabilitado: s.disabled, visivel: visivel(s),
      atributos: attrs(s), opcoes: s.options.length,
      amostra: [...s.options].slice(0, 10).map(o => ({ texto: forma(o.text), valor: forma(o.value), atributos: attrs(o), selecionada: o.selected }))
    }; });
    const caixas = [...doc.querySelectorAll('input[type=checkbox]')];
    caixas.forEach(anotar);
    r.caixas = { total: caixas.length, amostra: caixas.slice(0, 10).map(c => {
      const tr = c.closest('tr');
      return { name: ident(c.name), id: ident(c.id), value: forma(c.value), title: forma(c.title), marcada: c.checked, atributos: attrs(c),
        onclick: codigo(c.getAttribute('onclick'), 300), linha: tr ? [...tr.cells].map(resumoCelula) : null };
    }) };
    r.tabelas = [...doc.querySelectorAll('table')].filter(t => t.rows.length > 1).slice(0, 6).map(t => ({
      id: ident(t.id), classe: ident(t.className), visivel: visivel(t), linhas: t.rows.length, legenda: forma(t.caption ? t.caption.innerText : (t.getAttribute('summary') || '')),
      cabecalho: [...t.querySelectorAll('th')].slice(0, 12).map(th => forma(th.innerText)),
      amostra: [...t.rows].slice(0, 4).map(tr => [...tr.cells].map(resumoCelula))
    }));
    r.botoes = [...doc.querySelectorAll('button,input[type=button],input[type=submit]')].map(b => { anotar(b); return { name: ident(b.name), id: ident(b.id), texto: forma(b.innerText || b.value), onclick: codigo(b.getAttribute('onclick'), 300) }; });
    r.scripts_externos = [...doc.querySelectorAll('script[src]')].map(s => { try { return ident(new URL(s.src).pathname); } catch (e) { return '?'; } });
    const chave = /DocumentosApenas|DocumentosExceto|optApenas|optExceto|rdoTipo|selDocumentos|hdnFlagGerar|chkInfra|InfraItem/;
    const funcoes = {};
    [...doc.querySelectorAll('script:not([src])')].forEach(s => {
      const t = s.textContent || '';
      const re = /function\s+([A-Za-z_$][\w$]*)\s*\(/g; let m;
      while ((m = re.exec(t))) {
        let i = t.indexOf('{', m.index), n = 0, j = i;
        for (; j < t.length; j++) { if (t[j] === '{') n++; else if (t[j] === '}' && --n === 0) break; }
        const corpo = t.slice(m.index, j + 1);
        if (chave.test(corpo) || handlers.has(m[1])) funcoes[m[1]] = codigo(corpo, 4000);
      }
    });
    const win = doc.defaultView;
    handlers.forEach(n => { try { if (!funcoes[n] && win && typeof win[n] === 'function') funcoes[n] = '(global) ' + codigo(win[n].toString(), 4000); } catch (e) {} });
    r.funcoes = funcoes;
    return r;
  };
  const walk = (doc, caminho) => {
    let acao = null; try { acao = new URL(doc.location.href).searchParams.get('acao'); } catch (e) {}
    const fm = doc.getElementById('frmProcedimentoPdf');
    if (fm || acao === 'procedimento_gerar_pdf') out.push(descrever(doc, fm || doc.querySelector('form'), caminho, acao));
    doc.querySelectorAll('iframe,frame').forEach((fr, i) => {
      try { if (fr.contentDocument) walk(fr.contentDocument, caminho + '/' + (fr.name || fr.id || ('iframe' + i))); } catch (e) {}
    });
  };
  walk(document, 'top');
  return JSON.stringify(out);
})()`

// mapearGerar roda o mapeamento em todas as abas observadas.
func mapearGerar(abas []context.Context) {
	achou := false
	for _, tctx := range abas {
		raw, err := chromedp.Run(tctx, chromedp.Evaluate[string](mapaGerarJS))
		if err != nil || raw == "" || raw == "[]" {
			continue
		}
		achou = true
		fmt.Println("[gerar-pdf]", mask(raw))
	}
	if !achou {
		fmt.Println("[gerar-pdf] a tela \"Gerar Arquivo PDF do Processo\" não está aberta em nenhuma aba observada.")
	}
}
