<script setup lang="ts">
import { TIPOS } from "../tipos";
import { escolherTipo, estado } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";

const opcoes = [TIPOS.PADS, TIPOS.SR];
</script>

<template>
  <main class="inicio">
    <div class="cabeca">
      <h1>Qual minuta você vai elaborar?</h1>
      <p>Depois desta escolha, você indicará como obter os autos.</p>
    </div>

    <div class="pastas">
      <button
        v-for="o in opcoes"
        :key="o.tipo"
        class="pasta"
        :style="{ '--cor': o.cor, '--cor-clara': o.corClara }"
        :disabled="estado.ocupado"
        :data-teste="'tipo-' + o.tipo"
        @click="escolherTipo(o.tipo)"
      >
        <span class="aba">{{ o.tipo }}</span>
        <span class="capa">
          <span class="titulo">{{ o.titulo }}</span>
          <span class="produz">{{ o.produz }}</span>
          <span class="peca">Peça central: <strong>{{ o.pecaCentral }}</strong></span>
        </span>
      </button>
    </div>

    <AvisoErro v-if="estado.erro" :erro="estado.erro" />

    <p class="rodape">
      O aplicativo prepara minutas de apoio. A decisão e a revisão integral do texto são sempre suas.
    </p>
  </main>
</template>

<style scoped>
.inicio {
  max-width: 980px;
  margin: 0 auto;
  padding: 3.5rem 2rem 2rem;
  display: grid;
  gap: 2.2rem;
}
.cabeca h1 { font-size: var(--t-2xl); }
.cabeca p { margin-top: 0.5rem; color: var(--tinta-suave); }

.pastas {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1.8rem;
}

.pasta {
  text-align: left;
  border: 0;
  padding: 0;
  background: none;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}
.aba {
  background: var(--cor);
  color: #fff;
  font-weight: 600;
  font-size: var(--t-sm);
  letter-spacing: 0.04em;
  padding: 0.35rem 1.2rem 0.5rem;
  border-radius: 8px 8px 0 0;
  margin-bottom: -4px;
}
.capa {
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
  width: 100%;
  min-height: 230px;
  padding: 1.6rem 1.6rem 1.4rem;
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-top: 6px solid var(--cor);
  border-radius: var(--raio-pasta);
  transition: border-color 120ms ease, background 120ms ease;
}
.pasta:hover:not(:disabled) .capa,
.pasta:focus-visible .capa {
  background: var(--cor-clara);
  border-color: var(--cor);
}
.pasta:focus-visible { outline: none; }
.pasta:focus-visible .capa { outline: 3px solid var(--foco); outline-offset: 3px; }
.titulo {
  font-family: var(--serif);
  font-size: var(--t-xl);
  font-weight: 600;
  line-height: 1.2;
  color: var(--cor);
}
.produz { color: var(--tinta); max-width: 44ch; }
.peca {
  margin-top: auto;
  padding-top: 0.8rem;
  border-top: 1px dashed var(--linha-forte);
  font-size: var(--t-sm);
  color: var(--tinta-suave);
}
.peca strong { color: var(--tinta); font-weight: 600; }

.rodape { font-size: var(--t-sm); color: var(--tinta-suave); max-width: 70ch; }
</style>
