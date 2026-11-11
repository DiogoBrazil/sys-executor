<script setup lang="ts">
// Ícones de traço em SVG embutido (sem dependência; funcionam offline).
// Herdam a cor do texto (currentColor) e são decorativos (aria-hidden).
import { computed } from "vue";

export type NomeIcone =
  | "check" | "alerta" | "info" | "erro" | "pdf" | "pasta" | "pessoa" | "buscar" | "seta" | "chevron"
  | "lixeira" | "subir" | "descer" | "engrenagem" | "fechar" | "cadeado" | "relogio" | "mais" | "lampada" | "documento" | "expandir" | "ajuda";

const props = withDefaults(defineProps<{ nome: NomeIcone; tamanho?: number }>(), { tamanho: 18 });

const D: Record<NomeIcone, string> = {
  check: "M5 12.5l4.2 4.2L19 7",
  alerta: "M12 3.5L2.8 19.5h18.4L12 3.5zM12 10v4.2M12 17.2v.1",
  info: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 11v5.5M12 7.8v.1",
  erro: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM9 9l6 6M15 9l-6 6",
  pdf: "M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5zM14 3v5h5M8.5 13h7M8.5 16.5h5",
  documento: "M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5zM14 3v5h5",
  pasta: "M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z",
  pessoa: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM4.5 20.5c.8-3.6 3.8-5.5 7.5-5.5s6.7 1.9 7.5 5.5",
  buscar: "M10.5 18a7.5 7.5 0 1 0 0-15 7.5 7.5 0 0 0 0 15zM16 16l5 5",
  seta: "M5 12h14M13 6l6 6-6 6",
  chevron: "M9 6l6 6-6 6",
  lixeira: "M4 7h16M9.5 7V4.5h5V7M6.5 7l1 13h9l1-13M10 11v5.5M14 11v5.5",
  subir: "M12 19V5M6 11l6-6 6 6",
  descer: "M12 5v14M6 13l6 6 6-6",
  engrenagem: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 13.5l1.6 1.2-2 3.4-1.9-.7a7 7 0 0 1-2 1.2L14.8 21h-4l-.3-2.4a7 7 0 0 1-2-1.2l-1.9.7-2-3.4 1.6-1.2a7 7 0 0 1 0-2.4L4.6 9.9l2-3.4 1.9.7a7 7 0 0 1 2-1.2L10.8 3h4l.3 2.4a7 7 0 0 1 2 1.2l1.9-.7 2 3.4-1.6 1.2a7 7 0 0 1 0 2.4z",
  fechar: "M6 6l12 12M18 6L6 18",
  cadeado: "M6.5 11h11a1 1 0 0 1 1 1v7.5a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1V12a1 1 0 0 1 1-1zM8.5 11V8a3.5 3.5 0 0 1 7 0v3",
  relogio: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 7.5V12l3 2",
  mais: "M12 5v14M5 12h14",
  expandir: "M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5",
  ajuda: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM9.6 9.3a2.5 2.5 0 0 1 4.8.9c0 1.7-2.4 2.2-2.4 3.6M12 17v.1",
  lampada: "M9 18h6M10 21h4M12 3a6 6 0 0 0-3.5 10.9c.6.5 1 1.2 1 2.1h5c0-.9.4-1.6 1-2.1A6 6 0 0 0 12 3z",
};
const d = computed(() => D[props.nome]);
</script>

<template>
  <svg class="icone" :width="tamanho" :height="tamanho" viewBox="0 0 24 24" fill="none" stroke="currentColor"
    stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
    <path :d="d" />
  </svg>
</template>

<style scoped>
.icone { flex: none; display: inline-block; vertical-align: -0.2em; }
</style>
