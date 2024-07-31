/* eslint-disable no-unused-vars */
export interface RemoteModDetails {
  name: string;
  fileName: string;
  releaseDate: string;
  hash: string;
  downloadUrl: string;
}

export enum ReleaseType {
  ALPHA = 'alpha',
  BETA = 'beta',
  RELEASE = 'release'
}

export enum Platform {
  CURSEFORGE = 'curseforge',
  MODRINTH = 'modrinth'
}

export enum Loader {
  BUKKIT = 'bukkit',
  BUNGEECORD = 'bungeecord',
  CAULDRON = 'cauldron',
  DATAPACK = 'datapack',
  FABRIC = 'fabric',
  FOLIA = 'folia',
  FORGE = 'forge',
  LITELOADER = 'liteloader',
  MODLOADER = 'modloader',
  NEOFORGE = 'neoforge',
  PAPER = 'paper',
  PURPUR = 'purpur',
  QUILT = 'quilt',
  RIFT = 'rift',
  SPIGOT = 'spigot',
  SPONGE = 'sponge',
  VELOCITY = 'velocity',
  WATERFALL = 'waterfall',
}

export interface ModInstall {
  type: Platform,
  id: string,
  name: string,
  fileName: string,
  releasedOn: string,
  hash: string,
  downloadUrl: string,
}

export interface Mod {
  type: Platform,
  id: string,
  allowedReleaseTypes?: ReleaseType[]
  name: string,
  allowVersionFallback?: boolean,
  version?: string | undefined
}

export interface ModsJson {
  loader: Loader,
  gameVersion: string,
  defaultAllowedReleaseTypes: ReleaseType[],
  modsFolder: string,
  mods: Mod[]
}
