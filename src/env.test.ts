import * as process from 'process';
import { afterEach, describe, expect, it, vi } from 'vitest';

describe('The environment variables', () => {
  afterEach(() => {
    vi.resetModules();
  });

  it('are set to their defaults when they are not overridden', async () => {
    // @ts-ignore
    delete process.env.CURSEFORGE_API_KEY;
    // @ts-ignore
    delete process.env.MODRINTH_API_KEY;

    const { curseForgeApiKey, modrinthApiKey } = await import('./env.js');
    expect(curseForgeApiKey).toBe('REPL_CURSEFORGE_API_KEY');
    expect(modrinthApiKey).toBe('REPL_MODRINTH_API_KEY');
  });

  it('are set to their values when they are overridden', async () => {
    process.env.CURSEFORGE_API_KEY = 'cf-key';
    process.env.MODRINTH_API_KEY = 'mr-key';
    const { curseForgeApiKey, modrinthApiKey } = await import('./env.js');
    expect(curseForgeApiKey).toBe('cf-key');
    expect(modrinthApiKey).toBe('mr-key');
  });
});
