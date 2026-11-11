// Galeria de telas (só desenvolvimento; não entra no executável): cada tela
// com dados sintéticos, escolhida por #cenario. `npm run galeria`.
import { defineConfig, type Plugin } from "vite";
import vue from "@vitejs/plugin-vue";
import { join } from "node:path";

const falsa = join(__dirname, "src", "galeria", "api-falsa.ts");
function apiFalsa(): Plugin {
  return {
    name: "api-falsa",
    enforce: "pre",
    resolveId(fonte, importador) {
      if (importador && !importador.includes("/galeria/") && /(^|\/)\.{1,2}\/(\.\.\/)*api$/.test(fonte)) return falsa;
      return null;
    },
  };
}

export default defineConfig({
  plugins: [apiFalsa(), vue()],
  server: { host: "127.0.0.1", port: 9311, strictPort: true, open: "/galeria.html" },
  preview: { host: "127.0.0.1", port: 9311, strictPort: true },
  build: { rollupOptions: { input: join(__dirname, "galeria.html") }, sourcemap: false, chunkSizeWarningLimit: 4000 },
});
