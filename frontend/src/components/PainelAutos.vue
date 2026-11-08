<script setup lang="ts">
import { computed } from "vue";
import type { ResumoAutos } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

const props = defineProps<{ autos: ResumoAutos }>();
const emit = defineEmits<{ abrir: [pagina: number, inicio: number, fim: number] }>();

const ROTULO_SITUACAO: Record<string, string> = {
  LIDA: "lidas",
  OCR_OK: "por OCR",
  OCR_BAIXO: "OCR de baixa qualidade",
  ILEGIVEL: "ilegíveis",
  VAZIA: "em branco",
};

const cobertura = computed(() => {
  const c: Record<string, number> = {};
  for (const p of props.autos.paginas ?? []) c[p.situacao] = (c[p.situacao] ?? 0) + 1;
  return Object.keys(ROTULO_SITUACAO)
    .filter((k) => c[k])
    .map((k) => ({ chave: k, n: c[k], rotulo: ROTULO_SITUACAO[k] }));
});

const problemas = computed(() =>
  (props.autos.paginas ?? []).filter((p) => p.situacao === "ILEGIVEL" || p.situacao === "OCR_BAIXO"),
);

function paginas(a: number, b: number) {
  return a === b ? `p. ${a}` : `pp. ${a}–${b}`;
}
</script>

<template>
  <section class="autos" data-teste="painel-autos">
    <div class="cabeca">
      <h2>Autos lidos</h2>
      <p>{{ autos.totalPaginas }} páginas</p>
    </div>

    <ul class="cobertura" aria-label="Cobertura da leitura" data-teste="cobertura">
      <li v-for="c in cobertura" :key="c.chave" :class="c.chave">{{ c.n }} {{ c.rotulo }}</li>
    </ul>
    <p v-if="problemas.length" class="alerta" data-teste="paginas-problema">
      Páginas com leitura incerta:
      <button v-for="p in problemas" :key="p.numero" class="link" @click="emit('abrir', p.numero, 0, 0)">{{ p.numero }}</button>
    </p>

    <h3>Peças identificadas</h3>
    <ol class="pecas" data-teste="pecas">
      <li v-for="p in autos.pecas" :key="p.id" :class="{ central: p.id === autos.pecaCentral }">
        <button class="item" @click="emit('abrir', p.paginaInicial, 0, 0)">
          <span class="rotulo">
            {{ p.rotulo }}
            <em v-if="p.provavel" class="selo">provável, pela ordem dos autos</em>
            <em v-if="p.id === autos.pecaCentral" class="selo central">peça central</em>
          </span>
          <span class="meta">
            {{ paginas(p.paginaInicial, p.paginaFinal) }}<template v-if="p.sei"> · SEI {{ p.sei }}</template
            ><template v-else> · documento externo</template>
          </span>
          <span v-if="p.situacao === 'ILEGIVEL' || p.situacao === 'OCR_BAIXO'" class="selo alerta">leitura incerta</span>
        </button>
      </li>
    </ol>

    <template v-if="autos.midias?.length">
      <h3>Mídias mencionadas</h3>
      <p class="nota">
        Gravações, vídeos e links citados não fazem parte do PDF. O aplicativo vai pedir uma descrição ou
        transcrição antes de usar essas provas.
      </p>
      <ul class="midias" data-teste="midias">
        <li v-for="(m, i) in autos.midias" :key="i">
          <button class="item" @click="emit('abrir', m.pagina, m.inicio, m.fim)">
            <span class="rotulo">"{{ m.termo }}"</span>
            <span class="meta">p. {{ m.pagina }}</span>
          </button>
        </li>
      </ul>
    </template>
  </section>
</template>

<style scoped>
.autos {
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  padding: 1.3rem 1.4rem;
  display: grid;
  gap: 0.8rem;
}
.cabeca { display: flex; align-items: baseline; justify-content: space-between; }
.cabeca h2 { font-size: var(--t-lg); }
.cabeca p { color: var(--tinta-suave); }
h3 { font-family: var(--sans); font-size: var(--t-md); font-weight: 600; margin-top: 0.4rem; }
.cobertura { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; gap: 0.5rem; }
.cobertura li { font-size: var(--t-sm); padding: 0.2rem 0.6rem; border-radius: 999px; background: var(--papel); border: 1px solid var(--linha); }
.cobertura li.ILEGIVEL, .cobertura li.OCR_BAIXO { border-color: var(--alerta); color: var(--alerta); background: var(--alerta-claro); }
.alerta { color: var(--alerta); font-size: var(--t-sm); }
.link { border: 0; background: none; color: var(--foco); text-decoration: underline; padding: 0 0.2rem; }
.pecas, .midias { list-style: none; margin: 0; padding: 0; display: grid; gap: 0.3rem; }
.item {
  width: 100%;
  text-align: left;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.3rem 0.8rem;
  padding: 0.5rem 0.7rem;
  border: 1px solid transparent;
  border-radius: var(--raio);
  background: none;
}
.item:hover { border-color: var(--linha-forte); background: var(--papel); }
.central .item { border-color: var(--linha); }
.rotulo { font-weight: 500; }
.meta { color: var(--tinta-suave); font-size: var(--t-sm); }
.selo { font-style: normal; font-size: var(--t-xs); color: var(--tinta-suave); margin-left: 0.4rem; }
.selo.central { color: var(--tinta); font-weight: 600; }
.selo.alerta { color: var(--alerta); }
.nota { color: var(--tinta-suave); font-size: var(--t-sm); max-width: 70ch; }
</style>
