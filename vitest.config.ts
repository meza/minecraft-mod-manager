import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    dir: 'src',
    testTimeout: 10000,
    watch: false,
    coverage: {
      reportsDirectory: './reports/coverage/unit',
      reporter: ['text', 'json', 'html']
    }
  }
});
