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

	// Modelo de IA (Gemini API, emenda V4 §47). MOD-001…009 pertenciam à
	// inferência local, removida (D-10): ficam reservados e nunca serão
	// reutilizados.
	IAChaveAusente       Codigo = "IA-001"
	IAChaveInvalida      Codigo = "IA-002"
	IAPermissaoNegada    Codigo = "IA-003"
	IAModeloIndisponivel Codigo = "IA-004"
	IACota               Codigo = "IA-005"
	IAIndisponivel       Codigo = "IA-006"
	IASemConexao         Codigo = "IA-007"
	IATempoEsgotado      Codigo = "IA-008"
	IARespostaBloqueada  Codigo = "IA-009"
	IARespostaIncompleta Codigo = "IA-010"
	IAEnvioNaoAutorizado Codigo = "IA-011"
	IACredencial         Codigo = "IA-012"
	IASemCreditos        Codigo = "IA-013"

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
	TarefaRespostaInvalid Codigo = "TAR-006"

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

	def(IAChaveAusente, CatModelo, "Chave da Gemini API não cadastrada",
		"A análise por IA usa a Gemini API, e ainda não há uma chave cadastrada neste computador.",
		"Cadastre sua chave em Configurações de IA e tente novamente.")
	def(IAChaveInvalida, CatModelo, "Chave da Gemini API não aceita",
		"O Google não aceitou a chave cadastrada. Ela pode estar incorreta, revogada ou excluída.",
		"Cadastre uma chave válida em Configurações de IA e tente novamente.")
	def(IAPermissaoNegada, CatModelo, "Acesso negado pela Gemini API",
		"A chave cadastrada não tem permissão para usar a Gemini API ou o modelo escolhido, ou o serviço não está liberado para a sua conta ou região.",
		"Verifique a chave e o projeto no Google AI Studio, ou escolha outro modelo em Configurações de IA.")
	def(IAModeloIndisponivel, CatModelo, "Modelo de IA indisponível",
		"O modelo de IA escolhido não está disponível para a sua chave, ou nenhum modelo foi escolhido.",
		"Escolha um modelo em Configurações de IA e tente novamente.")
	def(IACota, CatModelo, "Limite de uso da Gemini API atingido",
		"A cota de uso da sua chave foi atingida (por minuto ou por dia).",
		"Aguarde alguns minutos e tente novamente. Se o limite for diário, ele é renovado no dia seguinte; os limites podem ser consultados no Google AI Studio.")
	def(IAIndisponivel, CatModelo, "Gemini API indisponível",
		"O serviço da Gemini API está temporariamente indisponível ou sobrecarregado.",
		"Tente novamente em alguns minutos. O andamento da tarefa foi mantido.")
	def(IASemConexao, CatModelo, "Sem conexão com a Gemini API",
		"A análise por IA exige internet, e não foi possível se conectar ao serviço do Google.",
		"Verifique a conexão (e o proxy da rede, se houver) e tente novamente. O andamento da tarefa foi mantido.")
	def(IATempoEsgotado, CatModelo, "A Gemini API demorou demais para responder",
		"A resposta da IA não chegou dentro do tempo limite.",
		"Tente novamente. O andamento da tarefa foi mantido.")
	def(IARespostaBloqueada, CatModelo, "Resposta recusada pelo provedor",
		"O Google recusou processar este conteúdo dos autos pelas regras dele.",
		"Tente novamente ou escolha outro modelo. Se persistir, esta etapa precisará ser feita manualmente.")
	def(IARespostaIncompleta, CatModelo, "Resposta incompleta da IA",
		"A resposta da Gemini API chegou incompleta ou vazia e foi descartada.",
		"Tente novamente. Nenhum resultado incompleto foi aproveitado.")
	def(IAEnvioNaoAutorizado, CatModelo, "Envio à Gemini API não autorizado",
		"Você optou por não enviar o conteúdo dos autos à Gemini API; nada foi enviado.",
		"Para continuar a análise por IA, use Tentar novamente e autorize o envio, ou cancele a tarefa.")
	def(IACredencial, CatModelo, "Não foi possível guardar ou ler a chave",
		"O Windows não conseguiu proteger ou abrir a chave da Gemini API desta conta.",
		"Cadastre a chave novamente em Configurações de IA.")
	def(IASemCreditos, CatModelo, "Sem créditos na conta da Gemini API",
		"O projeto do Google ligado à chave está sem créditos pré-pagos ou com o faturamento pendente, e o Google recusou a chamada.",
		"Adicione créditos ou regularize o faturamento do projeto no Google AI Studio (ai.studio/projects), ou cadastre a chave de outro projeto, e tente novamente.")

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
	def(TarefaRespostaInvalid, CatTarefa, "Resposta incompleta",
		"Há campos obrigatórios sem resposta ou com formato inválido.",
		"Confira os campos destacados e envie de novo.")

	def(InternoInesperado, CatInterno, "Erro inesperado",
		"Ocorreu um erro inesperado no aplicativo.",
		"Tente novamente. Se persistir, reinicie o aplicativo.")
}
