import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: {
    host: '127.0.0.1',
    allowedHosts: ['wayfinder-t1a.eastus2.cloudapp.azure.com'],
    port: 5173,
    strictPort: true,
    proxy: {
      '/mcp': 'http://127.0.0.1:8080',
      '/oauth': 'http://127.0.0.1:8080',
      '/.well-known': 'http://127.0.0.1:8080',
      '/api': 'http://127.0.0.1:8080',
      '/healthz': 'http://127.0.0.1:8080',
      '/readyz': 'http://127.0.0.1:8080',
    },
  },
});
