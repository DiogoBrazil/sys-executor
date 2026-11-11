<script setup lang="ts">
// Modal (centro) ou gaveta (lateral direita). O conteúdo fica no DOM mesmo
// fechado (v-show), para não perder o estado dos campos e para os testes de
// tela continuarem achando os elementos. Fecha com Esc, com o botão ou com
// um clique no fundo; ao abrir, o foco vai para o botão Fechar e, ao fechar,
// volta para quem abriu.
import { nextTick, onBeforeUnmount, ref, useId, watch } from "vue";
import Icone from "./Icone.vue";

const props = withDefaults(
  defineProps<{ aberta: boolean; titulo: string; tipo?: "modal" | "gaveta"; largura?: "m" | "g"; teste?: string }>(),
  { tipo: "modal", largura: "m", teste: undefined },
);
const emit = defineEmits<{ fechar: [] }>();

const fechar = ref<HTMLButtonElement | null>(null);
const id = "janela-" + useId();
let origem: HTMLElement | null = null;

function tecla(e: KeyboardEvent) {
  if (e.key === "Escape" && props.aberta) {
    e.stopPropagation();
    emit("fechar");
  }
}

watch(
  () => props.aberta,
  (a) => {
    if (typeof document === "undefined") return;
    if (a) {
      origem = document.activeElement as HTMLElement | null;
      document.addEventListener("keydown", tecla);
      document.body.classList.add("com-janela");
      void nextTick(() => fechar.value?.focus());
    } else {
      document.removeEventListener("keydown", tecla);
      document.body.classList.remove("com-janela");
      origem?.focus?.();
      origem = null;
    }
  },
);
onBeforeUnmount(() => {
  document.removeEventListener("keydown", tecla);
  if (props.aberta) document.body.classList.remove("com-janela");
});
</script>

<template>
  <Transition :name="tipo === 'gaveta' ? 'gaveta' : 'modal'">
    <div v-show="aberta" class="fundo-janela" :class="tipo" @click.self="emit('fechar')">
      <div class="janela" :class="['l-' + largura]" role="dialog" aria-modal="true" :aria-labelledby="id" :data-teste="teste">
        <header class="cab">
          <h2 :id="id">{{ titulo }}</h2>
          <slot name="acoes" />
          <button ref="fechar" type="button" class="botao icone" aria-label="Fechar" title="Fechar (Esc)" @click="emit('fechar')">
            <Icone nome="fechar" />
          </button>
        </header>
        <div class="corpo-janela"><slot /></div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.fundo-janela {
  position: fixed;
  inset: 0;
  z-index: 30;
  background: rgba(27, 36, 48, 0.42);
  display: grid;
  place-items: center;
  padding: 1rem;
}
.fundo-janela.gaveta { place-items: stretch end; padding: 0; }
.janela {
  background: var(--superficie);
  border-radius: var(--r-md);
  box-shadow: var(--sombra-2), 0 24px 60px -20px rgba(27, 36, 48, 0.45);
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  max-height: calc(100vh - 2rem);
  width: min(720px, 100%);
  overflow: hidden;
}
.janela.l-g { width: min(1180px, 100%); }
.gaveta .janela { width: min(560px, 100vw); max-height: none; height: 100vh; border-radius: var(--r-md) 0 0 var(--r-md); }
.gaveta .janela.l-g { width: min(760px, 100vw); }
.cab {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.8rem 0.9rem 0.8rem 1.3rem;
  border-bottom: 1px solid var(--linha);
  background: var(--superficie);
}
.cab h2 { font-size: var(--t-lg); margin-right: auto; }
.corpo-janela { overflow: auto; padding: 1.1rem 1.3rem 1.4rem; overscroll-behavior: contain; }

.modal-enter-active, .modal-leave-active { transition: opacity var(--dur-2) var(--ease); }
.modal-enter-active .janela, .modal-leave-active .janela { transition: transform var(--dur-3) var(--ease), opacity var(--dur-2) var(--ease); }
.modal-enter-from, .modal-leave-to { opacity: 0; }
.modal-enter-from .janela, .modal-leave-to .janela { transform: translateY(8px) scale(0.98); opacity: 0; }
.gaveta-enter-active, .gaveta-leave-active { transition: opacity var(--dur-2) var(--ease); }
.gaveta-enter-active .janela, .gaveta-leave-active .janela { transition: transform var(--dur-3) var(--ease); }
.gaveta-enter-from, .gaveta-leave-to { opacity: 0; }
.gaveta-enter-from .janela, .gaveta-leave-to .janela { transform: translateX(100%); }

@media (max-width: 640px) {
  .fundo-janela { padding: 0; place-items: stretch; }
  .janela, .janela.l-g { width: 100%; max-height: none; height: 100vh; border-radius: 0; }
}
</style>
