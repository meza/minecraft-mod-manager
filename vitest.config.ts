import isCi from 'is-ci';
import { defineConfig } from 'vitest/config';

const testReporters = ['default'];
const coverageReporters = ['text'];

if (!isCi) {
  // testReporters.push('verbose');
  coverageReporters.push('html');
} else {
  testReporters.push('junit');
  coverageReporters.push('cobertura');
}

export default defineConfig({
  cacheDir: '.cache',
  test: {
    globalSetup: './test/globalSetup.ts',
    dir: 'src',
    testTimeout: 10000,
    watch: false,
    outputFile: 'reports/junit.xml',
    reporters: testReporters,
    isolate: true,
    coverage: {
      include: ['src/**/*.ts'],
      exclude: ['**/*.testGameVersion.ts', '**/__mocks__/**.*', '**/*.d.ts', '**/*.test.ts'],
      all: true,
      reportsDirectory: './reports/coverage/unit',
      reporter: coverageReporters,
      thresholds: {
        branches: 100,
        functions: 100,
        lines: 100,
        statements: 100
      }
    }
  }
});
