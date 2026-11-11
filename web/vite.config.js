import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const proxy = {
  '/v1': {
    target: process.env.BOOKSTORE_API_TARGET || 'http://127.0.0.1:8090',
    changeOrigin: true,
  },
}

export default defineConfig({
  plugins: [vue()],
  server: { host: '127.0.0.1', proxy },
  preview: { host: '127.0.0.1', proxy },
})
