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
