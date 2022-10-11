import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getLatestMinecraftVersion, verifyMinecraftVersion } from './minecraftVersionVerifier.js';
import { chance } from 'jest-chance';
import { MinecraftVersionsCouldNotBeFetchedException } from '../errors/MinecraftVersionsCouldNotBeFetchedException.js';

describe('The Minecraft version verifier module', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    expect(vi.mocked(fetch)).toHaveBeenCalledOnce();
    expect(vi.mocked(fetch)).toHaveBeenCalledWith('https://launchermeta.mojang.com/mc/game/version_manifest.json');
    vi.resetAllMocks();
  });

  it('should return the latest Minecraft version', async () => {
    const randomVersion = chance.word();
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: vi.fn().mockResolvedValueOnce({
        latest: {
          release: randomVersion,
          snapshot: chance.word()
        },
        versions: []
      })
    } as unknown as Response);

    const latestVersion = await getLatestMinecraftVersion();

    expect(latestVersion).toBe(randomVersion);
  });

  it('should throw an error when the Minecraft versions could not be fetched for listing', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false
    } as unknown as Response);

    await expect(getLatestMinecraftVersion()).rejects.toThrow(new MinecraftVersionsCouldNotBeFetchedException());
  });

  it('should throw an error when the Minecraft versions could not be fetched for verifying', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false
    } as unknown as Response);

    await expect(verifyMinecraftVersion('')).rejects.toThrow(new MinecraftVersionsCouldNotBeFetchedException());
  });

  it('should return true if the Minecraft version is valid', async () => {
    const randomVersion = chance.word();
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: vi.fn().mockResolvedValueOnce({
        latest: {
          release: chance.word(),
          snapshot: chance.word()
        },
        versions: [
          {
            id: chance.word(),
            type: 'release',
            url: chance.url(),
            time: chance.date().toISOString(),
            releaseTime: chance.date().toISOString()
          },
          {
            id: randomVersion,
            type: 'release',
            url: chance.url(),
            time: chance.date().toISOString(),
            releaseTime: chance.date().toISOString()
          }
        ]
      })
    } as unknown as Response);

    const isValid = await verifyMinecraftVersion(randomVersion);

    expect(isValid).toBe(true);
  });

  it('should return false if the Minecraft version is invalid', async () => {
    const randomVersion = chance.word();
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: vi.fn().mockResolvedValueOnce({
        latest: {
          release: chance.word(),
          snapshot: chance.word()
        },
        versions: [
          {
            id: chance.word(),
            type: 'release',
            url: chance.url(),
            time: chance.date().toISOString(),
            releaseTime: chance.date().toISOString()
          },
          {
            id: chance.word(),
            type: 'release',
            url: chance.url(),
            time: chance.date().toISOString(),
            releaseTime: chance.date().toISOString()
          }
        ]
      })
    } as unknown as Response);

    const isValid = await verifyMinecraftVersion(randomVersion);

    expect(isValid).toBe(false);
  });
});
