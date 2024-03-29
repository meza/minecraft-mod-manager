import { defineConfig } from 'vitest/config';
import isCi from 'is-ci';

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
  test: {
    cache: {
      dir: '.cache/.vitest'
    },
    globalSetup: './test/globalSetup.ts',
    dir: 'src',
    testTimeout: 10000,
    watch: false,
    outputFile: 'reports/junit.xml',
    reporters: testReporters,
    coverage: {
      excludeNodeModules: true,
      include: ['src/**/*.ts'],
      exclude: ['**/*.testGameVersion.ts', '**/__mocks__/**.*', '**/*.d.ts', '**/*.test.ts'],
      all: true,
      reportsDirectory: './reports/coverage/unit',
      reporter: coverageReporters,
      statements: 100,
      branches: 100,
      functions: 100,
      lines: 100
    }
  }
});
