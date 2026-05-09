import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'
import { resolve } from 'path'

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      registerType: 'prompt',
      injectRegister: false,
      filename: 'sw.js',
      strategies: 'generateSW',
      includeAssets: [
        'favicon.svg',
        'favicon.ico',
        'browserconfig.xml',
        'offline.html',
        'icons/*',
        'icons/*.png',
        'icons/*.svg'
      ],
      workbox: {
        cleanupOutdatedCaches: true,
        sourcemap: false,
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//, /\/__\/auth/],
        runtimeCaching: [
          {
            urlPattern: ({ request }) => request.destination === 'document',
            handler: 'NetworkFirst',
            options: {
              cacheName: 'sc-pages',
              networkTimeoutSeconds: 5,
              expiration: { maxEntries: 20, maxAgeSeconds: 60 * 60 * 24 }
            }
          },
          {
            urlPattern: /\/api\/v1\//,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'sc-api',
              networkTimeoutSeconds: 6,
              cacheableResponse: { statuses: [0, 200] },
              expiration: { maxEntries: 60, maxAgeSeconds: 60 * 60 * 12 }
            }
          },
          {
            urlPattern: /\.(?:js|css|woff2?)$/,
            handler: 'CacheFirst',
            options: {
              cacheName: 'sc-static-resources',
              expiration: { maxEntries: 80, maxAgeSeconds: 60 * 60 * 24 * 30 }
            }
          },
          {
            urlPattern: /\.(?:png|jpg|jpeg|svg|webp|avif)$/,
            handler: 'CacheFirst',
            options: {
              cacheName: 'sc-images',
              expiration: { maxEntries: 120, maxAgeSeconds: 60 * 60 * 24 * 30 }
            }
          }
        ]
      }
    })
  ],
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
        target: 'http://127.0.0.1:8888',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://127.0.0.1:8888',
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
