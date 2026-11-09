// Comando sys-executor: aplicativo desktop Windows (Wails v3 + Vue 3).
//
// Compilação de produção (sem console, V4 §33 / D-09.6):
//
//	GOOS=windows go build -ldflags "-H windowsgui -s -w" -o sys-executor.exe ./cmd/sys-executor
package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/DiogoBrazil/sys-executor/frontend"
	"github.com/DiogoBrazil/sys-executor/internal/app"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/pastas"
	"github.com/DiogoBrazil/sys-executor/internal/pipeline"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/sei"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

func init() {
	application.RegisterEvent[*workflow.Visao](app.EventoVisao)
	application.RegisterEvent[*app.VisaoModelos](app.EventoModelos)
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
		Extrator:     extrator,
		LocalizarOCR: func() (ocr.Motor, error) { return ocr.Localizar(componentes) },
	}
	// IA local (Fase 4): llama-server em <pasta do executável>\llama\cpu (e
	// \llama\vulkan, opcional); modelos em %LOCALAPPDATA%\sys-executor\modelos.
	modelos, err := inference.NovoGerenciadorModelos(p.Modelos())
	if err != nil {
		os.Exit(1)
	}
	motor := &app.MotorLocal{Modelos: modelos, Loja: loja, Exe: app.Executaveis{
		CPU: inference.LocalizarServidor,
		GPU: func() (string, error) { return localizarVulkan(componentesBase()) },
	}}
	pipe.IA = motor

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

	fontes := app.NovaFontesService(ger)
	servModelos := &app.ModelosService{Motor: motor,
		Emitir: func(v *app.VisaoModelos) {
			if wapp != nil {
				wapp.Event.Emit(app.EventoModelos, v)
			}
		},
		Seletor: func() (string, error) {
			d := wapp.Dialog.OpenFile().SetTitle("Selecione o arquivo do modelo (GGUF)").CanChooseFiles(true).
				AddFilter("Modelo GGUF (*.gguf)", "*.gguf")
			if janela != nil {
				d = d.AttachToWindow(janela)
			}
			return d.PromptForSingleSelection()
		}}

	dist, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		os.Exit(1)
	}

	wapp = application.New(application.Options{
		Name:        "sys-executor",
		Description: "Minutas de julgamento de PADS e de relatório de Sindicância Regular",
		// Sem logs (V4 §33; SEG-LOG-01): o logger do Wails descarta tudo.
		Logger:   slog.New(slog.DiscardHandler),
		LogLevel: slog.LevelError + 100,
		Services: []application.Service{
			application.NewService(app.NovaTarefaService(ger, seletor)),
			application.NewService(app.NovaPreferenciasService(loja)),
			application.NewService(fontes),
			application.NewService(servModelos),
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
			ger.Encerrar()
			_ = extrator.Fechar()
			faxina.Do(agendarFaxina)
		},
	})

	janela = wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "sys-executor",
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

func componentesBase() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

// localizarVulkan: SYSEXEC_LLAMA_VULKAN ou <pasta>\llama\vulkan\llama-server.exe.
// Só é usado depois da sonda segura de GPU (D-08, D-09.1).
func localizarVulkan(base string) (string, error) {
	if v := os.Getenv("SYSEXEC_LLAMA_VULKAN"); v != "" {
		return v, nil
	}
	c := filepath.Join(base, "llama", "vulkan", "llama-server.exe")
	if _, err := os.Stat(c); err != nil {
		return "", inference.ErrSemGPU
	}
	return c, nil
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
