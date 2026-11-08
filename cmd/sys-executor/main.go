// Comando sys-executor: aplicativo desktop Windows (Wails v3 + Vue 3).
//
// Compilação de produção (sem console, V4 §33 / D-09.6):
//
//	GOOS=windows go build -ldflags "-H windowsgui -s -w" -o sys-executor.exe ./cmd/sys-executor
package main

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/DiogoBrazil/sys-executor/frontend"
	"github.com/DiogoBrazil/sys-executor/internal/app"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/pastas"
	"github.com/DiogoBrazil/sys-executor/internal/pipeline"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

func init() {
	application.RegisterEvent[*workflow.Visao](app.EventoVisao)
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

	var wapp *application.App
	ger := workflow.Novo(workflow.Opcoes{
		Sessoes: sessoes,
		Etapas:  pipe.Etapas(),
		Notificar: func(v *workflow.Visao) {
			if wapp != nil {
				wapp.Event.Emit(app.EventoVisao, v)
			}
		},
		SEIDisponivel: false, // Fase 2
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
