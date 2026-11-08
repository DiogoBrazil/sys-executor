package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
)

const caminhoCatalogo = "internal/rules/catalog"

var (
	reRegra    = regexp.MustCompile(`//\s*Regra:\s*(.+)$`)
	reIDMatriz = regexp.MustCompile(`(?m)^\| ((?:PADS|SR|TRV)-[A-Z0-9]+-\d{3}) \|`)
	reModMotor = regexp.MustCompile(`\b(engine|civil)\b`)
)

// relatorioCatalogo resume a conferência do catálogo (GOV-COV-01 completo).
type relatorioCatalogo struct {
	Regras           int
	ComEnforced      int
	MotorDET         int
	TestesV4         int
	V4SemRegra       []string // ID de teste da V4 ausente do catálogo
	SoNaMatriz       []string // regra da matriz ausente do catálogo
	SoNoCatalogo     []string // regra do catálogo ausente da matriz
	SimbolosFaltam   []string // regra: símbolo inexistente
	SemTeste         []string // regra com enforced_by sem teste anotado
	MotorSemEnforce  []string // regra DET do motor sem enforced_by
	RegraInexistente []string // "// Regra:" apontando ID fora do catálogo
}

func (r relatorioCatalogo) falhas() int {
	return len(r.V4SemRegra) + len(r.SoNaMatriz) + len(r.SoNoCatalogo) + len(r.SimbolosFaltam) +
		len(r.SemTeste) + len(r.MotorSemEnforce) + len(r.RegraInexistente)
}

func verificarCatalogo(raiz string, idsV4 map[string]bool, matriz string) (relatorioCatalogo, error) {
	var rel relatorioCatalogo
	cat, err := catalog.CarregarFS(os.DirFS(filepath.Join(raiz, caminhoCatalogo)))
	if err != nil {
		return rel, err
	}
	rel.Regras = len(cat.Regras)

	// Catálogo × matriz: mesmo conjunto de regras.
	naMatriz := map[string]bool{}
	for _, m := range reIDMatriz.FindAllStringSubmatch(matriz, -1) {
		naMatriz[m[1]] = true
		if _, ok := cat.PorID(m[1]); !ok {
			rel.SoNaMatriz = append(rel.SoNaMatriz, m[1])
		}
	}
	for _, r := range cat.Regras {
		if !naMatriz[r.ID] {
			rel.SoNoCatalogo = append(rel.SoNoCatalogo, r.ID)
		}
	}

	// Todo ID de teste da V4 citado por alguma regra vigente.
	testes := cat.Testes()
	rel.TestesV4 = len(idsV4)
	for id := range idsV4 {
		if len(testes[id]) == 0 {
			rel.V4SemRegra = append(rel.V4SemRegra, id)
		}
	}

	simbolos, err := simbolosExportados(filepath.Join(raiz, "internal"))
	if err != nil {
		return rel, err
	}
	dirInternal := filepath.Join(raiz, "internal")
	cobertos, err := anotacoes(dirInternal, reCobre)
	if err != nil {
		return rel, err
	}
	regras, err := anotacoes(dirInternal, reRegra)
	if err != nil {
		return rel, err
	}
	for id := range regras {
		if _, ok := cat.PorID(id); !ok {
			rel.RegraInexistente = append(rel.RegraInexistente, id)
		}
	}

	for _, r := range cat.Regras {
		if !r.Vigente() {
			continue
		}
		if r.Tem(catalog.Deterministica) && reModMotor.MatchString(r.Modulo) {
			rel.MotorDET++
			if len(r.EnforcedBy) == 0 {
				rel.MotorSemEnforce = append(rel.MotorSemEnforce, r.ID)
			}
		}
		if len(r.EnforcedBy) == 0 {
			continue
		}
		rel.ComEnforced++
		for _, s := range r.EnforcedBy {
			if !simbolos[s] {
				rel.SimbolosFaltam = append(rel.SimbolosFaltam, r.ID+": "+s)
			}
		}
		testada := regras[r.ID]
		for _, t := range r.Testes {
			testada = testada || cobertos[t]
		}
		if !testada {
			rel.SemTeste = append(rel.SemTeste, r.ID)
		}
	}
	for _, l := range [][]string{rel.V4SemRegra, rel.SoNaMatriz, rel.SoNoCatalogo, rel.SimbolosFaltam, rel.SemTeste, rel.MotorSemEnforce, rel.RegraInexistente} {
		sort.Strings(l)
	}
	return rel, nil
}

// simbolosExportados indexa "pacote.Nome" de funções sem receptor, tipos,
// variáveis e constantes exportados dos arquivos Go (não de teste) sob dir.
func simbolosExportados(dir string) (map[string]bool, error) {
	out := map[string]bool{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		pkg := f.Name.Name
		for _, dec := range f.Decls {
			switch d := dec.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					out[pkg+"."+d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					switch s := sp.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							out[pkg+"."+s.Name.Name] = true
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								out[pkg+"."+n.Name] = true
							}
						}
					}
				}
			}
		}
		return nil
	})
	return out, err
}

// anotacoes lê as anotações (// Cobre: … ou // Regra: …) dos testes sob dir.
func anotacoes(dir string, re *regexp.Regexp) (map[string]bool, error) {
	ids := map[string]bool{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, "_test.go") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, l := range strings.Split(string(b), "\n") {
			if m := re.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
				for _, id := range expandir(m[1]) {
					ids[id] = true
				}
			}
		}
		return nil
	})
	return ids, err
}

func imprimirCatalogo(r relatorioCatalogo) {
	fmt.Printf("GOV-COV-01 (catálogo): %d regras; %d IDs de teste da V4, %d sem regra; %d regras com enforced_by, %d DET do motor.\n",
		r.Regras, r.TestesV4, len(r.V4SemRegra), r.ComEnforced, r.MotorDET)
	itens := []struct {
		titulo string
		lista  []string
	}{
		{"IDs da V4 sem regra no catálogo", r.V4SemRegra},
		{"Regras só na matriz", r.SoNaMatriz},
		{"Regras só no catálogo", r.SoNoCatalogo},
		{"enforced_by inexistente", r.SimbolosFaltam},
		{"Regras garantidas sem teste anotado (// Regra: ou // Cobre:)", r.SemTeste},
		{"Regras DET do motor sem enforced_by", r.MotorSemEnforce},
		{"// Regra: com ID fora do catálogo", r.RegraInexistente},
	}
	for _, it := range itens {
		if len(it.lista) > 0 {
			fmt.Printf("  %s (%d): %s\n", it.titulo, len(it.lista), strings.Join(it.lista, "; "))
		}
	}
}
