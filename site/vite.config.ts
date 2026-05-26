import tailwindcss from '@tailwindcss/vite'
import { readFileSync } from 'node:fs'
import { defineConfig, type Plugin } from 'vite'

const installerPath = new URL('../scripts/install.sh', import.meta.url)

function rivetInstaller(): Plugin {
  return {
    name: 'rivet-installer',
    configureServer(server) {
      server.middlewares.use('/install.sh', (_req, res) => {
        res.setHeader('Content-Type', 'text/x-shellscript; charset=utf-8')
        res.end(readFileSync(installerPath))
      })
    },
    generateBundle() {
      this.emitFile({
        type: 'asset',
        fileName: 'install.sh',
        source: readFileSync(installerPath),
      })
    },
  }
}

export default defineConfig({
  plugins: [tailwindcss(), rivetInstaller()],
  build: {
    rollupOptions: {
      input: {
        main: 'index.html',
        docs: 'docs.html',
      },
    },
  },
})
