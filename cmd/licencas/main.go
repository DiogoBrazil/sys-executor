// Comando licencas: confere as licenças das dependências que entram no produto
// (plano §13 "Supply chain"; F1-07). Sem dependências externas.
//
//   - Go: módulos importados pelo executável Windows (go list -deps), com a
//     licença identificada pelo texto do arquivo LICENSE do módulo.
//   - Frontend: pacotes de produção do frontend/package-lock.json.
//
// Aprova apenas licenças permissivas conhecidas; copyleft ou desconhecida
// reprova. Uso: go run ./cmd/licencas [-raiz .] [-listar]
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var permitidas = map[string]bool{
	"MIT": true, "Apache-2.0": true, "BSD-2-Clause": true, "BSD-3-Clause": true,
	"ISC": true, "0BSD": true, "OFL-1.1": true, "BlueOak-1.0.0": true,
}

func main() {
	raiz := flag.String("raiz", ".", "raiz do repositório")
	listar := flag.Bool("listar", false, "imprimir todas as dependências e licenças (SBOM simples)")
	flag.Parse()

	var problemas []string
	goDeps, err := dependenciasGo(*raiz)
	if err != nil {
		fmt.Fprintln(os.Stderr, "licencas: go list:", err)
		os.Exit(1)
	}
	npmDeps, err := dependenciasNPM(filepath.Join(*raiz, "frontend", "package-lock.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "licencas: package-lock:", err)
		os.Exit(1)
	}
	for _, grupo := range []struct {
		nome string
		deps []dep
	}{{"go", goDeps}, {"npm", npmDeps}} {
		for _, d := range grupo.deps {
			if *listar {
				fmt.Printf("%-4s %-60s %-12s %s\n", grupo.nome, d.nome, d.versao, d.licenca)
			}
			if !aprovada(d.licenca) {
				problemas = append(problemas, fmt.Sprintf("%s %s@%s: %s", grupo.nome, d.nome, d.versao, d.licenca))
			}
		}
	}
	fmt.Printf("licenças: %d módulos Go e %d pacotes npm de produção verificados.\n", len(goDeps), len(npmDeps))
	if len(problemas) > 0 {
		sort.Strings(problemas)
		fmt.Println("Reprovadas ou não identificadas:")
		for _, p := range problemas {
			fmt.Println("  " + p)
		}
		os.Exit(1)
	}
}

type dep struct{ nome, versao, licenca string }

// aprovada aceita expressões SPDX simples com OR/AND entre licenças permitidas
// (em "OR" basta uma; em "AND" todas).
func aprovada(expr string) bool {
	expr = strings.Trim(strings.TrimSpace(expr), "()")
	if expr == "" {
		return false
	}
	if partes := strings.Split(expr, " OR "); len(partes) > 1 {
		for _, p := range partes {
			if aprovada(p) {
				return true
			}
		}
		return false
	}
	for _, p := range strings.Split(expr, " AND ") {
		if !permitidas[strings.Trim(strings.TrimSpace(p), "()")] {
			return false
		}
	}
	return true
}

func dependenciasGo(raiz string) ([]dep, error) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{with .Module}}{{if not .Main}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}{{end}}", "./cmd/sys-executor")
	cmd.Dir = raiz
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	vistos := map[string]bool{}
	var deps []dep
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) != 3 || vistos[c[0]] {
			continue
		}
		vistos[c[0]] = true
		deps = append(deps, dep{nome: c[0], versao: c[1], licenca: licencaDoModulo(c[2])})
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].nome < deps[j].nome })
	return deps, nil
}

var (
	reApache = regexp.MustCompile(`(?i)apache license,?\s+version 2\.0`)
	reMIT    = regexp.MustCompile(`(?i)permission is hereby granted, free of charge`)
	reBSD3   = regexp.MustCompile(`(?i)neither the name of .{0,120} nor the names of`)
	reBSD2   = regexp.MustCompile(`(?i)redistributions in binary form must reproduce`)
	reISC    = regexp.MustCompile(`(?i)permission to use, copy, modify, and/or distribute this software for any`)
	reGPL    = regexp.MustCompile(`(?i)gnu (affero |lesser |library )?general public license`)
	reMPL    = regexp.MustCompile(`(?i)mozilla public license`)
)

// licencaDoModulo identifica a licença pelo texto do arquivo LICENSE.
func licencaDoModulo(dir string) string {
	if dir == "" {
		return "desconhecida (módulo sem diretório)"
	}
	entradas, _ := os.ReadDir(dir)
	for _, e := range entradas {
		n := strings.ToUpper(e.Name())
		if e.IsDir() || !(strings.HasPrefix(n, "LICENSE") || strings.HasPrefix(n, "LICENCE") || strings.HasPrefix(n, "COPYING")) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		t := string(b)
		switch {
		case reGPL.MatchString(t):
			return "copyleft (GPL/LGPL/AGPL)"
		case reMPL.MatchString(t):
			return "MPL (copyleft fraco)"
		case reApache.MatchString(t):
			return "Apache-2.0"
		case reMIT.MatchString(t):
			return "MIT"
		case reBSD3.MatchString(t):
			return "BSD-3-Clause"
		case reBSD2.MatchString(t):
			return "BSD-2-Clause"
		case reISC.MatchString(t):
			return "ISC"
		}
		return "desconhecida (" + e.Name() + ")"
	}
	return "desconhecida (sem LICENSE)"
}

func dependenciasNPM(lock string) ([]dep, error) {
	b, err := os.ReadFile(lock)
	if err != nil {
		return nil, err
	}
	var l struct {
		Packages map[string]struct {
			Version string `json:"version"`
			License any    `json:"license"`
			Dev     bool   `json:"dev"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	var deps []dep
	for caminho, p := range l.Packages {
		if caminho == "" || p.Dev {
			continue
		}
		nome := caminho[strings.LastIndex(caminho, "node_modules/")+len("node_modules/"):]
		lic, _ := p.License.(string)
		deps = append(deps, dep{nome: nome, versao: p.Version, licenca: lic})
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].nome < deps[j].nome })
	return deps, nil
}
