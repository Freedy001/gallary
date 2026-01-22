import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import {fileURLToPath, URL} from 'node:url'

// 提取公共的 proxy 配置
const proxyConfig = {
  '/api': {
    target: 'http://localhost:9099',
    ws: true,
    changeOrigin: true
  },
  '/health': {
    target: 'http://localhost:9099',
    changeOrigin: true
  },
  '/static': {
    target: 'http://localhost:9099',
    changeOrigin: true
  },
  '/resouse': {
    target: 'http://localhost:9099',
    changeOrigin: true
  }
}

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: proxyConfig
  },
  preview: {
    host: '0.0.0.0',
    port: 8080,
    proxy: proxyConfig
  }
})