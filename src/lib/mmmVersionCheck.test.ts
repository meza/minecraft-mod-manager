import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { hasUpdate } from './mmmVersionCheck.js';
import { GithubReleasesNotFoundException } from '../errors/GithubReleasesNotFoundException.js';
import { chance } from 'jest-chance';

describe('The MMM Version Check module', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });
  afterEach(() => {
    expect(fetch).toHaveBeenCalledOnce();
    vi.resetAllMocks();
  });

  it('should throw an error if the response is not ok', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({ ok: false } as Response);
    await expect(hasUpdate('')).rejects.toThrow(new GithubReleasesNotFoundException());
  });

  it('should handle dev builds', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
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

    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {
    });
    const result = await hasUpdate('dev-' + chance.word());
    expect(result).toEqual({
      hasUpdate: false,
      latestVersion: 'v1.0.0',
      latestVersionUrl: 'release-url',
      releasedOn: 'Sun Oct 09 2022 22:28:59 GMT+0100 (British Summer Time)'
    });

    expect(consoleSpy).toHaveBeenCalledWith('\n[update] You are running a development version of MMM. '
      + 'Please update to the latest release from Sun Oct 09 2022 22:28:59 GMT+0100 (British Summer Time).');
    expect(consoleSpy).toHaveBeenCalledWith('[update] You can download it from release-url\n');

  });

  it('should return the current version if there is no update', () => {
    vi.mocked(fetch).mockResolvedValueOnce({
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
    expect(hasUpdate('1.0.0')).resolves.toEqual({
      hasUpdate: false,
      latestVersion: 'v1.0.0',
      latestVersionUrl: undefined,
      releasedOn: 'Thu Dec 09 2021 12:20:59 GMT+0000 (Greenwich Mean Time)'
    });
  });

  it('should prioritize releases over prereleases', () => {
    vi.mocked(fetch).mockResolvedValueOnce({
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
    expect(hasUpdate('0.0.9')).resolves.toEqual({
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
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve([
          // eslint-disable-next-line camelcase
          { tag_name: latestVersion, prerelease: prerelease, draft: draft, html_url: url }
        ])
      } as Response);
      expect(hasUpdate(currentVersion)).resolves.toEqual({
        hasUpdate: true,
        latestVersion: latestVersion,
        latestVersionUrl: url,
        releasedOn: expect.any(String)
      });
    });
  });
});
