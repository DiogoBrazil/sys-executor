<script setup lang="ts">
import type { PainelAnalise } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

defineProps<{ painel: PainelAnalise }>();
const emit = defineEmits<{ abrir: [pagina: number, inicio: number, fim: number] }>();
</script>

<template>
  <section class="analise" aria-label="Andamento da análise" data-teste="painel-analise">
    <details v-for="(s, i) in painel.secoes" :key="s.titulo" :open="i === 0">
      <summary>{{ s.titulo }} <span class="conta">({{ s.itens.length }})</span></summary>
      <ul role="list">
        <li v-for="(it, k) in s.itens" :key="k">
          <div class="cabeca">
            <strong>{{ it.titulo }}</strong>
            <span v-if="it.situacao" class="sit">{{ it.situacao }}</span>
          </div>
          <p v-for="(l, j) in it.linhas" :key="j" class="linha">{{ l }}</p>
          <div v-if="it.fontes?.length" class="fontes">
            <button
              v-for="(f, j) in it.fontes"
              :key="j"
              type="button"
              class="fonte"
              :title="f.trecho"
              @click="emit('abrir', f.pagina, f.inicio, f.fim)"
            >
              {{ f.rotulo || "Autos" }} — pág. {{ f.pagina }}
            </button>
          </div>
        </li>
      </ul>
    </details>
    <p class="nota">As propostas da IA só valem depois da sua confirmação.</p>
  </section>
</template>

<style scoped>
.analise {
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  padding: 0.9rem 1.2rem;
  display: grid;
  gap: 0.6rem;
}
summary { cursor: pointer; font-weight: 600; }
.conta { font-weight: 400; color: var(--tinta-suave); }
ul { list-style: none; margin: 0.5rem 0 0; padding: 0; display: grid; gap: 0.7rem; }
li { border-left: 3px solid var(--linha); padding-left: 0.8rem; display: grid; gap: 0.25rem; }
.cabeca { display: flex; flex-wrap: wrap; gap: 0.6rem; align-items: baseline; }
.sit { font-size: var(--t-sm); color: var(--tinta-suave); }
.linha { max-width: 72ch; font-size: var(--t-sm); }
.fontes { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.fonte {
  border: 1px solid var(--linha-forte);
  background: var(--papel);
  border-radius: 999px;
  padding: 0.1rem 0.6rem;
  font-size: var(--t-xs);
  cursor: pointer;
  color: var(--tinta);
}
.nota { font-size: var(--t-xs); color: var(--tinta-suave); }
</style>
