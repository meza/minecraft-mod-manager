import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    cache: {
      dir: '.cache/.vitest'
    },
    dir: 'src',
    testTimeout: 10000,
    watch: false,
    outputFile: 'reports/junit.xml',
    reporters: ['default', 'junit'],
    coverage: {
      reportsDirectory: './reports/coverage/unit',
      reporter: ['text', 'json', 'html'],
      statements: 100,
      branches: 100,
      functions: 100,
      lines: 100
    }
  }
});
