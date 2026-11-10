// Fronteira única com o backend (bindings Wails). Os testes substituem este módulo.
import { Events } from "@wailsio/runtime";
import * as Tarefa from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/tarefaservice";
import type { Resposta } from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/models";
import type { Visao } from "../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import { TarefaVisao } from "../bindings/events";

export type { Resposta, Visao };

export const api = {
  atual: (): Promise<Resposta> => Tarefa.Atual(),
  iniciar: (): Promise<Resposta> => Tarefa.Iniciar(),
  escolherTipo: (tipo: "PADS" | "SR"): Promise<Resposta> => Tarefa.EscolherTipo(tipo),
  escolherEntrada: (e: "SEI" | "UPLOAD"): Promise<Resposta> => Tarefa.EscolherEntrada(e),
  voltarParaEntrada: (): Promise<Resposta> => Tarefa.VoltarParaEntrada(),
  selecionarPDF: (): Promise<Resposta> => Tarefa.SelecionarPDF(),
  cancelar: (): Promise<Resposta> => Tarefa.Cancelar(),
  descartar: (): Promise<Resposta> => Tarefa.Descartar(),
  aoMudar(cb: (v: Visao | null) => void): () => void {
    return Events.On(TarefaVisao, (ev) => cb((ev.data as Visao | null) ?? null));
  },
};

import * as Fontes from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/fontesservice";
import type { RespostaBusca, RespostaDestaque, RespostaEndereco } from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/models";

export type { RespostaBusca, RespostaDestaque };

export const fontes = {
  enderecoPDF: (): Promise<RespostaEndereco> => Fontes.EnderecoPDF(),
  destaque: (pagina: number, inicio: number, fim: number): Promise<RespostaDestaque> => Fontes.Destaque(pagina, inicio, fim),
  buscar: (consulta: string): Promise<RespostaBusca> => Fontes.Buscar(consulta),
};

export const responder = (perguntaID: string, opcaoID: string, valores: Record<string, string> = {}): Promise<Resposta> =>
  Tarefa.Responder(perguntaID, opcaoID, valores);

export const tarefa = {
  revisar: (perguntaID: string): Promise<Resposta> => Tarefa.Revisar(perguntaID),
  retomar: (): Promise<Resposta> => Tarefa.Retomar(),
};

import * as IA from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/iaservice";
import type { VisaoIA } from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/models";
import { IaVisao } from "../bindings/events";

export type { VisaoIA };

// Configurações de IA (V4 §§47, 48): Gemini API (chave do usuário) e
// ChatGPT (plano do usuário). A chave só vai ao Go em salvarChave e nunca
// volta; nenhum token do ChatGPT chega aqui: VisaoIA não os contém.
export const ia = {
  estado: (): Promise<VisaoIA> => IA.Estado(),
  salvarChave: (chave: string): Promise<VisaoIA> => IA.SalvarChave(chave),
  removerChave: (): Promise<VisaoIA> => IA.RemoverChave(),
  verificar: (): Promise<VisaoIA> => IA.Verificar(),
  selecionarModelo: (id: string): Promise<VisaoIA> => IA.SelecionarModelo(id),
  testarModelo: (id: string): Promise<VisaoIA> => IA.TestarModelo(id),
  abrirPaginaChaves: (): Promise<VisaoIA> => IA.AbrirPaginaChaves(),
  selecionarProvedor: (p: string): Promise<VisaoIA> => IA.SelecionarProvedor(p),
  entrarChatGPT: (): Promise<VisaoIA> => IA.EntrarChatGPT(),
  cancelarEntradaChatGPT: (): Promise<VisaoIA> => IA.CancelarEntradaChatGPT(),
  desconectarChatGPT: (): Promise<VisaoIA> => IA.DesconectarChatGPT(),
  atualizarModelosChatGPT: (): Promise<VisaoIA> => IA.AtualizarModelosChatGPT(),
  selecionarModeloChatGPT: (id: string): Promise<VisaoIA> => IA.SelecionarModeloChatGPT(id),
  testarModeloChatGPT: (id: string): Promise<VisaoIA> => IA.TestarModeloChatGPT(id),
  abrirUsoChatGPT: (): Promise<VisaoIA> => IA.AbrirUsoChatGPT(),
  aoMudar(cb: (v: VisaoIA) => void): () => void {
    return Events.On(IaVisao, (ev) => cb(ev.data as VisaoIA));
  },
};

export const credenciais = {
  informar: (pedidoID: string, usuario: string, senha: string, orgao: string, processo: string): Promise<Resposta> =>
    Tarefa.InformarCredenciais(pedidoID, usuario, senha, orgao, processo),
  segredo: (pedidoID: string, valor: string): Promise<Resposta> => Tarefa.InformarSegredo(pedidoID, valor),
};

import type { DispositivoVisao } from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/models";
import type { Cadastro } from "../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

export type { Cadastro, DispositivoVisao };

// Cadastro de peças (V4 §48.4): os arquivos são escolhidos no diálogo nativo
// e lidos pelo Go; aqui chegam só nome, páginas e resumo do hash.
export const cadastro = {
  salvar: (c: Cadastro): Promise<Resposta> => Tarefa.SalvarCadastro(c),
  anexar: (pecaID: string): Promise<Resposta> => Tarefa.AnexarArquivos(pecaID),
  remover: (pecaID: string, arquivoID: string): Promise<Resposta> => Tarefa.RemoverArquivo(pecaID, arquivoID),
  mover: (pecaID: string, arquivoID: string, delta: number): Promise<Resposta> => Tarefa.MoverArquivo(pecaID, arquivoID, delta),
  concluir: (): Promise<Resposta> => Tarefa.ConcluirCadastro(),
  catalogo: (): Promise<DispositivoVisao[]> => Tarefa.CatalogoEnquadramento(),
};
