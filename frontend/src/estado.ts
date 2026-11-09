// Estado da interface: espelha a visão da tarefa enviada pelo Go.
// Nada aqui é persistido (sem localStorage): a tarefa vive só na sessão.
import { reactive } from "vue";
import { api, credenciais, modelos as modelosApi, responder as responderApi, tarefa, type Resposta, type Visao, type VisaoModelos } from "./api";

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
  modelos: null as VisaoModelos | null,
  verModelos: false, // tela Modelos aberta por cima da atual
});

function aplicar(r: Resposta) {
  estado.visao = r.visao ?? null;
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
  api.aoMudar((v) => {
    estado.visao = v;
  });
  modelosApi.aoMudar((m) => {
    estado.modelos = m;
  });
  await executar(api.atual);
  await atualizarModelos();
}

// ---- modelos de IA ----
export async function atualizarModelos() {
  try {
    estado.modelos = await modelosApi.estado();
  } catch {
    estado.modelos = null;
  }
}
async function comandoModelos(f: () => Promise<VisaoModelos>) {
  try {
    estado.modelos = await f();
  } catch {
    /* a visão anterior continua; o próximo evento atualiza */
  }
}
export const abrirModelos = async () => {
  estado.verModelos = true;
  await atualizarModelos();
};
export const fecharModelos = () => {
  estado.verModelos = false;
};
export const baixarModelo = (id: string) => comandoModelos(() => modelosApi.baixar(id));
export const importarModelo = () => comandoModelos(modelosApi.importar);
export const cancelarAndamento = () => comandoModelos(modelosApi.cancelar);
export const removerModelo = (id: string) => comandoModelos(() => modelosApi.remover(id));
export const selecionarModelo = (id: string) => comandoModelos(() => modelosApi.selecionar(id));
export const testarGPU = () => comandoModelos(modelosApi.testarGPU);
export const usarSoCPU = () => comandoModelos(modelosApi.usarSoCPU);

export async function escolherTipo(tipo: "PADS" | "SR") {
  if (!estado.visao || estado.visao.terminal) {
    if (estado.visao?.terminal) await executar(api.descartar);
    if (!(await executar(api.iniciar))) return;
  }
  await executar(() => api.escolherTipo(tipo));
}

export async function enviarPDF() {
  if (estado.visao?.estado === "TIPO_ESCOLHIDO") {
    if (!(await executar(() => api.escolherEntrada("UPLOAD")))) return;
  }
  await executar(api.selecionarPDF);
}

export const responder = (perguntaID: string, opcaoID: string, valores: Record<string, string> = {}) =>
  executar(() => responderApi(perguntaID, opcaoID, valores));
export const revisar = (perguntaID: string) => executar(() => tarefa.revisar(perguntaID));
export const retomar = () => executar(tarefa.retomar);
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
