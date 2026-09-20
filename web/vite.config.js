import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  resolve: {
    alias: {
      '~': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/admin': { target: 'http://127.0.0.1:8083', changeOrigin: true },
    },
  },
})
