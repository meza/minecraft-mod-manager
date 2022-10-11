import { describe, expect, it, vi } from 'vitest';
import { program } from './mmm.js';
import { hasUpdate } from './lib/mmmVersionCheck.js';

vi.mock('./mmm.js');
vi.mock('./version.js', () => ({ version: '0.0.0' }));
vi.mock('./lib/mmmVersionCheck.js');

describe('The main entry point', () => {
  it('calls the main program', async () => {
    vi.mocked(hasUpdate).mockResolvedValueOnce({
      hasUpdate: false,
      latestVersion: '',
      latestVersionUrl: ''
    });
    await import('./index.js');
    // kinda bogus test but at least it makes sure that we don't forget to call the commander thing
    expect(vi.mocked(program.parse)).toHaveBeenCalledWith(process.argv);
  });
});
