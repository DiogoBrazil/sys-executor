<script setup lang="ts">
import { computed } from "vue";
import { acessarSEI, cadastrarPecas, estado, voltarAoInicio, voltarParaEntrada } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";
import PainelSEI from "../components/PainelSEI.vue";
import Icone from "../components/ui/Icone.vue";
import Montagem from "./Montagem.vue";

const aguardandoArquivo = computed(() => estado.visao?.estado === "RECEBENDO_UPLOAD");
const emCadastro = computed(() => aguardandoArquivo.value && !!estado.visao?.cadastro);
const noSEI = computed(() => !!estado.visao?.estado.startsWith("SEI_"));
</script>

<template>
  <main class="entrada" :class="{ cadastro: emCadastro }">
    <template v-if="!emCadastro">
      <nav class="trilha-inicial" aria-label="Etapas iniciais">
        <span class="feito"><span class="bola"><Icone nome="check" :tamanho="14" /></span>Tipo de minuta</span>
        <span class="traco feito"></span>
        <span class="atual" aria-current="step"><span class="bola">2</span>Origem dos autos</span>
        <span class="traco"></span>
        <span><span class="bola">3</span>Processamento</span>
      </nav>
      <div class="cabeca">
        <h1>Como você vai trazer os autos?</h1>
        <p>Os dois caminhos levam ao mesmo processamento. Escolha o que for mais prático para este processo.</p>
      </div>
    </template>

    <PainelSEI v-if="noSEI" />

    <template v-else-if="!aguardandoArquivo">
      <div class="opcoes">
        <section class="opcao">
          <div class="topo-opcao">
            <span class="ic"><Icone nome="cadeado" :tamanho="26" /></span>
            <div>
              <h2>Buscar no SEI/RO</h2>
              <span class="req">Requer internet</span>
            </div>
          </div>
          <p class="desc">
            Você entra com seu usuário, senha e código de verificação. O aplicativo localiza o processo, gera o PDF completo e separa os
            documentos para você escolher os de cada peça.
          </p>
          <ul class="vantagens">
            <li><Icone nome="check" :tamanho="16" />A senha não é guardada</li>
            <li><Icone nome="check" :tamanho="16" />Só os documentos escolhidos entram nos autos</li>
          </ul>
          <button class="botao primario grande" :disabled="estado.ocupado" data-teste="entrada-SEI" @click="acessarSEI">
            Entrar no SEI<Icone nome="seta" :tamanho="16" />
          </button>
        </section>
        <section class="opcao">
          <div class="topo-opcao">
            <span class="ic"><Icone nome="pasta" :tamanho="26" /></span>
            <div>
              <h2>Cadastrar as peças</h2>
              <span class="req">Sem senha · a análise por IA requer internet</span>
            </div>
          </div>
          <p class="desc">
            Você indica cada peça (Memorando ou Portaria, defesa, depoimentos, alegações e outros documentos) e anexa os PDFs de cada uma,
            na ordem dos autos.
          </p>
          <ul class="vantagens">
            <li><Icone nome="check" :tamanho="16" />Vários PDFs por peça</li>
            <li><Icone nome="check" :tamanho="16" />O tipo que você indica é o que vale</li>
          </ul>
          <button class="botao grande contorno" :disabled="estado.ocupado" data-teste="entrada-UPLOAD" @click="cadastrarPecas">
            Cadastrar peças<Icone nome="seta" :tamanho="16" />
          </button>
        </section>
      </div>
      <p class="nota-cifra"><Icone nome="cadeado" :tamanho="18" />Os arquivos são copiados de forma cifrada para a área temporária desta tarefa e apagados ao final.</p>
    </template>

    <Montagem v-else-if="emCadastro" :key="estado.visao!.id" />

    <template v-if="!emCadastro">
      <AvisoErro v-if="estado.erro" :erro="estado.erro" />
      <AvisoErro v-else-if="!noSEI && estado.visao?.erro" :erro="estado.visao.erro" data-teste="erro-arquivo" />
    </template>

    <div class="links">
      <button v-if="emCadastro" class="botao discreto" :disabled="estado.ocupado" @click="voltarParaEntrada">Escolher outro caminho</button>
      <button class="botao discreto" :disabled="estado.ocupado" data-teste="voltar-inicio" @click="voltarAoInicio">Trocar o tipo de minuta</button>
    </div>
  </main>
</template>

<style scoped>
.entrada {
  max-width: 1040px;
  margin: 0 auto;
  padding: 3rem 2rem 2rem;
  display: grid;
  gap: 2rem;
}
.entrada.cadastro { max-width: 1240px; padding-top: 2.4rem; }
.trilha-inicial { display: flex; align-items: center; justify-content: center; gap: 0.8rem; flex-wrap: wrap; font-size: var(--t-sm); color: var(--tinta-suave); }
.trilha-inicial > span:not(.traco) { display: inline-flex; align-items: center; gap: 0.5rem; }
.bola { width: 1.6rem; height: 1.6rem; border-radius: 50%; border: 2px solid var(--linha); display: inline-grid; place-items: center; font-size: var(--t-xs); font-weight: 600; }
.feito .bola { background: var(--acento); border-color: var(--acento); color: #fff; }
.atual { color: var(--tinta); font-weight: 600; }
.atual .bola { border-color: var(--acento); color: var(--acento); }
.traco { width: 3rem; height: 2px; background: var(--linha); }
.traco.feito { background: var(--acento); }
.cabeca { text-align: center; display: grid; gap: 0.5rem; justify-items: center; }
.cabeca h1 { font-size: 2.2rem; }
.cabeca p { color: var(--tinta-suave); font-size: 1.05rem; max-width: 36rem; }
.opcoes { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 22rem), 1fr)); gap: 1.5rem; }
.opcao {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.8rem;
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: 14px;
  box-shadow: var(--sombra-2);
}
.topo-opcao { display: flex; align-items: center; gap: 0.9rem; }
.topo-opcao h2 { font-size: 1.4rem; }
.ic { width: 3.3rem; height: 3.3rem; border-radius: 12px; background: var(--acento-claro); color: var(--acento); display: grid; place-items: center; flex: none; }
.req { font-size: var(--t-sm); color: var(--tinta-suave); }
.desc { color: var(--tinta-suave); }
.vantagens { list-style: none; margin: 0 0 auto; padding: 0; display: grid; gap: 0.45rem; font-size: var(--t-sm); }
.vantagens li { display: flex; gap: 0.5rem; align-items: center; }
.vantagens .icone { color: var(--ok); }
.botao.grande { align-self: flex-start; min-height: 3rem; padding-inline: 1.3rem; font-weight: 600; }
.botao.contorno { color: var(--acento); border: 1.5px solid var(--acento); }
.botao.contorno:hover:not(:disabled) { background: var(--acento-claro); border-color: var(--acento); }
.nota-cifra {
  display: flex; align-items: center; justify-content: center; gap: 0.6rem; text-align: center;
  color: var(--tinta-suave); font-size: var(--t-sm); background: var(--superficie-2); border: 1px solid var(--linha); border-radius: var(--r-md); padding: 0.8rem 1rem;
}
.links { display: flex; justify-content: center; gap: 1.6rem; flex-wrap: wrap; }
@media (max-width: 640px) { .entrada { padding: 1.4rem 1rem; } .cabeca h1 { font-size: 1.7rem; } }
</style>
