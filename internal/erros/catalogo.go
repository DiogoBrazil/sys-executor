package erros

// Códigos do catálogo. Novos códigos entram no FIM de cada grupo; um código
// publicado nunca muda de significado.
const (
	// SEI (Fase 2 implementa a automação; os códigos já ficam reservados)
	SEIIndisponivel       Codigo = "SEI-001"
	SEICredencial         Codigo = "SEI-002"
	SEIOTP                Codigo = "SEI-003"
	SEIAcessoNegado       Codigo = "SEI-004"
	SEIInterfaceAlterada  Codigo = "SEI-005"
	SEIGeracaoFalhou      Codigo = "SEI-006"
	SEIDownloadIncompleto Codigo = "SEI-007"
	SEIProcessoNaoAchado  Codigo = "SEI-008"

	// Rede
	RedeSemConexao Codigo = "REDE-001"

	// PDF
	PDFVazio      Codigo = "PDF-001"
	PDFInvalido   Codigo = "PDF-002"
	PDFCorrompido Codigo = "PDF-003"
	PDFProtegido  Codigo = "PDF-004"
	PDFGrande     Codigo = "PDF-005"

	// OCR
	OCRFalhou        Codigo = "OCR-001"
	OCRBaixaConfianc Codigo = "OCR-002"

	// Arquivos
	ArquivoLeitura  Codigo = "ARQ-001"
	ArquivoEscrita  Codigo = "ARQ-002"
	ArquivoSemEspac Codigo = "ARQ-003"

	// Modelo de IA
	ModeloAusente       Codigo = "MOD-001"
	ModeloIncompativel  Codigo = "MOD-002"
	ModeloSemMemoria    Codigo = "MOD-003"
	ModeloTravado       Codigo = "MOD-004"
	ModeloGPUInstavel   Codigo = "MOD-005"
	ModeloHardwareFraco Codigo = "MOD-006"
	ModeloDownload      Codigo = "MOD-007"
	ModeloIntegridade   Codigo = "MOD-008"
	ModeloEmUso         Codigo = "MOD-009"

	// Biblioteca jurídica e referências
	BibliotecaIndisponivel Codigo = "BIB-001"
	BibliotecaEmUso        Codigo = "BIB-002"
	ReferenciaNaoEncontrad Codigo = "REF-001"

	// Validação e exportação
	ValidacaoBloqueou Codigo = "VAL-001"
	ExportacaoFalhou  Codigo = "EXP-001"

	// Pesquisa web
	PesquisaIndisponivel Codigo = "PESQ-001"

	// Sessão (diretório efêmero)
	SessaoCriacao  Codigo = "SES-001"
	SessaoProtecao Codigo = "SES-002"
	SessaoLimpeza  Codigo = "SES-003"

	// Tarefa (fluxo)
	TarefaJaAtiva         Codigo = "TAR-001"
	TarefaOrdemInvalida   Codigo = "TAR-002"
	TarefaSemTarefa       Codigo = "TAR-003"
	TarefaEtapaIndisponiv Codigo = "TAR-004"
	TarefaCancelada       Codigo = "TAR-005"

	// Interno
	InternoInesperado Codigo = "INT-001"
)

var ordem []Codigo

var catalogo = map[Codigo]Definicao{}

func def(c Codigo, cat Categoria, titulo, msg, acao string) {
	if _, dup := catalogo[c]; dup {
		panic("erros: código duplicado " + string(c))
	}
	catalogo[c] = Definicao{Codigo: c, Categoria: cat, Titulo: titulo, Mensagem: msg, Acao: acao}
	ordem = append(ordem, c)
}

func init() {
	def(SEIIndisponivel, CatSEI, "SEI indisponível",
		"Não foi possível acessar o SEI/RO agora. O sistema pode estar fora do ar ou lento.",
		"Tente novamente em alguns minutos ou envie o PDF completo do processo.")
	def(SEICredencial, CatSEI, "Usuário ou senha não aceitos",
		"O SEI/RO não aceitou o usuário ou a senha informados.",
		"Confira os dados e tente novamente. A senha não é guardada pelo aplicativo.")
	def(SEIOTP, CatSEI, "Código de verificação não aceito",
		"O código de autenticação em dois fatores não foi aceito pelo SEI/RO.",
		"Aguarde um novo código no aplicativo autenticador e informe-o novamente.")
	def(SEIAcessoNegado, CatSEI, "Acesso ao processo negado",
		"O SEI/RO informou que seu usuário não tem acesso a este processo.",
		"Verifique o número do processo e suas permissões, ou envie o PDF completo.")
	def(SEIInterfaceAlterada, CatSEI, "Tela do SEI não reconhecida",
		"A interface do SEI/RO parece ter mudado e o aplicativo não reconheceu a tela atual.",
		"Envie o PDF completo do processo para continuar. Nenhuma ação foi feita no SEI.")
	def(SEIGeracaoFalhou, CatSEI, "Falha ao gerar o PDF no SEI",
		"O SEI/RO não concluiu a geração do PDF do processo.",
		"Tente novamente ou gere o PDF no SEI e envie-o pelo aplicativo.")
	def(SEIDownloadIncompleto, CatSEI, "Download incompleto",
		"O PDF do processo não foi baixado por completo.",
		"Tente novamente. Se persistir, envie o PDF completo manualmente.")
	def(SEIProcessoNaoAchado, CatSEI, "Processo não encontrado",
		"O SEI/RO não encontrou um processo com o número informado.",
		"Confira o número do processo e tente novamente.")

	def(RedeSemConexao, CatRede, "Sem conexão com a internet",
		"A obtenção automática dos autos no SEI/RO exige internet, e não há conexão disponível.",
		"Envie o PDF completo do processo. A escolha do tipo de minuta será mantida.")

	def(PDFVazio, CatPDF, "Arquivo vazio",
		"O arquivo selecionado está vazio.",
		"Selecione o PDF completo do processo.")
	def(PDFInvalido, CatPDF, "O arquivo não é um PDF",
		"O arquivo selecionado não é um PDF válido.",
		"Selecione o PDF consolidado gerado pelo SEI/RO.")
	def(PDFCorrompido, CatPDF, "PDF danificado",
		"Não foi possível ler o PDF; o arquivo pode estar corrompido ou incompleto.",
		"Gere o PDF novamente no SEI/RO e envie a nova cópia.")
	def(PDFProtegido, CatPDF, "PDF protegido por senha",
		"O PDF está protegido e não pode ser lido pelo aplicativo.",
		"Gere uma cópia sem proteção no SEI/RO e envie-a.")
	def(PDFGrande, CatPDF, "PDF grande demais",
		"O arquivo ultrapassa o tamanho máximo aceito pelo aplicativo.",
		"Confira se o arquivo é mesmo o PDF do processo. Se for, informe o suporte.")

	def(OCRFalhou, CatOCR, "Falha no reconhecimento de texto",
		"Não foi possível reconhecer o texto de páginas digitalizadas.",
		"Tente novamente. Se persistir, envie uma versão do PDF com texto pesquisável.")
	def(OCRBaixaConfianc, CatOCR, "Página com leitura duvidosa",
		"Uma peça importante tem qualidade de imagem baixa e a leitura pode estar errada.",
		"Envie uma versão mais legível da peça ou confirme o conteúdo quando solicitado.")

	def(ArquivoLeitura, CatArquivo, "Não foi possível ler o arquivo",
		"O arquivo selecionado não pôde ser aberto.",
		"Verifique se ele não está aberto em outro programa e se você tem permissão de leitura.")
	def(ArquivoEscrita, CatArquivo, "Não foi possível gravar o arquivo",
		"O aplicativo não conseguiu gravar um arquivo necessário.",
		"Verifique as permissões da pasta escolhida e tente novamente.")
	def(ArquivoSemEspac, CatArquivo, "Espaço em disco insuficiente",
		"Não há espaço livre suficiente no disco para continuar.",
		"Libere espaço em disco e tente novamente.")

	def(ModeloAusente, CatModelo, "Modelo de IA não instalado",
		"Nenhum modelo de IA local está instalado.",
		"Baixe ou importe um modelo na tela Modelos.")
	def(ModeloIncompativel, CatModelo, "Modelo de IA incompatível",
		"O arquivo de modelo não é compatível com este aplicativo.",
		"Escolha um modelo da lista recomendada ou importe outro arquivo GGUF.")
	def(ModeloSemMemoria, CatModelo, "Memória insuficiente para o modelo",
		"O computador não tem memória livre suficiente para executar o modelo escolhido.",
		"Feche outros programas ou escolha um modelo menor.")
	def(ModeloTravado, CatModelo, "O modelo de IA parou de responder",
		"O mecanismo de IA local parou de responder e foi encerrado.",
		"Tente novamente. Se persistir, escolha um modelo menor.")
	def(ModeloGPUInstavel, CatModelo, "Placa de vídeo instável",
		"Uma tentativa anterior de usar a placa de vídeo não terminou bem; o aplicativo usará o processador.",
		"Nenhuma ação é necessária. O uso da placa de vídeo pode ser reativado em Configurações avançadas.")
	def(ModeloHardwareFraco, CatModelo, "Computador sem recursos suficientes",
		"Nenhum modelo disponível cabe nos recursos deste computador.",
		"Use um computador com mais memória ou feche outros programas.")
	def(ModeloDownload, CatModelo, "Download do modelo interrompido",
		"Não foi possível concluir o download do modelo de IA.",
		"Verifique a conexão e tente de novo; o download continua de onde parou.")
	def(ModeloIntegridade, CatModelo, "Arquivo de modelo diferente do esperado",
		"O arquivo do modelo não confere com a assinatura (SHA-256) do catálogo e foi descartado.",
		"Baixe o modelo novamente ou importe o arquivo original.")
	def(ModeloEmUso, CatModelo, "Modelo em uso",
		"Este modelo está sendo usado pela tarefa em andamento e não pode ser removido agora.",
		"Conclua ou cancele a tarefa e tente novamente.")

	def(BibliotecaIndisponivel, CatBiblioteca, "Biblioteca jurídica indisponível",
		"A biblioteca jurídica local não pôde ser aberta.",
		"Reinicie o aplicativo. Se persistir, reinstale-o; seus arquivos exportados não serão afetados.")
	def(BibliotecaEmUso, CatBiblioteca, "Documento em uso",
		"Este documento está sendo usado pela tarefa em andamento e não pode ser excluído agora.",
		"Conclua ou cancele a tarefa e tente novamente.")
	def(ReferenciaNaoEncontrad, CatReferencia, "Referência não encontrada",
		"Um dispositivo legal necessário não foi encontrado na biblioteca jurídica.",
		"Inclua o documento na biblioteca ou confirme a referência quando solicitado.")

	def(ValidacaoBloqueou, CatValidacao, "A minuta não passou na validação",
		"A validação automática encontrou problemas que não puderam ser corrigidos sozinhos.",
		"Responda às perguntas apresentadas ou solicite ajustes.")
	def(ExportacaoFalhou, CatExportacao, "Falha ao gerar o arquivo da minuta",
		"Não foi possível gerar o DOCX ou o PDF da minuta.",
		"Tente novamente ou escolha outra pasta de destino.")

	def(PesquisaIndisponivel, CatPesquisa, "Pesquisa externa indisponível",
		"A consulta a fontes oficiais na internet não está disponível agora.",
		"A análise continua só com a biblioteca local, e a lacuna será indicada na minuta.")

	def(SessaoCriacao, CatSessao, "Não foi possível preparar a área de trabalho",
		"O aplicativo não conseguiu criar a pasta temporária protegida desta tarefa.",
		"Verifique o espaço em disco e as permissões da sua conta do Windows.")
	def(SessaoProtecao, CatSessao, "Não foi possível proteger a área de trabalho",
		"O aplicativo não conseguiu restringir o acesso à pasta temporária da tarefa.",
		"A tarefa não será iniciada por segurança. Reinicie o aplicativo e tente novamente.")
	def(SessaoLimpeza, CatSessao, "Limpeza incompleta",
		"Alguns arquivos temporários não puderam ser removidos agora.",
		"Eles serão removidos automaticamente na próxima abertura do aplicativo.")

	def(TarefaJaAtiva, CatTarefa, "Já existe uma tarefa em andamento",
		"Só é possível trabalhar em uma minuta por vez.",
		"Conclua ou cancele a tarefa atual antes de iniciar outra.")
	def(TarefaOrdemInvalida, CatTarefa, "Passo fora de ordem",
		"Esta ação não está disponível neste momento da tarefa.",
		"Siga as etapas na ordem indicada na tela.")
	def(TarefaSemTarefa, CatTarefa, "Nenhuma tarefa em andamento",
		"Não há tarefa ativa para esta ação.",
		"Inicie uma nova tarefa na tela inicial.")
	def(TarefaEtapaIndisponiv, CatTarefa, "Etapa ainda não disponível",
		"Esta etapa do processamento ainda não está disponível nesta versão do aplicativo.",
		"Os arquivos temporários desta tarefa foram descartados. Inicie uma nova tarefa quando a etapa estiver disponível.")
	def(TarefaCancelada, CatTarefa, "Tarefa cancelada",
		"A tarefa foi cancelada e os arquivos temporários foram descartados.",
		"Inicie uma nova tarefa quando desejar.")

	def(InternoInesperado, CatInterno, "Erro inesperado",
		"Ocorreu um erro inesperado no aplicativo.",
		"Tente novamente. Se persistir, reinicie o aplicativo.")
}
