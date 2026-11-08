package main

import (
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/pastas"
	"github.com/DiogoBrazil/sys-executor/internal/session"
)

// Os processos do WebView2 só terminam depois que o processo do app termina;
// por isso a pasta de dados do WebView2 não pode ser esvaziada de dentro do
// app. No encerramento, o app relança o próprio executável em modo "faxina":
// a cópia espera o original terminar, esvazia a pasta e sai (SEG-LOG-01).
// A pasta é sempre derivada de pastas.Padrao(): nenhum caminho vem por argumento.
const argFaxina = "--faxina-webview"

// executarFaxinaSeSolicitada trata o modo faxina; devolve false no uso normal.
func executarFaxinaSeSolicitada() bool {
	if len(os.Args) != 4 || os.Args[1] != argFaxina {
		return false
	}
	pid, err1 := strconv.Atoi(os.Args[2])
	inicio, err2 := strconv.ParseInt(os.Args[3], 10, 64)
	p, err3 := pastas.Padrao()
	if err1 != nil || err2 != nil || err3 != nil {
		os.Exit(2)
	}
	for limite := time.Now().Add(2 * time.Minute); session.ProcessoVivo(pid, inicio) && time.Now().Before(limite); {
		time.Sleep(200 * time.Millisecond)
	}
	session.EsvaziarPasta(p.WebView(), 30*time.Second)
	os.Exit(0)
	return true
}

// agendarFaxina lança a cópia em modo faxina (melhor esforço; se falhar, a
// próxima abertura esvazia a pasta).
func agendarFaxina() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	pid := os.Getpid()
	inicio, err := session.InicioProcesso(pid)
	if err != nil {
		return
	}
	cmd := exec.Command(exe, argFaxina, strconv.Itoa(pid), strconv.FormatInt(inicio, 10))
	prepararFaxina(cmd)
	if cmd.Start() == nil {
		_ = cmd.Process.Release()
	}
}
