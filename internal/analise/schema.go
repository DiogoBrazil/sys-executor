package analise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Construtores de JSON Schema (subconjunto aceito pela gramática do
// llama-server: object/array/string/integer, enum, required,
// additionalProperties=false, minItems/maxItems, maxLength, minimum/maximum).
// Os enums de IDs válidos são injetados por chamada (estratégia §2.4).

func objeto(props map[string]any) map[string]any {
	req := make([]string, 0, len(props))
	for _, k := range ordemChaves(props) {
		req = append(req, k)
	}
	return map[string]any{"type": "object", "additionalProperties": false, "required": req, "properties": props}
}

func lista(item any, min, max int) map[string]any {
	s := map[string]any{"type": "array", "items": item}
	if min > 0 {
		s["minItems"] = min
	}
	if max > 0 {
		s["maxItems"] = max
	}
	return s
}

func str(max int) map[string]any {
	s := map[string]any{"type": "string"}
	if max > 0 {
		s["maxLength"] = max
	}
	return s
}

func inteiro(min, max int) map[string]any {
	return map[string]any{"type": "integer", "minimum": min, "maximum": max}
}

func enum(vals ...string) map[string]any {
	if len(vals) == 0 {
		vals = []string{""} // schema válido mesmo sem opções (nenhum ID disponível)
	}
	return map[string]any{"type": "string", "enum": vals}
}

// ordemChaves devolve as chaves em ordem estável (o schema entra no hash e
// nos testes).
func ordemChaves(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	for i := 1; i < len(ks); i++ {
		for j := i; j > 0 && ks[j] < ks[j-1]; j-- {
			ks[j], ks[j-1] = ks[j-1], ks[j]
		}
	}
	return ks
}

// ConferirSchema confere a resposta contra o schema da chamada no Go, sem
// depender de o provedor ter imposto o schema (a Gemini API, por exemplo,
// recusa maxItems/minItems em schemas maiores e recebe o schema sem eles).
// Cobre o subconjunto usado aqui: type, properties, required,
// additionalProperties=false, items, minItems/maxItems, maxLength,
// minimum/maximum e enum. Devolve até maxViolacoes violações.
func ConferirSchema(sch map[string]any, resposta []byte) error {
	var v any
	if err := json.Unmarshal(bytes.TrimSpace(resposta), &v); err != nil {
		return fmt.Errorf("JSON inválido: %v", err)
	}
	var out []string
	violacoesSchema(sch, v, "$", &out)
	if len(out) == 0 {
		return nil
	}
	if len(out) > maxViolacoes {
		out = append(out[:maxViolacoes], fmt.Sprintf("e mais %d", len(out)-maxViolacoes))
	}
	return fmt.Errorf("a resposta não obedece ao schema: %s", strings.Join(out, "; "))
}

const maxViolacoes = 6

func numero(x any) (float64, bool) {
	switch n := x.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

func violacoesSchema(sch map[string]any, v any, caminho string, out *[]string) {
	if vals, ok := sch["enum"].([]string); ok {
		s, _ := v.(string)
		achou := false
		for _, e := range vals {
			if e == s {
				achou = true
			}
		}
		if !achou {
			*out = append(*out, fmt.Sprintf("%s: valor %q fora das opções permitidas", caminho, s))
			return
		}
	}
	switch sch["type"] {
	case "object":
		m, ok := v.(map[string]any)
		if !ok {
			*out = append(*out, caminho+": deveria ser objeto")
			return
		}
		props, _ := sch["properties"].(map[string]any)
		if req, ok := sch["required"].([]string); ok {
			for _, r := range req {
				if _, tem := m[r]; !tem {
					*out = append(*out, fmt.Sprintf("%s: falta o campo %q", caminho, r))
				}
			}
		}
		for _, k := range ordemChaves(m) {
			ps, ok := props[k].(map[string]any)
			if !ok {
				*out = append(*out, fmt.Sprintf("%s: campo %q não existe no schema", caminho, k))
				continue
			}
			violacoesSchema(ps, m[k], caminho+"."+k, out)
		}
	case "array":
		a, ok := v.([]any)
		if !ok {
			*out = append(*out, caminho+": deveria ser lista")
			return
		}
		if n, ok := numero(sch["minItems"]); ok && float64(len(a)) < n {
			*out = append(*out, fmt.Sprintf("%s: %d item(ns), mínimo %v", caminho, len(a), n))
		}
		if n, ok := numero(sch["maxItems"]); ok && float64(len(a)) > n {
			*out = append(*out, fmt.Sprintf("%s: %d itens, máximo %v", caminho, len(a), n))
		}
		if it, ok := sch["items"].(map[string]any); ok {
			for i, x := range a {
				violacoesSchema(it, x, fmt.Sprintf("%s[%d]", caminho, i), out)
			}
		}
	case "string":
		s, ok := v.(string)
		if !ok {
			*out = append(*out, caminho+": deveria ser texto")
			return
		}
		if n, ok := numero(sch["maxLength"]); ok && float64(utf8.RuneCountInString(s)) > n {
			*out = append(*out, fmt.Sprintf("%s: %d caracteres, máximo %v", caminho, utf8.RuneCountInString(s), n))
		}
	case "integer":
		f, ok := v.(float64)
		if !ok || f != float64(int64(f)) {
			*out = append(*out, caminho+": deveria ser número inteiro")
			return
		}
		if n, ok := numero(sch["minimum"]); ok && f < n {
			*out = append(*out, fmt.Sprintf("%s: %v, mínimo %v", caminho, f, n))
		}
		if n, ok := numero(sch["maximum"]); ok && f > n {
			*out = append(*out, fmt.Sprintf("%s: %v, máximo %v", caminho, f, n))
		}
	}
}
