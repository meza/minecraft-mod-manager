import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    cache: {
      dir: '.cache/.vitest'
    },
    globalSetup: './test/globalSetup.ts',
    maxConcurrency: 100,
    dir: 'src',
    testTimeout: 5000,
    watch: false,
    css: false,
    outputFile: 'reports/junit.xml',
    reporters: ['verbose', 'junit'],
    coverage: {
      all: true,
      reportsDirectory: './reports/coverage/unit',
      reporter: ['text', 'cobertura'],
      statements: 100,
      branches: 100,
      functions: 100,
      lines: 100
    }
  }
});
