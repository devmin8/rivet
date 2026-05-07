import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import vueDevTools from "vite-plugin-vue-devtools";
import Components from "unplugin-vue-components/vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [Components(), vue(), tailwindcss(), vueDevTools()],
  server: {
    proxy: {
      "/api": {
        target: process.env.VITE_RIVET_PROXY_TARGET ?? "http://rivet-server.localhost",
        changeOrigin: true,
      },
    },
  },
  resolve: {
    alias: {
      "~": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
});
