import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import wails from '@wailsio/runtime/plugins/vite'

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [vue(), wails('./bindings')],
  test: {
    // @wailsio/runtime starts a window timer on import under jsdom; it fires after a fast test file's
    // environment is gone ("window is not defined"), so only DOM tests opt into jsdom per file
    environment: 'node',
  },
})
