// Package workflow controla o ciclo de vida de uma tarefa (plano §6).
//
// A tarefa existe só em memória e na sessão efêmera; não há serialização
// para retomada (V4 §29.1). Uma única tarefa ativa por instância (SES-CONC-01).
package workflow

// Tipo é o procedimento escolhido na primeira tela (V4 §30.1).
type Tipo string

const (
	TipoPADS Tipo = "PADS"
	TipoSR   Tipo = "SR"
)

// Entrada é a forma de obtenção dos autos (V4 §30.1, segunda escolha).
type Entrada string

const (
	EntradaSEI    Entrada = "SEI"
	EntradaUpload Entrada = "UPLOAD"
)

// Estado da tarefa.
type Estado string

const (
	Nova             Estado = "NOVA"
	TipoEscolhido    Estado = "TIPO_ESCOLHIDO"
	EntradaEscolhida Estado = "ENTRADA_ESCOLHIDA"

	// Obtenção dos autos pelo SEI (Fase 2).
	SEIConectividade  Estado = "SEI_CONECTIVIDADE"
	SEILogin          Estado = "SEI_LOGIN"
	SEIOTP            Estado = "SEI_OTP"
	SEILocalizando    Estado = "SEI_LOCALIZANDO"
	SEIReautenticacao Estado = "SEI_REAUTENTICACAO"
	SEIGerandoPDF     Estado = "SEI_GERANDO_PDF"
	SEIBaixando       Estado = "SEI_BAIXANDO"

	// Obtenção dos autos por upload.
	RecebendoUpload Estado = "RECEBENDO_UPLOAD"

	Verificando       Estado = "VERIFICANDO"
	Extraindo         Estado = "EXTRAINDO"
	Classificando     Estado = "CLASSIFICANDO"
	Compatibilidade   Estado = "COMPATIBILIDADE"
	Delimitando       Estado = "DELIMITANDO"
	Analisando        Estado = "ANALISANDO"
	ConsultandoFontes Estado = "CONSULTANDO_FONTES"
	Calculando        Estado = "CALCULANDO"
	Redigindo         Estado = "REDIGINDO"
	Validando         Estado = "VALIDANDO"
	AguardandoRevisao Estado = "AGUARDANDO_REVISAO"
	Ajustando         Estado = "AJUSTANDO"
	Finalizando       Estado = "FINALIZANDO"

	AguardandoEsclarecimento Estado = "AGUARDANDO_ESCLARECIMENTO"

	Finalizada Estado = "FINALIZADA"
	Cancelada  Estado = "CANCELADA"
	Falha      Estado = "FALHA"
)

// transicoes lista os sucessores permitidos de cada estado. CANCELADA e FALHA
// são permitidas a partir de qualquer estado não terminal (ver Permitida), e
// AGUARDANDO_ESCLARECIMENTO a partir dos estados analíticos.
var transicoes = map[Estado][]Estado{
	Nova:             {TipoEscolhido},
	TipoEscolhido:    {TipoEscolhido, EntradaEscolhida},
	EntradaEscolhida: {TipoEscolhido, SEIConectividade, RecebendoUpload},

	// T03: sem internet → volta à escolha da entrada, preservando o tipo.
	SEIConectividade:  {TipoEscolhido, SEILogin},
	SEILogin:          {SEIOTP, SEILocalizando},
	SEIOTP:            {SEILocalizando},
	SEILocalizando:    {SEIReautenticacao, SEIGerandoPDF},
	SEIReautenticacao: {SEIGerandoPDF},
	SEIGerandoPDF:     {SEIBaixando},
	SEIBaixando:       {Verificando},

	// Upload: o usuário pode voltar e escolher o SEI enquanto não enviou o PDF.
	RecebendoUpload: {TipoEscolhido, Verificando},

	Verificando:       {Extraindo, RecebendoUpload}, // PDF corrompido/protegido: pedir outro
	Extraindo:         {Classificando},
	Classificando:     {Compatibilidade},
	Compatibilidade:   {Delimitando},
	Delimitando:       {Analisando},
	Analisando:        {ConsultandoFontes},
	ConsultandoFontes: {Calculando, Redigindo}, // SR não calcula (V4 §21)
	Calculando:        {Redigindo},
	Redigindo:         {Validando},
	Validando:         {AguardandoRevisao, Redigindo}, // laço de correção
	AguardandoRevisao: {Ajustando, Finalizando},
	Ajustando:         {Validando},
	Finalizando:       {Finalizada},
}

// analiticos podem pausar para uma pergunta pontual (V4 §31.2).
var analiticos = map[Estado]bool{
	Classificando: true, Compatibilidade: true, Delimitando: true, Analisando: true,
	ConsultandoFontes: true, Calculando: true, Redigindo: true, Validando: true, Ajustando: true,
}

// Terminal informa se o estado encerra a tarefa.
func (e Estado) Terminal() bool { return e == Finalizada || e == Cancelada || e == Falha }

// Permitida informa se de → para é uma transição válida. A saída de
// AGUARDANDO_ESCLARECIMENTO é validada pelo gerenciador (volta à origem).
func Permitida(de, para Estado) bool {
	if de.Terminal() {
		return false
	}
	if para == Cancelada || para == Falha {
		return true
	}
	if para == AguardandoEsclarecimento {
		return analiticos[de]
	}
	for _, s := range transicoes[de] {
		if s == para {
			return true
		}
	}
	return false
}

// Todos lista todos os estados (para testes e para a interface).
func Todos() []Estado {
	return []Estado{Nova, TipoEscolhido, EntradaEscolhida, SEIConectividade, SEILogin, SEIOTP,
		SEILocalizando, SEIReautenticacao, SEIGerandoPDF, SEIBaixando, RecebendoUpload, Verificando,
		Extraindo, Classificando, Compatibilidade, Delimitando, Analisando, ConsultandoFontes,
		Calculando, Redigindo, Validando, AguardandoRevisao, Ajustando, Finalizando,
		AguardandoEsclarecimento, Finalizada, Cancelada, Falha}
}
