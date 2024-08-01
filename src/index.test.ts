import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { hasUpdate } from './lib/mmmVersionCheck.js';
import { logger, program } from './mmm.js';

vi.mock('./lib/Logger.js');
vi.mock('./mmm.js');
vi.mock('./version.js', () => ({ version: '0.0.0' }));
vi.mock('./lib/mmmVersionCheck.js');

describe('The main entry point', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.resetAllMocks();
    vi.mocked(program.parseAsync).mockResolvedValue({} as never);
  });
  it('calls the main program when there are no updates', async () => {
    vi.mocked(hasUpdate).mockResolvedValueOnce({
      hasUpdate: false,
      latestVersion: '',
      latestVersionUrl: '',
      releasedOn: ''
    });
    await import('./index.js');
    expect(vi.mocked(program.parseAsync)).toHaveBeenCalledWith(process.argv);
  });

  it('alerts when there are updates and still calls the main program', async () => {
    const randomVersion = chance.word();
    const randomUrl = chance.url();
    const releasedOn = chance.date().toISOString();

    vi.mocked(hasUpdate).mockResolvedValueOnce({
      hasUpdate: true,
      latestVersion: randomVersion,
      latestVersionUrl: randomUrl,
      releasedOn: releasedOn
    });
    await import('./index.js');

    expect(vi.mocked(logger.log)).toHaveBeenCalledWith(
      `There is a new version of MMM available: ${randomVersion} from ${releasedOn}`
    );
    expect(vi.mocked(logger.log)).toHaveBeenCalledWith(`You can download it from ${randomUrl}`);
    expect(vi.mocked(program.parseAsync)).toHaveBeenCalledWith(process.argv);
  });
});
