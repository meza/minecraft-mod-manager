export interface ModDetails {
  name: string
  fileName: string
  releaseDate: string
  hash: string
  downloadUrl: string
}

export enum ReleaseType {
  ALPHA='alpha',
  BETA='beta',
  RELEASE='release'
}

export enum Platform {
  CURSEFORGE='curseforge',
  MODRINTH='modrinth'
}

export enum Loader {
  FORGE='forge',
  FABRIC='fabric'
}

export interface ModInstall {
  fileName: string,
  releasedOn: string,
  hash: string
}

export interface ModConfig {
  type: Platform,
  id: string,
  installed?: ModInstall,
  allowedReleaseTypes: ReleaseType[]
}

export interface ModlistConfig {
  loader: Loader,
  gameVersion: string,
  defaultAllowedReleaseTypes: ReleaseType[],
  modsFolder: string,
  mods: ModConfig[]
}
