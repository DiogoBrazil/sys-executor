<script setup lang="ts">
// Indicador de carregamento (anel girando + texto), usado ao abrir os autos
// em PDF. Anunciado aos leitores de tela (role=status).
withDefaults(defineProps<{ texto?: string; sobreposto?: boolean }>(), { texto: "Carregando…", sobreposto: false });
</script>

<template>
  <div class="carregando" :class="{ sobreposto }" role="status" data-teste="carregando">
    <span class="anel" aria-hidden="true"></span>
    <span class="texto">{{ texto }}</span>
  </div>
</template>

<style scoped>
.carregando { display: inline-flex; align-items: center; gap: 0.7rem; color: var(--tinta-suave); font-size: var(--t-sm); }
.carregando.sobreposto {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  background: rgba(236, 238, 233, 0.82);
  z-index: 2;
  font-size: var(--t-md);
}
.anel {
  width: 1.6rem;
  height: 1.6rem;
  border-radius: 50%;
  border: 3px solid var(--linha);
  border-top-color: var(--acento);
  animation: gira-anel 0.8s linear infinite;
}
.sobreposto .anel { width: 2.6rem; height: 2.6rem; border-width: 4px; }
@keyframes gira-anel { to { transform: rotate(360deg); } }
</style>
