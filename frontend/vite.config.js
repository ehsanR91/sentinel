import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  css: {
    preprocessorOptions: {
      scss: {
        // Silence deprecation warnings from Bootstrap SCSS (these are non-breaking)
        // Bootstrap 5 uses deprecated Sass syntax that will be updated in future versions
        api: 'modern-compiler',
        silenceDeprecations: [
          'if-function',
          'global-builtin',
          'color-functions',
          'mixed-decls',
          'slash-div',
          'calc',
          'legacy-js-api',
          'import'
        ]
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://127.0.0.1:8080',
        ws: true
      }
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'static',
    indexHtml: 'public/index.html'
  }
})
