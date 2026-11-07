import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';

export default defineConfig({
  // JSX in the DOM tests needs transforming; the node-only suite never did.
  plugins: [react()],
  test: {
    // Pure logic runs in node. The React files that constitute the user-visible
    // fix opt into happy-dom with a @vitest-environment docblock, which keeps
    // the choice next to the test instead of in a glob that can drift.
    //
    // happy-dom rather than jsdom: jsdom pulls in undici, which needs a newer
    // Node than CI runs, and the worker died before any test executed.
    environment: 'node',
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
});
