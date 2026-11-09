package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Experimento: um pedido e o que se espera dele.
type experimento struct {
	nome     string
	pedido   Pedido
	esperado string // "ok" ou "recusa"
	conferir func(r *Resultado) error
}

const instrucaoSintetica = "Você é um assistente de testes. Todo o conteúdo é sintético e fictício. Responda em português, de forma muito curta."

func bateria(ctx context.Context, c *Credencial, modelo, dirSchemas string, so string) error {
	exps := []experimento{
		{nome: "simples", esperado: "ok",
			pedido: Pedido{Instrucoes: instrucaoSintetica, Entrada: []map[string]any{msgUsuario("Responda exatamente: ok")}},
			conferir: func(r *Resultado) error {
				if !strings.Contains(strings.ToLower(r.Texto), "ok") {
					return fmt.Errorf("texto inesperado: %q", cortar(r.Texto, 80))
				}
				return nil
			}},
		{nome: "historico_reenvio", esperado: "ok",
			pedido: Pedido{Instrucoes: instrucaoSintetica, Entrada: []map[string]any{
				msgUsuario("Responda só com a letra A."), msgAssistente("A"),
				msgUsuario("A resposta anterior foi rejeitada pelo validador. Agora responda só com a letra B.")}},
			conferir: func(r *Resultado) error {
				if !strings.Contains(r.Texto, "B") {
					return fmt.Errorf("texto inesperado: %q", cortar(r.Texto, 80))
				}
				return nil
			}},
		{nome: "recusa_role_system", esperado: "recusa",
			pedido: Pedido{Entrada: []map[string]any{{"type": "message", "role": "system", "content": "teste"}, msgUsuario("ok?")}}},
		{nome: "recusa_max_output_tokens", esperado: "recusa",
			pedido: Pedido{Instrucoes: instrucaoSintetica, Entrada: []map[string]any{msgUsuario("ok?")}, Extra: map[string]any{"max_output_tokens": 64}}},
		{nome: "recusa_temperature", esperado: "recusa",
			pedido: Pedido{Instrucoes: instrucaoSintetica, Entrada: []map[string]any{msgUsuario("ok?")}, Extra: map[string]any{"temperature": 0.1}}},
		{nome: "reasoning_low", esperado: "ok",
			pedido: Pedido{Instrucoes: instrucaoSintetica, Entrada: []map[string]any{msgUsuario("Quanto é 2+2? Só o número.")}, Raciocinio: map[string]any{"effort": "low"}}},
		{nome: "json_object", esperado: "ok",
			pedido: Pedido{Instrucoes: instrucaoSintetica + " Responda somente em JSON.", Entrada: []map[string]any{msgUsuario(`Devolva o JSON {"ok": true}.`)}, Formato: map[string]any{"type": "json_object"}},
			conferir: func(r *Resultado) error {
				if !json.Valid([]byte(r.Texto)) {
					return fmt.Errorf("JSON inválido: %q", cortar(r.Texto, 80))
				}
				return nil
			}},
	}
	schemas, err := filepath.Glob(filepath.Join(dirSchemas, "*.json"))
	if err != nil {
		return err
	}
	sort.Strings(schemas)

	tok, err := tokenValido(ctx, c)
	if err != nil {
		return err
	}
	fmt.Printf("Modelo: %s · %d experimentos básicos + %d schemas\n\n", modelo, len(exps), len(schemas))
	falhas := 0
	for _, e := range exps {
		if so != "" && !strings.Contains(e.nome, so) {
			continue
		}
		e.pedido.Modelo = modelo
		if !rodar(ctx, tok, e) {
			falhas++
		}
	}
	for _, arq := range schemas {
		nome := strings.TrimSuffix(filepath.Base(arq), ".json")
		if so != "" && !strings.Contains(nome, so) {
			continue
		}
		if !testarSchema(ctx, tok, modelo, nome, arq) {
			falhas++
		}
	}
	fmt.Printf("\n== Fim da bateria: %d resultado(s) diferente(s) do esperado ==\n", falhas)
	return nil
}

func rodar(ctx context.Context, tok string, e experimento) bool {
	r, err := gerar(ctx, tok, e.pedido, 90*time.Second)
	switch {
	case e.esperado == "recusa" && err != nil:
		fmt.Printf("[ok]     %-28s recusado como esperado: %v\n", e.nome, err)
		return true
	case e.esperado == "recusa":
		fmt.Printf("[ATENÇÃO] %-27s ACEITO (esperava recusa) status=%s\n", e.nome, r.Status)
		return false
	case err != nil:
		fmt.Printf("[FALHA]  %-28s %v\n", e.nome, err)
		if r != nil && len(r.Eventos) > 0 {
			fmt.Printf("         eventos: %s\n", resumoEventos(r.Eventos))
		}
		return false
	}
	if e.conferir != nil {
		if err := e.conferir(r); err != nil {
			fmt.Printf("[FALHA]  %-28s concluído, mas %v\n", e.nome, err)
			return false
		}
	}
	if e.nome == "simples" {
		fmt.Printf("         Content-Type=%q req=%s cabeçalhos=%v\n", r.ContentType, r.RequestID, r.Cabecalhos)
	}
	fmt.Printf("[ok]     %-28s status=%s 1º byte=%s total=%s uso=%s\n         texto=%q final==deltas:%v\n         eventos: %s\n",
		e.nome, r.Status, r.PrimeiroByte.Round(time.Millisecond), r.Duracao.Round(time.Millisecond), resumoUso(r.Uso),
		cortar(r.Texto, 60), r.TextoFinal == r.Texto, resumoEventos(r.Eventos))
	return true
}

// testarSchema: strict com o schema real; se recusado, tenta variantes
// sem palavras-chave suspeitas para descobrir o que o modo não aceita.
func testarSchema(ctx context.Context, tok, modelo, nome, arq string) bool {
	b, err := os.ReadFile(arq)
	if err != nil {
		fmt.Printf("[FALHA]  schema %-21s %v\n", nome, err)
		return false
	}
	var sch map[string]any
	if err := json.Unmarshal(b, &sch); err != nil {
		fmt.Printf("[FALHA]  schema %-21s ilegível: %v\n", nome, err)
		return false
	}
	variantes := []struct {
		rotulo  string
		remover []string
	}{
		{"original", nil},
		{"sem maxLength", []string{"maxLength"}},
		{"sem limites", []string{"maxLength", "minItems", "maxItems", "minimum", "maximum"}},
	}
	for i, v := range variantes {
		s := limpar(sch, v.remover)
		p := Pedido{Modelo: modelo, Instrucoes: instrucaoSintetica,
			Entrada: []map[string]any{msgUsuario("Produza um exemplo curto e fictício que obedeça ao formato exigido. Use listas vazias e textos curtos sempre que permitido.")},
			Formato: map[string]any{"type": "json_schema", "name": nome, "schema": s, "strict": true}}
		r, err := gerar(ctx, tok, p, 120*time.Second)
		if err != nil {
			fmt.Printf("[recusa] schema %-21s (%s): %v\n", nome, v.rotulo, err)
			var ea *ErroAPI
			if errors.As(err, &ea) && ea.Status != 400 && ea.Codigo != "subscription_sharing_unsupported_capability" && !ea.NoFluxo {
				return false // erro que não é de schema: não insiste
			}
			continue
		}
		var val any
		if err := json.Unmarshal([]byte(r.Texto), &val); err != nil {
			fmt.Printf("[FALHA]  schema %-21s (%s) aceito, mas a saída não é JSON: %q\n", nome, v.rotulo, cortar(r.Texto, 80))
			return false
		}
		viol := validar(sch, val, "$") // sempre contra o schema ORIGINAL
		situacao := "[ok]    "
		if i > 0 {
			situacao = "[parcial]"
		}
		fmt.Printf("%s schema %-21s aceito (%s) · %d violação(ões) do schema original · total=%s uso=%s\n",
			situacao, nome, v.rotulo, len(viol), r.Duracao.Round(time.Millisecond), resumoUso(r.Uso))
		for j, x := range viol {
			if j == 5 {
				fmt.Printf("         … +%d\n", len(viol)-5)
				break
			}
			fmt.Printf("         - %s\n", x)
		}
		return i == 0 && len(viol) == 0
	}
	return false
}

func limpar(v any, remover []string) any {
	switch x := v.(type) {
	case map[string]any:
		m := map[string]any{}
		for k, val := range x {
			if contem(remover, k) {
				continue
			}
			m[k] = limpar(val, remover)
		}
		return m
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = limpar(x[i], remover)
		}
		return out
	}
	return v
}

func contem(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// validar confere o subconjunto usado pelo app (analise/schema.go).
func validar(sch map[string]any, v any, caminho string) []string {
	var out []string
	if en, ok := sch["enum"].([]any); ok {
		achou := false
		for _, e := range en {
			if e == v {
				achou = true
			}
		}
		if !achou {
			out = append(out, fmt.Sprintf("%s: %v fora do enum", caminho, v))
		}
	}
	switch sch["type"] {
	case "object":
		m, ok := v.(map[string]any)
		if !ok {
			return append(out, caminho+": não é objeto")
		}
		props, _ := sch["properties"].(map[string]any)
		if req, ok := sch["required"].([]any); ok {
			for _, r := range req {
				if _, ok := m[r.(string)]; !ok {
					out = append(out, fmt.Sprintf("%s: falta %q", caminho, r))
				}
			}
		}
		for k, val := range m {
			ps, ok := props[k].(map[string]any)
			if !ok {
				out = append(out, fmt.Sprintf("%s: propriedade extra %q", caminho, k))
				continue
			}
			out = append(out, validar(ps, val, caminho+"."+k)...)
		}
	case "array":
		a, ok := v.([]any)
		if !ok {
			return append(out, caminho+": não é lista")
		}
		if n, ok := sch["minItems"].(float64); ok && float64(len(a)) < n {
			out = append(out, fmt.Sprintf("%s: %d itens < minItems %v", caminho, len(a), n))
		}
		if n, ok := sch["maxItems"].(float64); ok && float64(len(a)) > n {
			out = append(out, fmt.Sprintf("%s: %d itens > maxItems %v", caminho, len(a), n))
		}
		if it, ok := sch["items"].(map[string]any); ok {
			for i, x := range a {
				out = append(out, validar(it, x, fmt.Sprintf("%s[%d]", caminho, i))...)
			}
		}
	case "string":
		s, ok := v.(string)
		if !ok {
			return append(out, caminho+": não é texto")
		}
		if n, ok := sch["maxLength"].(float64); ok && float64(len([]rune(s))) > n {
			out = append(out, fmt.Sprintf("%s: %d caracteres > maxLength %v", caminho, len([]rune(s)), n))
		}
	case "integer":
		f, ok := v.(float64)
		if !ok || f != float64(int64(f)) {
			return append(out, caminho+": não é inteiro")
		}
		if n, ok := sch["minimum"].(float64); ok && f < n {
			out = append(out, fmt.Sprintf("%s: %v < minimum %v", caminho, f, n))
		}
		if n, ok := sch["maximum"].(float64); ok && f > n {
			out = append(out, fmt.Sprintf("%s: %v > maximum %v", caminho, f, n))
		}
	}
	return out
}

func resumoUso(u map[string]any) string {
	if u == nil {
		return "-"
	}
	s := fmt.Sprintf("in=%v out=%v", u["input_tokens"], u["output_tokens"])
	if d, ok := u["output_tokens_details"].(map[string]any); ok {
		s += fmt.Sprintf(" raciocínio=%v", d["reasoning_tokens"])
	}
	return s
}
