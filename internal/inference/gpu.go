package inference

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// GPU segura (D-08, D-09.1): nada aqui roda sozinho. A listagem e a sonda
// só são chamadas por ação explícita do usuário em "avançado", com aviso, e
// a sonda é protegida por uma sentinela gravada antes (gerida pelo app).

var reDispositivo = regexp.MustCompile(`^\s*([A-Za-z]+)(\d+):\s*(.+?)\s*\((\d+) MiB,\s*(\d+) MiB free\)`)

// ListarDispositivos pergunta ao llama-server (build com GPU) os dispositivos
// utilizáveis (`--list-devices`).
func ListarDispositivos(ctx context.Context, exe string) ([]GPU, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "--list-devices")
	prepararProcesso(cmd)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return nil, erros.Com(erros.ModeloGPUInstavel, err)
	}
	return lerDispositivos(out.Bytes()), nil
}

func lerDispositivos(b []byte) []GPU {
	var gs []GPU
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		m := reDispositivo.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		tot, _ := strconv.ParseUint(m[4], 10, 64)
		livre, _ := strconv.ParseUint(m[5], 10, 64)
		gs = append(gs, GPU{Backend: m[1], Nome: m[3], VRAMTotal: tot * mib, VRAMLivre: livre * mib})
	}
	return gs
}

// SondarGPU: carga curta com poucas camadas na GPU e 16 tokens, com tempo
// limite. Falha → a GPU não é usada (CPU segue).
func SondarGPU(ctx context.Context, exe, modelo string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	s, err := IniciarServidor(ctx, ConfigServidor{Executavel: exe, Modelo: modelo, Contexto: 512, CamadasGPU: 4,
		TimeoutCarga: 90 * time.Second, TimeoutPedido: 30 * time.Second})
	if err != nil {
		return erros.Com(erros.ModeloGPUInstavel, err)
	}
	defer s.Fechar()
	if _, err := s.Gerar(ctx, Pedido{Usuario: "Responda apenas: ok", MaxTokens: 16}); err != nil {
		return erros.Com(erros.ModeloGPUInstavel, err)
	}
	return nil
}

// ErrSemGPU: não há build com GPU instalado.
var ErrSemGPU = errors.New("componente de GPU não instalado")
