package session

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Varredura resume a limpeza de resíduos (sem nenhum dado das tarefas).
type Varredura struct {
	Removidas int // pastas de sessões abandonadas removidas
	Pendentes int // pastas que não puderam ser removidas agora
	Ativas    int // pastas de processos ainda vivos (preservadas)
}

// VarrerResiduos remove pastas de sessões cujo processo dono não está mais
// vivo (encerramento abrupto) ou que não têm marcador válido (queda antes de
// gravá-lo). Não lê nem restaura conteúdo: a varredura nunca é mecanismo de
// recuperação de tarefa (V4 §29.2; T12; SEG-CRASH-01).
func (g *Gerenciador) VarrerResiduos() Varredura {
	var v Varredura
	entradas, err := os.ReadDir(g.raiz)
	if err != nil {
		return v
	}
	for _, e := range entradas {
		p := filepath.Join(g.raiz, e.Name())
		if !e.IsDir() {
			// Nada além de pastas de sessão deve existir na raiz.
			sobrescrever(p)
			if os.Remove(p) != nil {
				v.Pendentes++
			}
			continue
		}
		if dono, ok := lerMarcador(p); ok && processoVivo(dono.PID, dono.Inicio) {
			v.Ativas++
			continue
		}
		if apagarPasta(p) != nil {
			v.Pendentes++
			continue
		}
		v.Removidas++
	}
	return v
}

func lerMarcador(dir string) (marcador, bool) {
	var m marcador
	b, err := os.ReadFile(filepath.Join(dir, nomeMarcador))
	if err != nil || json.Unmarshal(b, &m) != nil || m.PID <= 0 {
		return m, false
	}
	return m, true
}

// InicioProcesso identifica o instante de criação de um processo (ver marcador).
func InicioProcesso(pid int) (int64, error) { return inicioProcesso(pid) }

// ProcessoVivo informa se o processo (pid, inicio) ainda executa.
func ProcessoVivo(pid int, inicio int64) bool { return processoVivo(pid, inicio) }
