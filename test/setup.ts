import { afterEach, vi } from 'vitest';

afterEach(() => {
  // these run after every single test
  vi.unstubAllEnvs();
});
