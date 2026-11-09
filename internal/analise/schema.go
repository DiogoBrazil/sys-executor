package analise

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
