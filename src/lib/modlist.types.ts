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
  FORGE = 'forge',
  FABRIC = 'fabric'
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
  name?: string,
}

export interface ModsJson {
  loader: Loader,
  gameVersion: string,
  allowVersionFallback: boolean,
  defaultAllowedReleaseTypes: ReleaseType[],
  modsFolder: string,
  mods: Mod[]
}
