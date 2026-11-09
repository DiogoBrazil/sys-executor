// PoC isolada do Sign in with ChatGPT (etapa 1 do plano D-11).
// Só dados sintéticos. Nunca imprime tokens.
//
//	go run . login      [-porta N] [-nao-abrir] [-consentimento]
//	go run . relogin    [-porta N] [-nao-abrir]
//	go run . status
//	go run . renovar
//	go run . modelos
//	go run . bateria    -modelo SLUG [-schemas DIR] [-so FILTRO]
//	go run . reuso      (reapresenta um refresh já usado; PODE revogar a sessão)
//	go run . revogar
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		uso()
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	porta := fs.Int("porta", 0, "porta do retorno em 127.0.0.1 (0 = aleatória)")
	naoAbrir := fs.Bool("nao-abrir", false, "não abrir o navegador automaticamente")
	consent := fs.Bool("consentimento", false, "pedir consentimento de novo (prompt=consent)")
	ident := fs.Bool("so-identidade", false, "pedir só escopos de identidade (sem uso do plano)")
	modelo := fs.String("modelo", "", "slug do modelo (bateria)")
	schemas := fs.String("schemas", "schemas", "pasta com os schemas reais")
	so := fs.String("so", "", "rodar só experimentos cujo nome contém este texto")
	_ = fs.Parse(os.Args[2:])

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt)
	defer parar()
	c, err := lerCredencial()
	falhar(err)
	if p, err := caminhoCredencial(); err == nil {
		fmt.Println("Credencial da PoC:", p)
	}

	switch cmd {
	case "login":
		falhar(login(ctx, c, *porta, !*naoAbrir, false, *consent, *ident))
	case "relogin":
		falhar(login(ctx, c, *porta, !*naoAbrir, true, false, false))
	case "status":
		fmt.Printf("host_id=%s\nclient_id=%s\nconta=%s sub=%s…\nescopos=%v\npermissão do plano=%v\naccess_token: presente=%v expira=%s (em %s)\nrefresh_token presente=%v\nearliest_refresh_at=%s\n",
			c.HostID, c.ClientID, mascararEmail(c.Email), cortar(c.Sub, 8), c.Escopos, c.temEscopo(escopoPlano),
			c.AccessToken != "", c.Expira.Format(time.RFC3339), time.Until(c.Expira).Round(time.Second), c.RefreshToken != "", c.CedoRenovar.Format(time.RFC3339))
	case "renovar":
		falhar(renovar(ctx, c))
	case "modelos":
		tok, err := tokenValido(ctx, c)
		falhar(err)
		ms, campos, err := listarModelos(ctx, tok)
		falhar(err)
		fmt.Printf("campos do 1º item: %v\n%d modelo(s), na ordem do servidor:\n", campos, len(ms))
		for _, m := range ms {
			marca := " "
			if m.Visibility == "list" {
				marca = "*"
			}
			fmt.Printf(" %s %-18s vis=%-4s ctx=%s max=%s api=%s entradas=%s\n     raciocínio=%s padrão=%s planos=%s\n",
				marca, m.Slug, m.Visibility, m.ContextWindow, m.MaxContext, m.NaAPI, m.Entradas, compacto(m.Raciocinio), m.RacPadrao, m.Planos)
		}
		fmt.Println("(* = visibility \"list\": aparece na escolha do usuário)")
	case "bateria":
		if *modelo == "" {
			falhar(errors.New("informe -modelo SLUG (veja 'modelos')"))
		}
		falhar(bateria(ctx, c, *modelo, *schemas, *so))
	case "reuso":
		falhar(testarReuso(ctx, c))
	case "revogar":
		falhar(revogar(ctx, c))
		fmt.Println("credencial local removida")
	default:
		uso()
	}
}

// testarReuso: renova e depois reapresenta o refresh ANTIGO. Espera-se
// refresh_token_reused; pode invalidar toda a sessão (refaça o login depois).
func testarReuso(ctx context.Context, c *Credencial) error {
	antigo := c.RefreshToken
	if err := renovar(ctx, c); err != nil {
		return err
	}
	atual := *c
	tmp := *c
	tmp.RefreshToken = antigo
	// não grava: usa cópia em memória
	errReuso := renovarSemGravar(ctx, &tmp)
	fmt.Println("reapresentação do refresh antigo:", errReuso)
	// verifica se o refresh atual ainda vale
	errAtual := renovar(ctx, &atual)
	fmt.Println("refresh atual depois do reuso:", map[bool]string{true: "VÁLIDO", false: fmt.Sprint("inválido: ", errAtual)}[errAtual == nil])
	*c = atual
	return nil
}

func falhar(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERRO:", err)
		os.Exit(1)
	}
}

func uso() {
	fmt.Fprintln(os.Stderr, "uso: go run . {login|relogin|status|renovar|modelos|bateria -modelo SLUG|reuso|revogar}")
	os.Exit(2)
}

// compacto resume supported_reasoning_levels (lista de objetos ou de textos).
func compacto(raw []byte) string {
	var objs []map[string]any
	if json.Unmarshal(raw, &objs) == nil && len(objs) > 0 {
		var p []string
		for _, o := range objs {
			p = append(p, fmt.Sprint(o["effort"]))
		}
		return fmt.Sprint(p)
	}
	return string(raw)
}
