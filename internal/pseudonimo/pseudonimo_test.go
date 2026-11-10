package pseudonimo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
)

// Autos sintéticos (nomes, números e endereços fictícios).
const autosSinteticos = `POLÍCIA MILITAR DO ESTADO DE RONDÔNIA
CORREGEDORIA GERAL
MEMORANDO DISCIPLINAR Nº 12/2026
Processo SEI nº 0021.123456/2026-12
O Sd PM 100012345 JOÃO CARLOS DA SILVA, matrícula 100012345, é acusado de faltar ao serviço em 07/11/2026.
Testemunha: Maria Aparecida Souza, CPF 529.982.247-25, RG nº 1234567 SSP/RO, residente na Rua das Acácias, nº 45, Bairro Centro, CEP 76801-123, telefone (69) 99999-1234, e-mail maria.souza@exemplo.com.
A declarante, nascida em 02/03/1990, filha de José Pereira Souza e de Ana Lúcia Souza, disse que viu o Cb PM Pedro Henrique Alves no local.
O advogado Dr. Roberto Nunes Lima, OAB/RO nº 4567, apresentou defesa prévia.
Conforme o art. 39, inciso III, do RDPM, a natureza é GRAVE. Documento SEI 0012345678.
Ocorrência na Avenida Jorge Teixeira, em Porto Velho.

JOSÉ ANTÔNIO FERREIRA
Cap PM - Encarregado
`

func formas(m *Mapa) map[string]string { return m.marcador }

// Regra: TRV-SEG-011
func TestDetectaComAncoras(t *testing.T) {
	m := NovoMapa()
	m.Aprender(autosSinteticos)
	f := formas(m)
	devem := []string{
		"JOÃO CARLOS DA SILVA", "100012345", "Maria Aparecida Souza", "529.982.247-25", "1234567",
		"Rua das Acácias, nº 45, Bairro Centro", "76801-123", "(69) 99999-1234", "maria.souza@exemplo.com",
		"02/03/1990", "José Pereira Souza", "Ana Lúcia Souza", "Pedro Henrique Alves", "Roberto Nunes Lima", "4567",
		"JOSÉ ANTÔNIO FERREIRA",
	}
	for _, d := range devem {
		if _, ok := f[d]; !ok {
			t.Errorf("não detectou %q", d)
		}
	}
	naoDevem := []string{"POLÍCIA MILITAR", "RONDÔNIA", "CORREGEDORIA GERAL", "0021.123456/2026-12", "0012345678",
		"07/11/2026", "Avenida Jorge Teixeira", "Porto Velho", "GRAVE", "RDPM", "Encarregado"}
	for k := range f {
		for _, n := range naoDevem {
			if strings.Contains(k, n) {
				t.Errorf("não deveria ocultar %q (forma %q)", n, k)
			}
		}
	}
	oc := m.Ocultar(autosSinteticos)
	for _, d := range devem {
		if strings.Contains(oc, d) && d != "1234567" && d != "4567" { // ambos aparecem dentro do nº SEI 0012345678
			t.Errorf("%q continua no texto enviado", d)
		}
	}
	for _, d := range []string{"RG nº 1234567", "OAB/RO nº 4567", "matrícula 100012345", "Sd PM 100012345"} {
		if strings.Contains(oc, d) {
			t.Errorf("%q continua no texto enviado", d)
		}
	}
	for _, n := range []string{"Sd PM", "Cb PM", "Cap PM", "0021.123456/2026-12", "0012345678", "07/11/2026", "art. 39, inciso III"} {
		if !strings.Contains(oc, n) {
			t.Errorf("%q deveria permanecer (posto, número SEI, data do fato, dispositivo)", n)
		}
	}
	r := m.Resumir()
	if r.PorTipo[CPF] != 1 || r.PorTipo[Pessoa] < 7 || !strings.Contains(r.Texto(), "pessoas") {
		t.Fatalf("resumo %+v %q", r, r.Texto())
	}
}

func TestCPFInvalidoSemRotuloNaoEhCPF(t *testing.T) {
	m := NovoMapa()
	m.Aprender("Número 123.456.789-00 e protocolo 111.111.111-11.")
	if !m.Vazio() {
		t.Fatalf("dígitos inválidos sem rótulo não são CPF: %v", formas(m))
	}
	if !cpfValido("529.982.247-25") || cpfValido("529.982.247-26") {
		t.Fatal("dígito verificador")
	}
}

func TestIdaEVoltaByteAByte(t *testing.T) {
	m := NovoMapa()
	m.Aprender(autosSinteticos)
	oc := m.Ocultar(autosSinteticos)
	de, err := m.Restaurar(oc)
	if err != nil || de != autosSinteticos {
		t.Fatalf("ida e volta divergiu: %v", err)
	}
}

func TestMesmaPessoaCaixaDiferente(t *testing.T) {
	m := NovoMapa()
	m.Aprender("O Sd PM João Carlos da Silva faltou. Assinatura:\nJOÃO CARLOS DA SILVA\nSd PM")
	a, b := m.marcador["João Carlos da Silva"], m.marcador["JOÃO CARLOS DA SILVA"]
	if a == "" || b == "" || a == b || !strings.HasPrefix(b, strings.TrimSuffix(a, "]")+".") {
		t.Fatalf("grafias da mesma pessoa devem compartilhar o número: %q %q", a, b)
	}
}

func TestNaoTrocaDentroDePalavra(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Silva"})
	if oc := m.Ocultar("Silvana e Silva."); oc != "Silvana e [PESSOA-01]." {
		t.Fatalf("limite de palavra: %q", oc)
	}
}

func TestRestaurarJSONEscapa(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{`Ana "Nina" Reis`})
	b, err := m.RestaurarJSON([]byte(`{"quem":"[PESSOA-01]"}`))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]string
	if err := json.Unmarshal(b, &v); err != nil || v["quem"] != `Ana "Nina" Reis` {
		t.Fatalf("JSON restaurado: %s %v", b, err)
	}
}

func TestMarcadorDesconhecidoOuDeformado(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Maria Souza"})
	for _, s := range []string{`{"q":"[PESSOA-07]"}`, `{"q":"PESSOA-01"}`, `{"q":"[PESSOA 01]"}`, `{"q":"[CPF-1"}`} {
		if _, err := m.RestaurarJSON([]byte(s)); !errors.As(err, new(ErrMarcador)) {
			t.Errorf("%s deveria ser rejeitado: %v", s, err)
		}
	}
}

// motorEco devolve o que o roteiro mandar e guarda o pedido recebido.
type motorEco struct {
	pedidos   []inference.Pedido
	respostas []string
}

func (e *motorEco) Gerar(_ context.Context, p inference.Pedido) (inference.Resposta, error) {
	e.pedidos = append(e.pedidos, p)
	i := len(e.pedidos) - 1
	if i >= len(e.respostas) {
		i = len(e.respostas) - 1
	}
	return inference.Resposta{Conteudo: []byte(e.respostas[i])}, nil
}
func (e *motorEco) ContarTokens(_ context.Context, t string) (int, error) { return len(t) / 3, nil }
func (e *motorEco) Contexto() int                                         { return 8192 }

// Regra: TRV-SEG-011
func TestMotorOcultaNaIdaERestauraNaVolta(t *testing.T) {
	m := NovoMapa()
	m.Aprender(autosSinteticos)
	m.Liberar("Jorge Teixeira") // decisão do usuário na revisão: nome da avenida, não dado pessoal
	mk := m.marcador["Maria Aparecida Souza"]
	eco := &motorEco{respostas: []string{`{"trecho_literal":"Testemunha: ` + mk + `, CPF [CPF-01]","quem":"` + mk + `"}`}}
	p := Envolver(eco, m)
	r, err := p.Gerar(context.Background(), inference.Pedido{Sistema: "REGRAS", Usuario: autosSinteticos,
		Historico: []inference.Mensagem{{Papel: "assistant", Conteudo: `{"quem":"Maria Aparecida Souza"}`}}})
	if err != nil {
		t.Fatal(err)
	}
	enviado := eco.pedidos[0]
	for _, sentinela := range []string{"Maria Aparecida Souza", "529.982.247-25", "JOÃO CARLOS DA SILVA", "maria.souza@exemplo.com"} {
		if strings.Contains(enviado.Usuario, sentinela) || strings.Contains(enviado.Historico[0].Conteudo, sentinela) {
			t.Fatalf("%q foi enviado à IA", sentinela)
		}
	}
	// A IA não pode comentar os marcadores na redação (visto no teste real de
	// 09/11/2026: "O marcador X identifica a mesma pessoa que Y").
	if !strings.Contains(enviado.Sistema, "não mencione a existência de marcadores") {
		t.Fatal("instrução contra comentar marcadores ausente")
	}
	if !strings.HasPrefix(enviado.Sistema, "REGRAS") || !strings.Contains(enviado.Sistema, "MARCADORES DE DADOS PESSOAIS") {
		t.Fatal("instrução dos marcadores ausente ou regras alteradas")
	}
	var v map[string]string
	if err := json.Unmarshal(r.Conteudo, &v); err != nil {
		t.Fatal(err)
	}
	if v["trecho_literal"] != "Testemunha: Maria Aparecida Souza, CPF 529.982.247-25" || v["quem"] != "Maria Aparecida Souza" {
		t.Fatalf("restauração: %+v", v)
	}
	// O trecho restaurado existe literalmente nos autos (conferência literal intacta).
	if !strings.Contains(autosSinteticos, v["trecho_literal"]) {
		t.Fatal("trecho restaurado não confere com os autos")
	}
}

func TestMotorPedeCorrecaoDeMarcador(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Maria Souza"})
	eco := &motorEco{respostas: []string{`{"q":"PESSOA-01"}`, `{"q":"[PESSOA-01]"}`}}
	r, err := Envolver(eco, m).Gerar(context.Background(), inference.Pedido{Usuario: "Maria Souza"})
	if err != nil || string(r.Conteudo) != `{"q":"Maria Souza"}` {
		t.Fatalf("correção: %s %v", r.Conteudo, err)
	}
	if len(eco.pedidos) != 2 || !strings.Contains(eco.pedidos[1].Historico[len(eco.pedidos[1].Historico)-1].Conteudo, "marcadores") {
		t.Fatal("deveria pedir correção à IA")
	}
	eco = &motorEco{respostas: []string{`{"q":"[PESSOA-09]"}`}}
	_, err = Envolver(eco, m).Gerar(context.Background(), inference.Pedido{Usuario: "x"})
	var e *erros.Erro
	if !errors.As(err, &e) || e.Codigo != erros.IARespostaIncompleta || len(eco.pedidos) != 1+reenviosMarcador {
		t.Fatalf("esperado IA-010 após %d correções: %v (%d pedidos)", reenviosMarcador, err, len(eco.pedidos))
	}
}

func TestNomesDoUsuario(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"  Beltrano  de   Tal ", "x", "998877"})
	oc := m.Ocultar("Beltrano de Tal e BELTRANO DE TAL, mat 998877.")
	if strings.Contains(oc, "Beltrano") || strings.Contains(oc, "BELTRANO") || strings.Contains(oc, "998877") {
		t.Fatalf("nomes informados pelo usuário: %q", oc)
	}
}
