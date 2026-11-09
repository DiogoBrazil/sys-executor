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

import * as Modelos from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/modelosservice";
import type { VisaoModelos } from "../bindings/github.com/DiogoBrazil/sys-executor/internal/app/models";
import { ModelosVisao } from "../bindings/events";

export type { VisaoModelos };

export const modelos = {
  estado: (): Promise<VisaoModelos> => Modelos.Estado(),
  baixar: (id: string): Promise<VisaoModelos> => Modelos.Baixar(id),
  importar: (): Promise<VisaoModelos> => Modelos.Importar(),
  cancelar: (): Promise<VisaoModelos> => Modelos.CancelarAndamento(),
  remover: (id: string): Promise<VisaoModelos> => Modelos.Remover(id),
  selecionar: (id: string): Promise<VisaoModelos> => Modelos.Selecionar(id),
  testarGPU: (): Promise<VisaoModelos> => Modelos.TestarGPU(),
  usarSoCPU: (): Promise<VisaoModelos> => Modelos.UsarSoCPU(),
  aoMudar(cb: (v: VisaoModelos) => void): () => void {
    return Events.On(ModelosVisao, (ev) => cb(ev.data as VisaoModelos));
  },
};

export const credenciais = {
  informar: (pedidoID: string, usuario: string, senha: string, orgao: string, processo: string): Promise<Resposta> =>
    Tarefa.InformarCredenciais(pedidoID, usuario, senha, orgao, processo),
  segredo: (pedidoID: string, valor: string): Promise<Resposta> => Tarefa.InformarSegredo(pedidoID, valor),
};
