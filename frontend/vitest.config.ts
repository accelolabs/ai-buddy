/// <reference types="vitest" />
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'happy-dom',
    setupFiles: ['./setupTests.ts'],
    passWithNoTests: true,
    css: false,
    coverage: {
      provider: 'v8',
      reporter: ['text'],
      exclude: ['node_modules/', './setupTests.ts'],
    },
  },
});
