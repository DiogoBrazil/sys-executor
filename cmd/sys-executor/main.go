// Comando sys-executor: o aplicativo desktop Windows JURISPM — LEXIA,
// Assistente Inteligente de Análise Jurídico-Disciplinar (Wails v3 + Vue 3).
// O nome do projeto e a pasta de dados continuam "sys-executor".
//
// Compilação de produção (sem console, V4 §33 / D-09.6):
//
//	GOOS=windows go build -ldflags "-H windowsgui -s -w" -o JURISPM.exe ./cmd/sys-executor
package main

import (
	"context"
	_ "embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/DiogoBrazil/sys-executor/frontend"
	"github.com/DiogoBrazil/sys-executor/internal/app"
	"github.com/DiogoBrazil/sys-executor/internal/chatgpt"
	"github.com/DiogoBrazil/sys-executor/internal/credencial"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/pastas"
	"github.com/DiogoBrazil/sys-executor/internal/pipeline"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/sei"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// nomeApp é o nome do aplicativo na janela (decisão do usuário, 12/11/2026).
const nomeApp = "JURISPM"

// icone: escudo com a balança da justiça (icone/icone.svg).
//
//go:embed icone/icone.png
var icone []byte

func init() {
	application.RegisterEvent[*workflow.Visao](app.EventoVisao)
	application.RegisterEvent[*app.VisaoIA](app.EventoIA)
}

// faxina garante um único lançamento da limpeza do WebView2 por execução.
var faxina sync.Once

func main() {
	if executarFaxinaSeSolicitada() {
		return
	}
	p, err := pastas.Padrao()
	if err != nil {
		os.Exit(1)
	}

	// Antes de qualquer janela: dumps desligados, resíduos removidos, WebView limpo.
	excluirDoWER()
	sessoes := session.NovoGerenciador(p.Trabalho())
	sessoes.VarrerResiduos()
	session.EsvaziarPasta(p.WebView(), 0)
	_ = session.ProtegerPasta(p.WebView())

	loja, _ := prefs.Abrir(p.Preferencias()) // falha de gravação: segue com o padrão em memória

	// Ingestão (Fase 2): pdfium em WebAssembly e Tesseract empacotado em
	// <pasta do executável>\componentes\tesseract (SYSEXEC_TESSERACT em desenvolvimento).
	extrator := pdf.NovoExtrator()
	componentes := ""
	if exe, err := os.Executable(); err == nil {
		componentes = filepath.Join(filepath.Dir(exe), "componentes")
	}
	pipe := &pipeline.Pipeline{
		Extrator:    extrator,
		Componentes: componentes, // Typst (PDF da minuta) e logotipo do cabeçalho
		LocalizarOCR: func() (ocr.Motor, error) {
			// Sem Tesseract, devolver a interface nula de verdade: um
			// *Tesseract nulo dentro de ocr.Motor não é nil e derrubava o app.
			t, err := ocr.Localizar(componentes)
			if err != nil {
				return nil, err
			}
			return t, nil
		},
	}
	// IA (V4 §§47, 48): provedor escolhido pelo usuário nas preferências, sem
	// troca automática. Gemini API com a chave do usuário, cifrada com DPAPI em
	// %LOCALAPPDATA%\sys-executor\credenciais. ChatGPT: etapa 4 do plano D-11.
	// Sem inferência local e sem fallback local (D-10).
	fonteIA := &app.FonteGemini{Credenciais: credencial.NovaLojaArquivo(p.Credenciais()), Loja: loja}
	// ChatGPT (V4 §48.3): conta conectada pelo Sign in with ChatGPT; a
	// credencial fica cifrada com DPAPI em credenciais\chatgpt.dpapi, com
	// entropia própria; o login abre o navegador padrão do sistema (nunca a
	// WebView do aplicativo).
	var wappRef *application.App
	contaChatGPT := chatgpt.Nova(chatgpt.Config{
		Loja: credencial.NovaLojaNomeada(p.Credenciais(), chatgpt.ArquivoCredencial, chatgpt.EntropiaCredencial),
		AbrirURL: func(u string) error {
			if wappRef == nil {
				return errors.New("aplicativo ainda não iniciado")
			}
			return wappRef.Browser.OpenURL(u)
		},
	})
	fonteChatGPT := &app.FonteChatGPT{Conta: contaChatGPT, Loja: loja}
	pipe.IA = &app.FonteIA{Loja: loja, Gemini: fonteIA, ChatGPT: fonteChatGPT}

	// SEI/RO pelo Microsoft Edge instalado (Fase 2). Sem Edge, só o envio do PDF.
	navegador, errNav := sei.LocalizarNavegador()
	if errNav == nil {
		cfgSEI := configSEI(navegador)
		pipe.SEI = func(ctx context.Context, pasta string, in sei.Interlocutor) ([]byte, error) {
			c := cfgSEI
			c.PastaTrabalho = pasta
			return sei.ObterPDF(ctx, c, in)
		}
	}

	var wapp *application.App
	ger := workflow.Novo(workflow.Opcoes{
		Sessoes: sessoes,
		Etapas:  pipe.Etapas(),
		Notificar: func(v *workflow.Visao) {
			if wapp != nil {
				wapp.Event.Emit(app.EventoVisao, v)
			}
		},
		SEIDisponivel: pipe.SEI != nil,
		// Cadastro de peças (V4 §48.4): cada arquivo é aberto no pdfium para
		// conferir que é legível e contar as páginas.
		ContarPaginas: func(b []byte) (int, error) { return pdf.ContarPaginas(context.Background(), extrator, b) },
	})

	var janela *application.WebviewWindow
	seletor := func() (string, error) {
		d := wapp.Dialog.OpenFile().
			SetTitle("Selecione o PDF completo do processo").
			CanChooseFiles(true).
			AddFilter("PDF do processo (*.pdf)", "*.pdf")
		if janela != nil {
			d = d.AttachToWindow(janela)
		}
		return d.PromptForSingleSelection()
	}

	seletorVarios := func() ([]string, error) {
		d := wapp.Dialog.OpenFile().
			SetTitle("Selecione os arquivos PDF desta peça").
			CanChooseFiles(true).
			AddFilter("Documentos PDF (*.pdf)", "*.pdf")
		if janela != nil {
			d = d.AttachToWindow(janela)
		}
		return d.PromptForMultipleSelection()
	}
	servTarefa := app.NovaTarefaService(ger, seletor)
	servTarefa.SelecionarArquivos = seletorVarios
	// Minuta em revisão (V4 §31.3): o arquivo é gerado e gravado onde o
	// usuário escolher; o aplicativo não guarda cópia nem o caminho.
	servTarefa.Documento = pipe.GerarDocumento
	servTarefa.SelecionarDestino = func(nome, formato string) (string, error) {
		filtro := "Documento do Word (*.docx)"
		if formato == "pdf" {
			filtro = "Documento PDF (*.pdf)"
		}
		d := wapp.Dialog.SaveFile().
			SetMessage("Salvar a minuta").
			SetFilename(nome).
			CanCreateDirectories(true).
			AddFilter(filtro, "*."+formato)
		if janela != nil {
			d = d.AttachToWindow(janela)
		}
		return d.PromptForSingleSelection()
	}

	fontes := app.NovaFontesService(ger)
	servIA := &app.IAService{Fonte: fonteIA, ChatGPT: fonteChatGPT,
		Emitir: func(v *app.VisaoIA) {
			if wapp != nil {
				wapp.Event.Emit(app.EventoIA, v)
			}
		},
		AbrirURL: func(u string) error { return wapp.Browser.OpenURL(u) },
	}

	dist, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		os.Exit(1)
	}

	wapp = application.New(application.Options{
		Name:        nomeApp,
		Description: "LEXIA — Assistente Inteligente de Análise Jurídico-Disciplinar",
		// Ícone da janela e da barra de tarefas (o Wails o usa quando o .exe
		// não traz o recurso de ícone nº 3).
		Icon: icone,
		// Sem logs (V4 §33; SEG-LOG-01): o logger do Wails descarta tudo.
		Logger:   slog.New(slog.DiscardHandler),
		LogLevel: slog.LevelError + 100,
		Services: []application.Service{
			application.NewService(servTarefa),
			application.NewService(app.NovaPreferenciasService(loja)),
			application.NewService(fontes),
			application.NewService(servIA),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(dist),
			Middleware: fontes.Middleware, // PDF dos autos, sem cache (SEG-CACHE-01)
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "sys-executor.desktop",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if janela != nil {
					janela.Restore()
					janela.Focus()
				}
			},
		},
		Windows: application.WindowsOptions{
			// Dados do WebView2 numa pasta própria, restrita ao usuário e
			// esvaziada na abertura e no encerramento (SEG-LOG-01, SEG-CACHE-01).
			WebviewUserDataPath:   p.WebView(),
			AdditionalBrowserArgs: []string{"--disable-crash-reporter"},
		},
		// Fechar o app descarta a tarefa em andamento (V4 §29.1: sem retomada).
		OnShutdown: func() {
			servIA.CancelarEntradaChatGPT() // fecha o servidor de retorno do login, se aberto
			ger.Encerrar()
			_ = extrator.Fechar()
			faxina.Do(agendarFaxina)
		},
	})
	wappRef = wapp

	janela = wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            nomeApp,
		Width:            1180,
		Height:           780,
		MinWidth:         900,
		MinHeight:        620,
		BackgroundColour: application.NewRGB(238, 241, 236),
		URL:              "/",
	})

	err = wapp.Run()
	faxina.Do(agendarFaxina)
	if err != nil {
		os.Exit(1)
	}
}

// configSEI monta a configuração do cliente SEI. SYSEXEC_SEI_URL permite
// apontar para o simulador local de testes (só http://127.0.0.1); em
// produção vale o endereço oficial (https). SYSEXEC_SEI_VISIVEL=1 mostra a
// janela do navegador (diagnóstico).
func configSEI(navegador string) sei.Config {
	c := sei.Config{Navegador: navegador, URLLogin: sei.URLLoginPadrao, Visivel: os.Getenv("SYSEXEC_SEI_VISIVEL") == "1"}
	if u, err := url.Parse(os.Getenv("SYSEXEC_SEI_URL")); err == nil && u.Host != "" {
		switch {
		case u.Scheme == "https":
			c.URLLogin = u.String()
		case u.Scheme == "http" && u.Hostname() == "127.0.0.1":
			c.URLLogin, c.PermitirHTTP = u.String(), true
		}
	}
	return c
}
