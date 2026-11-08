package main

import (
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type GreetService struct{}

func (g *GreetService) Greet(name string) string {
	return "Olá, " + name + "!"
}

// Report recebe do frontend o resultado do autoteste (binding + evento) e encerra o app
// quando SYSEXEC_SELFTEST=1.
func (g *GreetService) Report(greet, evento string, msDesdeBoot int) {
	fmt.Printf("SELFTEST binding=%q evento=%q front_pronto_em=%dms go_uptime=%v\n", greet, evento, msDesdeBoot, time.Since(inicio).Round(time.Millisecond))
	if os.Getenv("SYSEXEC_SELFTEST") == "1" {
		go func() { time.Sleep(200 * time.Millisecond); application.Get().Quit() }()
	}
}

var inicio = time.Now()
