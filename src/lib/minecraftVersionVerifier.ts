import { MinecraftVersionsCouldNotBeFetchedException } from '../errors/MinecraftVersionsCouldNotBeFetchedException.js';
import { rateLimitingFetch } from './rateLimiter/index.js';

export interface MinecraftVersionInfo {
  id: string,
  type: 'release' | 'snapshot',
  url: string,
  time: string,
  releaseTime: string
}

export interface MinecraftVersionsApi {
  latest: {
    release: string,
    snapshot: string
  },
  versions: MinecraftVersionInfo[]
}

const listMinecraftVersions = async (): Promise<MinecraftVersionsApi> => {
  const url = 'https://launchermeta.mojang.com/mc/game/version_manifest.json';

  try {
    const response = await rateLimitingFetch(url);

    if (!response.ok) {
      throw new MinecraftVersionsCouldNotBeFetchedException();
    }

    return await response.json();
  } catch {
    throw new MinecraftVersionsCouldNotBeFetchedException();
  }
};

export const getLatestMinecraftVersion = async (): Promise<string> => {
  const { latest } = await listMinecraftVersions();
  return latest.release;
};

export const verifyMinecraftVersion = async (input: string): Promise<boolean> => {
  try {
    const { versions } = await listMinecraftVersions();

    return versions.some(({ id }) => id === input);
  } catch (e) {
    return true;
  }
};
