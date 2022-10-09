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

export const listMinecraftVersions = async (): Promise<MinecraftVersionsApi> => {
  const url = 'https://launchermeta.mojang.com/mc/game/version_manifest.json';

  const response = await fetch(url);

  if (!response.ok) {
    throw new Error('Minecraft versions not found');
  }

  return await response.json();
};

export const getLatestMinecraftVersion = async (): Promise<string> => {
  const { latest } = await listMinecraftVersions();
  return latest.release;
};

export const verifyMinecraftVersion = async (input: string): Promise<boolean> => {
  const { versions } = await listMinecraftVersions();

  return versions.some(({ id }) => id === input);
};
