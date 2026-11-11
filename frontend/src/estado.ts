// Estado da interface: espelha a visão da tarefa enviada pelo Go.
// Nada aqui é persistido (sem localStorage): a tarefa vive só na sessão.
import { reactive } from "vue";
import {
  api, cadastro as cadastroApi, credenciais, ia as iaApi, responder as responderApi, tarefa,
  type Cadastro, type DispositivoVisao, type Resposta, type Visao, type VisaoIA,
} from "./api";

export interface ErroTela {
  codigo: string;
  titulo: string;
  mensagem: string;
  acao: string;
  detalhe?: string;
}

export const estado = reactive({
  visao: null as Visao | null,
  erro: null as ErroTela | null, // erro do último comando (não encerra a tarefa)
  ocupado: false,
  ia: null as VisaoIA | null,
  verConfigIA: false, // tela Configurações de IA aberta por cima da atual
});

// mostrar aplica uma visão do Go, salvo se for mais antiga que a atual: a
// resposta de um comando pode chegar depois dos eventos emitidos em seguida.
function mostrar(v: Visao | null) {
  const atual = estado.visao;
  if (v && atual && atual.id === v.id && (v.sequencia ?? 0) < (atual.sequencia ?? 0)) return;
  estado.visao = v;
}

function aplicar(r: Resposta) {
  mostrar(r.visao ?? null);
  estado.erro = r.erro ? { ...r.erro } : null;
}

async function executar(cmd: () => Promise<Resposta>): Promise<boolean> {
  if (estado.ocupado) return false;
  estado.ocupado = true;
  try {
    const r = await cmd();
    aplicar(r);
    return !r.erro;
  } catch {
    // Falha de comunicação com o backend: mensagem do catálogo (INT-001).
    estado.erro = {
      codigo: "INT-001",
      titulo: "Erro inesperado",
      mensagem: "Ocorreu um erro inesperado no aplicativo.",
      acao: "Tente novamente. Se persistir, reinicie o aplicativo.",
    };
    return false;
  } finally {
    estado.ocupado = false;
  }
}

export async function carregar() {
  api.aoMudar((v) => mostrar(v));
  iaApi.aoMudar((m) => {
    estado.ia = m;
  });
  await executar(api.atual);
  await atualizarIA();
}

// ---- configurações de IA (Gemini API e ChatGPT) ----
export async function atualizarIA() {
  try {
    estado.ia = await iaApi.estado();
  } catch {
    estado.ia = null;
  }
}
async function comandoIA(f: () => Promise<VisaoIA>) {
  try {
    estado.ia = await f();
  } catch {
    /* a visão anterior continua; o próximo evento atualiza */
  }
}
export const abrirConfigIA = async () => {
  estado.verConfigIA = true;
  await atualizarIA();
};
export const fecharConfigIA = () => {
  estado.verConfigIA = false;
};
// A chave não fica em nenhum estado da interface: vai direto ao Go.
export const salvarChave = (chave: string) => comandoIA(() => iaApi.salvarChave(chave));
export const removerChave = () => comandoIA(iaApi.removerChave);
export const verificarIA = () => comandoIA(iaApi.verificar);
export const selecionarModelo = (id: string) => comandoIA(() => iaApi.selecionarModelo(id));
export const testarModelo = (id: string) => comandoIA(() => iaApi.testarModelo(id));
export const abrirPaginaChaves = () => comandoIA(iaApi.abrirPaginaChaves);
// ChatGPT (V4 §48.3): a entrada abre o navegador do sistema e volta na hora;
// o resultado chega pelo evento ia:visao.
export const selecionarProvedor = (p: string) => comandoIA(() => iaApi.selecionarProvedor(p));
export const entrarChatGPT = () => comandoIA(iaApi.entrarChatGPT);
export const cancelarEntradaChatGPT = () => comandoIA(iaApi.cancelarEntradaChatGPT);
export const desconectarChatGPT = () => comandoIA(iaApi.desconectarChatGPT);
export const atualizarModelosChatGPT = () => comandoIA(iaApi.atualizarModelosChatGPT);
export const selecionarModeloChatGPT = (id: string) => comandoIA(() => iaApi.selecionarModeloChatGPT(id));
export const testarModeloChatGPT = (id: string) => comandoIA(() => iaApi.testarModeloChatGPT(id));
export const abrirUsoChatGPT = () => comandoIA(iaApi.abrirUsoChatGPT);

// iaPronta: o serviço escolhido está configurado para analisar.
export function iaPronta(): boolean {
  const m = estado.ia;
  if (!m) return false;
  const provedor = m.provedor || (m.modeloSelecionado ? "GEMINI" : "");
  if (provedor === "CHATGPT") {
    const c = m.chatgpt;
    return !!c && c.conectada && c.planoAutorizado && !!c.modeloSelecionado;
  }
  if (provedor === "GEMINI") return m.chaveCadastrada && !!m.modeloSelecionado;
  return false;
}

export async function escolherTipo(tipo: "PADS" | "SR") {
  if (!estado.visao || estado.visao.terminal) {
    if (estado.visao?.terminal) await executar(api.descartar);
    if (!(await executar(api.iniciar))) return;
  }
  await executar(() => api.escolherTipo(tipo));
}

// cadastrarPecas: entrada peça por peça (V4 §48.4; o PDF único não é mais aceito).
export async function cadastrarPecas() {
  if (estado.visao?.estado === "TIPO_ESCOLHIDO") await executar(() => api.escolherEntrada("UPLOAD"));
}

// ---- cadastro de peças ----
// Cada comando de arquivo grava antes os metadados editados (salvar), para
// que nada digitado se perca quando a visão nova chega do Go.
export const salvarCadastro = (c: Cadastro) => executar(() => cadastroApi.salvar(c));
async function comSalvar(c: Cadastro | null, cmd: () => Promise<Resposta>) {
  if (c && !(await salvarCadastro(c))) return false;
  return executar(cmd);
}
export const anexarArquivos = (c: Cadastro | null, pecaID: string) => comSalvar(c, () => cadastroApi.anexar(pecaID));
export const anexarDocumentosSEI = (c: Cadastro | null, pecaID: string, documentos: string[]) =>
  comSalvar(c, () => cadastroApi.anexarSEI(pecaID, documentos));
export const removerArquivo = (c: Cadastro | null, pecaID: string, arqID: string) => comSalvar(c, () => cadastroApi.remover(pecaID, arqID));
export const moverArquivo = (c: Cadastro | null, pecaID: string, arqID: string, delta: number) =>
  comSalvar(c, () => cadastroApi.mover(pecaID, arqID, delta));
export const concluirCadastro = (c: Cadastro | null) => comSalvar(c, cadastroApi.concluir);

let catalogo: Promise<DispositivoVisao[]> | null = null;
export function catalogoEnquadramento(): Promise<DispositivoVisao[]> {
  if (!catalogo) catalogo = cadastroApi.catalogo().catch(() => { catalogo = null; return []; });
  return catalogo;
}

export const responder = (perguntaID: string, opcaoID: string, valores: Record<string, string> = {}) =>
  executar(() => responderApi(perguntaID, opcaoID, valores));
export const revisar = (perguntaID: string) => executar(() => tarefa.revisar(perguntaID));
export const retomar = () => executar(tarefa.retomar);
// Minuta em revisão: o pedido de ajuste não fica guardado na interface.
export const solicitarAjuste = (texto: string) => executar(() => tarefa.solicitarAjuste(texto));
export const aprovar = () => executar(tarefa.aprovar);
// salvarMinuta devolve true quando o arquivo foi gravado (false se o
// usuário fechou o diálogo ou houve erro, que fica em estado.erro).
export async function salvarMinuta(formato: "docx" | "pdf"): Promise<boolean> {
  let salvo = false;
  await executar(async () => {
    const r = await tarefa.salvarMinuta(formato);
    salvo = !!r.salvo;
    return r;
  });
  return salvo;
}
export const informarCredenciais = (id: string, usuario: string, senha: string, orgao: string, processo: string) =>
  executar(() => credenciais.informar(id, usuario, senha, orgao, processo));
export const informarSegredo = (id: string, valor: string) => executar(() => credenciais.segredo(id, valor));
export const acessarSEI = () => executar(() => api.escolherEntrada("SEI"));
export const voltarParaEntrada = () => executar(api.voltarParaEntrada);
export const cancelar = () => executar(api.cancelar);
export const limparErro = () => {
  estado.erro = null;
};

// Volta à tela inicial: cancela a tarefa ainda aberta (se houver) e a descarta.
export async function voltarAoInicio() {
  if (estado.visao && !estado.visao.terminal) {
    if (!(await executar(api.cancelar))) return;
  }
  await executar(api.descartar);
}

export type Tela = "inicio" | "entrada" | "processamento";

export function telaAtual(v: Visao | null): Tela {
  if (!v || v.estado === "NOVA") return "inicio";
  if (["TIPO_ESCOLHIDO", "ENTRADA_ESCOLHIDA", "RECEBENDO_UPLOAD"].includes(v.estado) || v.estado.startsWith("SEI_")) return "entrada";
  return "processamento";
}
