<script setup lang="ts">
import type { PainelAnalise } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import Icone from "./ui/Icone.vue";
import Recolhivel from "./ui/Recolhivel.vue";

defineProps<{ painel: PainelAnalise }>();
const emit = defineEmits<{ abrir: [pagina: number, inicio: number, fim: number] }>();

// Situações confirmadas pelo usuário em verde; as que aguardam, em âmbar.
const CONFIRMADAS = /confirmad|congelad|admitid|reconhecid|aceit/i;
const AGUARDANDO = /aguard|propost|pendent/i;
const classeSituacao = (s: string) => (CONFIRMADAS.test(s) ? "ok" : AGUARDANDO.test(s) ? "alerta" : "");
</script>

<template>
  <section class="analise" aria-label="Andamento da análise" data-teste="painel-analise">
    <Recolhivel v-for="(s, i) in painel.secoes" :key="s.titulo" :titulo="s.titulo" :contador="s.itens.length" :aberto="i === 0 || painel.secoes.length <= 3" class="secao">
      <ul role="list">
        <li v-for="(it, k) in s.itens" :key="k">
          <div class="cabeca">
            <strong>{{ it.titulo }}</strong>
            <span v-if="it.situacao" class="selo" :class="classeSituacao(it.situacao)">{{ it.situacao }}</span>
          </div>
          <p v-for="(l, j) in it.linhas" :key="j" class="linha">{{ l }}</p>
          <div v-if="it.fontes?.length" class="fontes">
            <button v-for="(f, j) in it.fontes" :key="j" type="button" class="chip fonte" :title="f.trecho" @click="emit('abrir', f.pagina, f.inicio, f.fim)">
              <Icone nome="pdf" :tamanho="14" />{{ f.rotulo || "Autos" }} — pág. {{ f.pagina }}
            </button>
          </div>
        </li>
      </ul>
    </Recolhivel>
    <p class="nota"><Icone nome="info" :tamanho="15" />As propostas da IA só valem depois da sua confirmação.</p>
  </section>
</template>

<style scoped>
.analise { display: grid; gap: 0.9rem; }
.secao { border-bottom: 1px solid var(--linha); padding-bottom: 0.6rem; }
ul { list-style: none; margin: 0.2rem 0 0; padding: 0; display: grid; gap: 0.6rem; }
li { border-left: 3px solid var(--acento-claro); padding: 0.2rem 0 0.2rem 0.8rem; display: grid; gap: 0.25rem; }
.cabeca { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; }
.linha { max-width: 80ch; font-size: var(--t-sm); }
.fontes { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.fonte { font-size: var(--t-xs); padding: 0.1rem 0.55rem; }
.fonte .icone { color: var(--acento); }
.nota { display: flex; gap: 0.4rem; align-items: center; font-size: var(--t-xs); color: var(--tinta-suave); }
</style>
