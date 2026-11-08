// Package civil trata datas e horas de calendário dos cálculos disciplinares
// (plano §8; V4 §§7, 15.2, 15.5; D-07).
//
// Datas não têm fuso nem hora implícita. Horas são opcionais: só entram na
// comparação quando conhecidas dos dois lados (V4 §15.5-5).
package civil

import (
	"errors"
	"fmt"
	"time"
)

// Data de calendário.
type Data struct {
	Ano, Mes, Dia int
}

// NovaData valida e cria a data.
func NovaData(ano, mes, dia int) (Data, error) {
	d := Data{Ano: ano, Mes: mes, Dia: dia}
	if !d.Valida() {
		return Data{}, fmt.Errorf("data inválida: %02d/%02d/%04d", dia, mes, ano)
	}
	return d, nil
}

// D cria a data e entra em pânico se for inválida (uso em testes e tabelas fixas).
func D(dia, mes, ano int) Data {
	d, err := NovaData(ano, mes, dia)
	if err != nil {
		panic(err)
	}
	return d
}

// Valida informa se a data existe no calendário.
func (d Data) Valida() bool {
	if d.Mes < 1 || d.Mes > 12 || d.Dia < 1 || d.Ano < 1 {
		return false
	}
	t := time.Date(d.Ano, time.Month(d.Mes), d.Dia, 0, 0, 0, 0, time.UTC)
	return t.Day() == d.Dia && int(t.Month()) == d.Mes
}

// Zero informa se a data não foi informada.
func (d Data) Zero() bool { return d == Data{} }

func (d Data) String() string { return fmt.Sprintf("%02d/%02d/%04d", d.Dia, d.Mes, d.Ano) }

// Cmp compara: -1 se d < o; 0 se iguais; 1 se d > o.
func (d Data) Cmp(o Data) int {
	switch {
	case d.Ano != o.Ano:
		return sinal(d.Ano - o.Ano)
	case d.Mes != o.Mes:
		return sinal(d.Mes - o.Mes)
	default:
		return sinal(d.Dia - o.Dia)
	}
}

func sinal(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	}
	return 0
}

// Bissexto informa se o ano é bissexto.
func Bissexto(ano int) bool { return ano%4 == 0 && (ano%100 != 0 || ano%400 == 0) }

// Hora do dia, opcional.
type Hora struct {
	H, Min    int
	Conhecida bool
}

// H cria uma hora conhecida.
func H(h, min int) Hora { return Hora{H: h, Min: min, Conhecida: true} }

// Instante é uma data com hora opcional.
type Instante struct {
	Data Data
	Hora Hora
}

// Relacao entre dois instantes.
type Relacao string

const (
	Antes         Relacao = "ANTES"
	Depois        Relacao = "DEPOIS"
	Simultaneo    Relacao = "SIMULTANEO"
	Indeterminado Relacao = "INDETERMINADO" // mesma data e alguma hora desconhecida
)

// Comparar diz se a está antes, depois ou ao mesmo tempo que b. Na mesma data,
// só as horas conhecidas dos dois lados decidem; faltando uma, é indeterminado
// (V4 §15.5-5: sem anterioridade comprovada).
func Comparar(a, b Instante) Relacao {
	switch a.Data.Cmp(b.Data) {
	case -1:
		return Antes
	case 1:
		return Depois
	}
	if !a.Hora.Conhecida || !b.Hora.Conhecida {
		return Indeterminado
	}
	ma, mb := a.Hora.H*60+a.Hora.Min, b.Hora.H*60+b.Hora.Min
	switch {
	case ma < mb:
		return Antes
	case ma > mb:
		return Depois
	}
	return Simultaneo
}

// Regra29Fev é a escolha do usuário para o aniversário de 29/02 em ano não
// bissexto (V4 §§12.2-8, 15.2; P-03). O motor não escolhe sozinho.
type Regra29Fev string

const (
	SemEscolha29Fev Regra29Fev = ""      // padrão: devolve ErrCasoLimite29Fev
	Usar28Fev       Regra29Fev = "28/02" // escolhido pelo usuário para o caso
	Usar01Mar       Regra29Fev = "01/03" // escolhido pelo usuário para o caso
)

// ErrCasoLimite29Fev pede a decisão do usuário (só quando o caso for material).
var ErrCasoLimite29Fev = errors.New("aniversário de 29/02 em ano não bissexto: critério não definido; perguntar ao usuário")

// Aniversario devolve a data n anos depois de d, no calendário.
func Aniversario(d Data, n int, regra Regra29Fev) (Data, error) {
	alvo := d.Ano + n
	if d.Mes == 2 && d.Dia == 29 && !Bissexto(alvo) {
		switch regra {
		case Usar28Fev:
			return Data{Ano: alvo, Mes: 2, Dia: 28}, nil
		case Usar01Mar:
			return Data{Ano: alvo, Mes: 3, Dia: 1}, nil
		default:
			return Data{}, ErrCasoLimite29Fev
		}
	}
	return Data{Ano: alvo, Mes: d.Mes, Dia: d.Dia}, nil
}

// MenosDeUmAno informa se o intervalo entre as datas é ESTRITAMENTE inferior
// a um ano (V4 §15.2): o aniversário exato já não vale. A ordem das datas não
// importa. Ex.: 07/11/2026 → 06/11/2027 vale; → 07/11/2027 não vale.
func MenosDeUmAno(a, b Data, regra Regra29Fev) (bool, error) {
	if a.Cmp(b) > 0 {
		a, b = b, a
	}
	lim, err := Aniversario(a, 1, regra)
	if err != nil {
		return false, err
	}
	return b.Cmp(lim) < 0, nil
}

// DentroPrazoReincidencia informa se o fato atual ocorreu dentro de n anos do
// término do cumprimento da punição anterior, CONTANDO o dia do aniversário
// (D-07; RDPM art. 39, p.ú.: só se desconsidera tempo "superior"). Função
// distinta de MenosDeUmAno de propósito (V4 §15.6; PADS-REI-011).
// Não verifica a anterioridade (ver Comparar).
func DentroPrazoReincidencia(termino, fato Data, anos int, regra Regra29Fev) (bool, error) {
	lim, err := Aniversario(termino, anos, regra)
	if err != nil {
		return false, err
	}
	return fato.Cmp(lim) <= 0, nil
}
