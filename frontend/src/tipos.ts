// Textos e cores de cada procedimento (V4 §30.1–30.2).
export interface InfoTipo {
  tipo: "PADS" | "SR";
  titulo: string;
  produz: string;
  pecaCentral: string;
  cor: string;
  corClara: string;
}

export const TIPOS: Record<"PADS" | "SR", InfoTipo> = {
  PADS: {
    tipo: "PADS",
    titulo: "Julgamento de PADS",
    produz: "Minuta que julga cada acusação do Memorando Disciplinar: procedência, procedência parcial ou improcedência.",
    pecaCentral: "Memorando Disciplinar",
    cor: "var(--pads)",
    corClara: "var(--pads-claro)",
  },
  SR: {
    tipo: "SR",
    titulo: "Relatório de Sindicância Regular",
    produz: "Minuta do relatório que aponta, para cada fato e sindicado, indícios de crime comum, crime militar ou transgressão.",
    pecaCentral: "Portaria de instauração",
    cor: "var(--sr)",
    corClara: "var(--sr-claro)",
  },
};
