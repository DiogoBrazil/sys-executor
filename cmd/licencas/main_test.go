package main

import "testing"

func TestAprovada(t *testing.T) {
	for expr, esperado := range map[string]bool{
		"MIT": true, "(MIT OR Apache-2.0)": true, "MIT AND ISC": true, "OFL-1.1": true,
		"GPL-3.0": false, "MIT AND GPL-2.0": false, "LGPL-2.1 OR GPL-2.0": false, "": false,
		"copyleft (GPL/LGPL/AGPL)": false, "desconhecida (sem LICENSE)": false,
	} {
		if aprovada(expr) != esperado {
			t.Errorf("%q: esperado %v", expr, esperado)
		}
	}
}
