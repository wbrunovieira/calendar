import { defineConfig } from 'vitest/config';
import swc from 'unplugin-swc';
import { resolve } from 'path';

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    root: './',
    include: ['test/**/*.e2e-spec.ts'],
    exclude: ['node_modules', 'dist'],
    testTimeout: 30000,
    hookTimeout: 30000,
    setupFiles: ['./test/setup-e2e.ts'],
    // setup-e2e.ts wipes the schema in beforeAll. Run the files one at a time, or a second worker
    // truncates the tables the first one is mid-way through using.
    fileParallelism: false,
  },
  plugins: [
    swc.vite({
      module: { type: 'es6' },
    }),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@domains': resolve(__dirname, './src/domains'),
      '@common': resolve(__dirname, './src/common'),
    },
  },
});
