// PoC F0-08 — Mapeamento das telas do SEI/RO, EXECUTADO PELO PRÓPRIO USUÁRIO.
//
// O que faz: abre o Microsoft Edge com perfil descartável (apagado ao sair). O USUÁRIO faz login
// e navega manualmente (somente consulta). A ferramenta NÃO clica, NÃO digita e NÃO grava nada em
// disco: apenas observa e imprime no console metadados ESTRUTURAIS:
//   - valores de `acao=` e NOMES (não valores) dos parâmetros de cada navegação;
//   - hosts contatados;
//   - estrutura de formulários (tipo/nome dos campos, rótulos) e textos de botões;
//   - links/opções cujo texto coincide com rótulos conhecidos da interface (ex.: "Gerar Arquivo PDF do Processo");
//   - início de download (nome sugerido com dígitos mascarados) — o download é BLOQUEADO.
//
// Todos os dígitos são mascarados como '#'. Nenhum texto de documento é lido.
//
// Etapa 7 do plano D-11 (mapear.go): comando "mapear" (forma da tela "Gerar
// Arquivo PDF do Processo", sem texto) e nomes/forma dos campos do POST da
// geração. Ver LEIA-ME.md.
//
// Uso (Windows): sei-mapa.exe [-url <login>] [-segundos N]
package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

const loginPadrao = "https://sei.sistemas.ro.gov.br/sip/login.php?sigla_orgao_sistema=RO&sigla_sistema=SEI"

var digitos = regexp.MustCompile(`\d`)

func mask(s string) string { return digitos.ReplaceAllString(s, "#") }

// Coletado no navegador. Lê apenas estrutura e rótulos de controles, nunca o conteúdo dos documentos.
const snapshotJS = `(() => {
  const ROTULOS = [/gerar arquivo pdf/i, /todos os documentos/i, /^gerar$/i, /pesquis/i, /acessar/i, /entrar/i,
    /autentica/i, /c[oó]digo/i, /senha/i, /usu[aá]rio/i, /[oó]rg[aã]o/i, /download/i, /\bpdf\b/i, /controle de processos/i, /sigilo|restrit/i];
  const conhecido = t => ROTULOS.some(r => r.test(t));
  const limpa = t => (t || '').replace(/\s+/g, ' ').trim().slice(0, 80);
  const out = [];
  const walk = (doc, caminho) => {
    let acao = null; try { acao = new URL(doc.location.href).searchParams.get('acao'); } catch (e) {}
    const f = { quadro: caminho, acao, titulo: limpa(doc.title), formularios: [], botoes: [], rotulos: [] };
    doc.querySelectorAll('form').forEach(fm => {
      let fa = null; try { fa = new URL(fm.action, doc.location.href).searchParams.get('acao'); } catch (e) {}
      const campos = [...fm.querySelectorAll('input,select,textarea')].filter(i => i.type !== 'hidden')
        .map(i => (i.tagName.toLowerCase() + ':' + (i.type || '') + ':' + (i.name || i.id || '?')));
      const ocultos = [...fm.querySelectorAll('input[type=hidden]')].map(i => i.name || i.id || '?');
      f.formularios.push({ metodo: (fm.method || 'get').toUpperCase(), acao_destino: fa, campos, ocultos });
    });
    doc.querySelectorAll('button,input[type=submit],input[type=button]').forEach(b => {
      const t = limpa(b.innerText || b.value || b.title); if (t) f.botoes.push(t);
    });
    doc.querySelectorAll('label,a,option,img[title],img[alt]').forEach(e => {
      const t = limpa(e.innerText || e.title || e.alt);
      if (t && conhecido(t)) {
        let ea = null; try { if (e.href) ea = new URL(e.href).searchParams.get('acao'); } catch (x) {}
        f.rotulos.push(e.tagName.toLowerCase() + ':' + t + (ea ? ' → acao=' + ea : ''));
      }
    });
    f.botoes = [...new Set(f.botoes)]; f.rotulos = [...new Set(f.rotulos)];
    out.push(f);
    doc.querySelectorAll('iframe,frame').forEach((fr, i) => {
      try { if (fr.contentDocument) walk(fr.contentDocument, caminho + '/' + (fr.name || fr.id || ('iframe' + i))); }
      catch (e) { out.push({ quadro: caminho + '/' + (fr.name || fr.id || i), erro: 'cross-origin' }); }
    });
  };
  walk(document, 'top');
  return JSON.stringify(out);
})()`

type estado struct {
	mu        sync.Mutex
	hosts     map[string]int
	acoes     map[string]map[string]bool // acao -> nomes de parâmetros
	ordem     []string
	vistos    map[string]bool
	downloads []string
	reqs      map[network.RequestID]string // requestId → chave (só documentos), para diagnóstico
	dialogo   context.Context              // diálogo confirm/prompt aguardando resposta do usuário
	dialogos  []string
	abas      []context.Context // abas observadas (para o comando "mapear")
}

// responderDialogo resolve o diálogo JavaScript pendente (fora do handler de eventos, para não travar).
func responderDialogo(tctx context.Context, aceitar bool) {
	go func() {
		if _, err := chromedp.Call(tctx, page.HandleJavaScriptDialog, page.HandleJavaScriptDialogParams{Accept: aceitar}); err != nil {
			fmt.Println("[dialogo] erro ao responder:", err)
		}
	}()
}

func limita(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func main() {
	login := flag.String("url", loginPadrao, "URL inicial")
	segundos := flag.Int("segundos", 0, "encerrar após N segundos (0 = até digitar 'sair')")
	flag.Parse()

	fmt.Println("== sys-executor F0-08 — mapeamento do SEI (somente observação, nada é gravado) ==")
	fmt.Println("Faça login e navegue SOMENTE em consulta: abrir processo → 'Gerar Arquivo PDF do Processo' →")
	fmt.Println("'Todos os documentos disponíveis' → 'Gerar'. O download será bloqueado de propósito.")
	fmt.Println("Etapa 7: na tela de gerar, marque 'Apenas selecionados', marque 1 ou 2 documentos e digite 'mapear' + Enter.")
	fmt.Println("Para terminar, digite 'sair' + Enter aqui. Depois copie o RESUMO final.")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`),
		chromedp.Flag("disable-sync", true), chromedp.Flag("no-first-run", true),
		chromedp.Flag("disable-features", "msEdgeSignIn,EdgeCollections,msShopping"),
		chromedp.VisibleWindow,
	)
	actx, cancelA := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelA()
	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()

	st := &estado{hosts: map[string]int{}, acoes: map[string]map[string]bool{}, vistos: map[string]bool{}, reqs: map[network.RequestID]string{}}
	snap := make(chan context.Context, 8)
	escutar := func(tctx context.Context) {
		st.mu.Lock()
		st.abas = append(st.abas, tctx)
		st.mu.Unlock()
		chromedp.Listen(tctx, func(ev any) {
			switch e := ev.(type) {
			case *network.EventRequestWillBeSent:
				u, err := url.Parse(e.Request.URL)
				if err != nil || u.Host == "" {
					return
				}
				st.mu.Lock()
				st.hosts[u.Host]++
				if e.Type == network.ResourceTypeDocument || e.Type == network.ResourceTypeXHR || e.Type == network.ResourceTypeFetch {
					q := u.Query()
					a := q.Get("acao")
					if a == "" {
						a = "(sem acao) " + mask(u.Path)
					}
					chave := string(e.Type) + " " + e.Request.Method + " " + a
					if e.Type == network.ResourceTypeDocument {
						st.reqs[e.RequestID] = chave
						postDaGeracao(e) // só o POST de procedimento_gerar_pdf; nomes e formas
					}
					if st.acoes[chave] == nil {
						st.acoes[chave] = map[string]bool{}
						st.ordem = append(st.ordem, chave)
						fmt.Printf("[nav] %s  %s%s\n", chave, u.Host, mask(u.Path))
					}
					for k := range q {
						st.acoes[chave][k] = true
					}
				}
				st.mu.Unlock()
			case *network.EventResponseReceived:
				st.mu.Lock()
				if chave, ok := st.reqs[e.RequestID]; ok && e.Response != nil {
					fmt.Printf("[resp] HTTP %d  %s\n", e.Response.Status, chave)
				}
				st.mu.Unlock()
			case *network.EventLoadingFailed:
				st.mu.Lock()
				if chave, ok := st.reqs[e.RequestID]; ok {
					fmt.Printf("[falha] %s  erro=%q cancelado=%v bloqueio=%q\n", chave, e.ErrorText, e.Canceled, string(e.BlockedReason))
				}
				st.mu.Unlock()
			case *page.EventJavascriptDialogOpening:
				msg := mask(limita(e.Message, 120))
				st.mu.Lock()
				st.dialogos = append(st.dialogos, string(e.Type)+": "+msg)
				st.mu.Unlock()
				fmt.Printf("[dialogo] tipo=%s navegador_exibe=%v mensagem=%q\n", e.Type, e.HasBrowserHandler, msg)
				switch {
				case e.HasBrowserHandler:
					fmt.Println("[dialogo] responda na própria janela do Edge.")
				case e.Type == page.DialogTypeAlert:
					responderDialogo(tctx, true) // alert só tem "OK": equivale ao clique do usuário
					fmt.Println("[dialogo] alert confirmado com OK automaticamente.")
				default:
					st.mu.Lock()
					st.dialogo = tctx
					st.mu.Unlock()
					fmt.Println("[dialogo] >>> a página está PARADA aguardando você. Digite 'aceitar' ou 'cancelar' + Enter aqui.")
				}
			case *page.EventLoadEventFired, *page.EventFrameStoppedLoading:
				select {
				case snap <- tctx:
				default:
				}
			}
		})
	}
	escutar(ctx)

	// Novas abas/janelas (popups do SEI): anexar e observar também.
	chromedp.ListenBrowser(ctx, func(ev any) {
		if e, ok := ev.(*target.EventTargetCreated); ok && e.TargetInfo.Type == "page" {
			go func() {
				tctx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(e.TargetInfo.TargetID))
				_, err1 := chromedp.Call(tctx, network.Enable, network.EnableParams{})
				_, err2 := chromedp.Call(tctx, page.Enable, page.EnableParams{})
				if err1 == nil && err2 == nil {
					escutar(tctx)
					fmt.Println("[aba] nova janela/aba observada")
				}
			}()
		}
		if e, ok := ev.(*browser.EventDownloadWillBegin); ok {
			st.mu.Lock()
			d := mask(e.SuggestedFilename)
			st.downloads = append(st.downloads, d)
			st.mu.Unlock()
			fmt.Printf("[download] iniciado e BLOQUEADO: %s\n", d)
		}
	})

	must(chromedp.Do(ctx)) // inicia o navegador
	ev := true
	_, err := chromedp.CallBrowser(ctx, browser.SetDownloadBehavior, browser.SetDownloadBehaviorParams{Behavior: browser.SetDownloadBehaviorBehaviorDeny, EventsEnabled: &ev})
	must(err)
	_, err = chromedp.Call(ctx, network.Enable, network.EnableParams{})
	must(err)
	_, err = chromedp.Call(ctx, page.Enable, page.EnableParams{})
	must(err)
	must(chromedp.Do(ctx, chromedp.Navigate(*login)))

	// Snapshots estruturais com debounce.
	go func() {
		for tctx := range snap {
			time.Sleep(900 * time.Millisecond)
			for len(snap) > 0 {
				tctx = <-snap
			}
			raw, err := chromedp.Run(tctx, chromedp.Evaluate[string](snapshotJS))
			if err != nil {
				continue
			}
			var quadros []map[string]any
			if json.Unmarshal([]byte(raw), &quadros) != nil {
				continue
			}
			for _, q := range quadros {
				b, _ := json.Marshal(q)
				s := mask(string(b))
				h := fmt.Sprintf("%x", sha1.Sum([]byte(s)))
				st.mu.Lock()
				novo := !st.vistos[h]
				st.vistos[h] = true
				st.mu.Unlock()
				if novo {
					fmt.Println("[tela]", s)
				}
			}
		}
	}()

	if *segundos > 0 {
		time.Sleep(time.Duration(*segundos) * time.Second)
	} else {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			cmd := strings.TrimSpace(strings.ToLower(sc.Text()))
			if cmd == "sair" {
				break
			}
			if cmd == "mapear" {
				st.mu.Lock()
				abas := append([]context.Context(nil), st.abas...)
				st.mu.Unlock()
				mapearGerar(abas)
				continue
			}
			if cmd == "aceitar" || cmd == "cancelar" {
				st.mu.Lock()
				d := st.dialogo
				st.dialogo = nil
				st.mu.Unlock()
				if d == nil {
					fmt.Println("[dialogo] nenhum diálogo pendente.")
					continue
				}
				responderDialogo(d, cmd == "aceitar")
				fmt.Println("[dialogo] resposta enviada:", cmd)
			}
		}
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	fmt.Println("\n========== RESUMO (sem dados do processo; pode copiar e colar) ==========")
	fmt.Println("Navegações (tipo método acao → nomes de parâmetros):")
	for _, k := range st.ordem {
		var ps []string
		for p := range st.acoes[k] {
			ps = append(ps, p)
		}
		sort.Strings(ps)
		fmt.Printf("  %s → %s\n", k, strings.Join(ps, ","))
	}
	fmt.Println("Hosts contatados:")
	var hs []string
	for h, n := range st.hosts {
		hs = append(hs, fmt.Sprintf("  %s (%d)", h, n))
	}
	sort.Strings(hs)
	fmt.Println(strings.Join(hs, "\n"))
	fmt.Println("Downloads (bloqueados):", st.downloads)
	fmt.Println("Diálogos JavaScript:", st.dialogos)
	fmt.Println("=========================================================================")
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}
