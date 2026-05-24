import { fileURLToPath, URL } from 'node:url'

import react from '@vitejs/plugin-react'
import { visualizer } from 'rollup-plugin-visualizer'
import { defineConfig } from 'vite'

const analyzeBundle = process.env.BUILD_STATS === 'true'

export default defineConfig({
  plugins: [
    react(),
    ...(analyzeBundle
      ? [
          visualizer({
            brotliSize: true,
            filename: 'stats.html',
            gzipSize: true,
            open: false,
            template: 'treemap',
          }),
        ]
      : []),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'es2020',
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (
            id.includes('node_modules/react-dom') ||
            id.includes('node_modules/react/') ||
            id.includes('node_modules/react-router')
          ) {
            return 'vendor'
          }
          if (id.includes('node_modules/antd') || id.includes('node_modules/@ant-design/icons')) {
            return 'antd'
          }
          if (id.includes('node_modules/@tanstack/react-query')) {
            return 'query'
          }
        },
      },
    },
  },
})
