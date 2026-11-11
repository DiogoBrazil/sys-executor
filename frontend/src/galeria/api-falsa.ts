// Galeria de telas (só desenvolvimento): substitui src/api.ts por um backend
// parado que devolve a visão do cenário escolhido. Dados sintéticos.
import type { Resposta } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/app/models";
import type { Visao } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import type { RespostaBusca, RespostaDestaque, RespostaEndereco, VisaoIA, DispositivoVisao } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/app/models";
import type { Cadastro } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

export type { Resposta, Visao, RespostaBusca, RespostaDestaque, VisaoIA, Cadastro, DispositivoVisao };

export const palco = { visao: null as any, ia: null as any };
const r = (): Promise<Resposta> => Promise.resolve({ visao: palco.visao, erro: null } as any);

export const api = {
  atual: r, iniciar: r, escolherTipo: (_t: string) => r(), escolherEntrada: (_e: string) => r(), voltarParaEntrada: r,
  selecionarPDF: r, cancelar: r, descartar: r,
  aoMudar: (_cb: (v: Visao | null) => void) => () => {},
};
export const fontes = {
  enderecoPDF: (): Promise<RespostaEndereco> => Promise.resolve({ url: "", erro: { codigo: "INT-001", titulo: "Galeria", mensagem: "Sem PDF na galeria.", acao: "" } } as any),
  destaque: (_p: number, _i: number, _f: number): Promise<RespostaDestaque> => Promise.resolve({ retangulos: [], erro: null } as any),
  buscar: (_c: string): Promise<RespostaBusca> => Promise.resolve({ ocorrencias: [], erro: null } as any),
};
export const responder = (_p: string, _o: string, _v: Record<string, string> = {}) => r();
export const tarefa = {
  revisar: (_p: string) => r(), retomar: r, solicitarAjuste: (_t: string) => r(), aprovar: r,
  salvarMinuta: async (formato: string) => ({ ...(await r()), salvo: true, formato }),
};
const i = (): Promise<VisaoIA> => Promise.resolve(palco.ia);
export const ia = {
  estado: i, salvarChave: (_c: string) => i(), removerChave: i, verificar: i, selecionarModelo: (_m: string) => i(),
  testarModelo: (_m: string) => i(), abrirPaginaChaves: i, selecionarProvedor: (_p: string) => i(), entrarChatGPT: i,
  cancelarEntradaChatGPT: i, desconectarChatGPT: i, atualizarModelosChatGPT: i, selecionarModeloChatGPT: (_m: string) => i(),
  testarModeloChatGPT: (_m: string) => i(), abrirUsoChatGPT: i, aoMudar: (_cb: (v: VisaoIA) => void) => () => {},
};
export const credenciais = {
  informar: (_a: string, _b: string, _c: string, _d: string, _e: string) => r(),
  segredo: (_a: string, _b: string) => r(),
};
export const cadastro = {
  salvar: (_c: Cadastro) => r(), anexar: (_p: string) => r(), anexarSEI: (_p: string, _d: string[]) => r(),
  remover: (_p: string, _a: string) => r(), mover: (_p: string, _a: string, _d: number) => r(), concluir: r,
  catalogo: (): Promise<DispositivoVisao[]> => Promise.resolve([
    { id: "RDPM-17-XI", diploma: "RDPM", artigo: 17, inciso: "XI", alinea: "", texto: "empregar força ou medida desnecessária em ato de serviço, ainda que não resulte dano", natureza: "GRAVE", referencia: "art. 17, inciso XI, do RDPM" },
    { id: "RDPM-16-V", diploma: "RDPM", artigo: 16, inciso: "V", alinea: "", texto: "faltar ao serviço", natureza: "MEDIA", referencia: "art. 16, inciso V, do RDPM" },
  ] as any),
};
