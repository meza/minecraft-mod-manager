import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { hasUpdate } from './mmmVersionCheck.js';
import { chance } from 'jest-chance';
import { Logger } from './Logger.js';
import { rateLimitingFetch } from './rateLimiter/index.js';

vi.mock('./Logger.js');
vi.mock('./rateLimiter/index.js');

describe('The MMM Version Check module', () => {
  let logger: Logger;

  beforeEach(() => {
    logger = new Logger({} as never);
    vi.useFakeTimers();
  });
  afterEach(() => {
    expect(rateLimitingFetch).toHaveBeenCalledOnce();
    vi.resetAllMocks();
    vi.useRealTimers();
  });

  it('should stay silent if the latest version cannot be fetched', async () => {
    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({ ok: false } as Response);
    const actual = await hasUpdate('', logger);
    expect(actual.hasUpdate).toBeFalsy();
    expect(actual.latestVersion).toEqual('vDEV');
    expect(actual.latestVersionUrl).toEqual('<github cannot be reached>');
  });

  it('should throw an error if the fetch errors', async () => {
    vi.mocked(rateLimitingFetch).mockRejectedValueOnce(new Error('something'));
    await expect(hasUpdate('', logger)).rejects.toThrow('something');
  });

  it('should handle dev builds', async () => {
    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: true,
      json: vi.fn().mockResolvedValueOnce([
        {
          // eslint-disable-next-line camelcase
          tag_name: 'v1.0.0',
          prerelease: false,
          draft: false,
          // eslint-disable-next-line camelcase
          html_url: 'release-url',
          // eslint-disable-next-line camelcase
          published_at: '2022-10-09T21:28:59Z'
        }
      ])
    } as unknown as Response);

    const result = await hasUpdate('dev-' + chance.word(), logger);
    expect(result).toEqual({
      hasUpdate: false,
      latestVersion: 'v1.0.0',
      latestVersionUrl: 'release-url',
      releasedOn: 'Sun Oct 09 2022 21:28:59 GMT+0000 (Greenwich Mean Time)'
    });

    expect(logger.log).toHaveBeenCalledWith('\n[update] You are running a development version of MMM. '
      + 'Please update to the latest release from Sun Oct 09 2022 21:28:59 GMT+0000 (Greenwich Mean Time).');
    expect(logger.log).toHaveBeenCalledWith('[update] You can download it from release-url\n');

  });

  it('should return the current version if there is no update', () => {
    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve([
        // eslint-disable-next-line camelcase
        { tag_name: 'v1.0.0', prerelease: false, draft: false, published_at: '2021-12-09T12:20:59Z' },
        // eslint-disable-next-line camelcase
        { tag_name: 'v0.9.0', prerelease: false, draft: false },
        // eslint-disable-next-line camelcase
        { tag_name: 'v0.8.0', prerelease: false, draft: false }
      ])
    } as Response);
    expect(hasUpdate('1.0.0', logger)).resolves.toEqual({
      hasUpdate: false,
      latestVersion: 'v1.0.0',
      latestVersionUrl: undefined,
      releasedOn: 'Thu Dec 09 2021 12:20:59 GMT+0000 (Greenwich Mean Time)'
    });
  });

  it('should prioritize releases over prereleases', () => {
    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve([
        // eslint-disable-next-line camelcase
        { tag_name: 'v1.0.1', prerelease: true, draft: false },
        // eslint-disable-next-line camelcase
        { tag_name: 'v1.0.0', prerelease: false, draft: false, published_at: '2019-02-03T02:20:59Z' },
        // eslint-disable-next-line camelcase
        { tag_name: 'v0.8.0', prerelease: false, draft: false }
      ])
    } as Response);
    expect(hasUpdate('0.0.9', logger)).resolves.toEqual({
      hasUpdate: true,
      latestVersion: 'v1.0.0',
      latestVersionUrl: undefined,
      releasedOn: 'Sun Feb 03 2019 02:20:59 GMT+0000 (Greenwich Mean Time)'
    });
  });

  describe.each([
    { type: 'prerelease', prerelease: true, draft: false },
    { type: 'release', prerelease: false, draft: false }
  ])('should work for $type only', ({ prerelease, draft }) => {
    it.each([
      { currentVersion: '1.0.0', latestVersion: 'v1.0.1', name: 'patch' },
      { currentVersion: '1.0.0', latestVersion: 'v1.1.0', name: 'minor' },
      { currentVersion: '1.0.0', latestVersion: 'v2.0.0', name: 'major' }
    ])('should return the new version when there is a $name update', ({ currentVersion, latestVersion }) => {
      const url = chance.url();
      vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve([
          // eslint-disable-next-line camelcase
          { tag_name: latestVersion, prerelease: prerelease, draft: draft, html_url: url }
        ])
      } as Response);
      expect(hasUpdate(currentVersion, logger)).resolves.toEqual({
        hasUpdate: true,
        latestVersion: latestVersion,
        latestVersionUrl: url,
        releasedOn: expect.any(String)
      });
    });
  });
});
