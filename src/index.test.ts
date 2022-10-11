import { afterEach, describe, expect, it, vi } from 'vitest';
import { program } from './mmm.js';
import { hasUpdate } from './lib/mmmVersionCheck.js';
import { chance } from 'jest-chance';

vi.mock('./mmm.js');
vi.mock('./version.js', () => ({ version: '0.0.0' }));
vi.mock('./lib/mmmVersionCheck.js');

describe('The main entry point', () => {
  afterEach(() => {
    vi.resetModules();
    vi.resetAllMocks();
  });
  it('calls the main program when there are no updates', async () => {
    vi.mocked(hasUpdate).mockResolvedValueOnce({
      hasUpdate: false,
      latestVersion: '',
      latestVersionUrl: ''
    });
    await import('./index.js');
    expect(vi.mocked(program.parse)).toHaveBeenCalledWith(process.argv);
  });

  it('alerts when there are updates and still calls the main program', async () => {
    const randomVersion = chance.word();
    const randomUrl = chance.url();

    const consoleLogSpy = vi.spyOn(console, 'log').mockImplementation(() => {
    });

    vi.mocked(hasUpdate).mockResolvedValueOnce({
      hasUpdate: true,
      latestVersion: randomVersion,
      latestVersionUrl: randomUrl
    });
    await import('./index.js');

    expect(consoleLogSpy).toHaveBeenCalledWith(`There is a new version of MMM available: ${randomVersion}`);
    expect(consoleLogSpy).toHaveBeenCalledWith(`You can download it from ${randomUrl}`);
    expect(vi.mocked(program.parse)).toHaveBeenCalledWith(process.argv);
  });
});
