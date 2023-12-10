import { Loader, Platform, ReleaseType, RemoteModDetails } from '../../lib/modlist.types.js';
import { CouldNotFindModException } from '../../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../../errors/NoRemoteFileFound.js';
import { Modrinth } from './index.js';
import { rateLimitingFetch } from '../../lib/rateLimiter/index.js';

export interface Hash {
  sha1: string;
  sha512: string;
}

export interface ModrinthFile {
  hashes: Hash;
  url: string;
  filename: string;
}

export interface ModrinthVersion {
  project_id: string;
  name: string;
  loaders: string[];
  game_versions: string[];
  date_published: string;
  version_type: ReleaseType;
  files: ModrinthFile[];
}

interface ModrinthMod {
  name: string,
  versions: ModrinthVersion[]
}

const getName = async (projectId: string): Promise<string> => {
  const url = `https://api.modrinth.com/v2/project/${projectId}`;
  const modInfoRequest = await rateLimitingFetch(url, {
    headers: Modrinth.API_HEADERS
  });

  if (!modInfoRequest.ok) {
    throw new CouldNotFindModException(projectId, Platform.MODRINTH);
  }

  const modInfo = await modInfoRequest.json();
  return modInfo.title;
};

const getModDetails = async (projectId: string, gameVersion: string, loader: Loader): Promise<ModrinthMod> => {
  const name = await getName(projectId);
  const url = `https://api.modrinth.com/v2/project/${projectId}/version?game_versions=["${gameVersion}"]&loaders=["${loader}"]`;

  const modDetailsRequest = await rateLimitingFetch(url, {
    headers: Modrinth.API_HEADERS
  });

  if (!modDetailsRequest.ok) {
    throw new CouldNotFindModException(projectId, Platform.MODRINTH);
  }

  const modVersions = await modDetailsRequest.json() as ModrinthVersion[];

  return {
    versions: modVersions,
    name: name
  };
};

const hasTheCorrectLoader = (version: ModrinthVersion, loader: string) => {
  return version.loaders.map((origLoader: string) => origLoader.toLowerCase()).includes(loader.toLowerCase());
};

const hasTheCorrectReleaseType = (version: ModrinthVersion, allowedReleaseTypes: ReleaseType[]) => {
  return allowedReleaseTypes.includes(version.version_type);
};

const hasTheCorrectVersion = (version: ModrinthVersion, allowedGameVersion: string, allowFallback: boolean) => {
  if (version.game_versions.includes(allowedGameVersion)) {
    return true;
  }

  if (allowFallback) {

    const [major, minor, patch] = allowedGameVersion.split('.').map((num) => parseInt(num, 10));

    if (patch && patch > 1) {
      if (version.game_versions.includes(`${major}.${minor}.${patch - 1}`)) {
        return true;
      }
    }

    if (version.game_versions.includes(`${major}.${minor}`)) {
      return true;
    }
  }

  return false;
};

export const getMod = async (
  projectId: string,
  allowedReleaseTypes: ReleaseType[],
  allowedGameVersion: string,
  loader: Loader,
  allowFallback: boolean): Promise<RemoteModDetails> => {

  const { name, versions } = await getModDetails(projectId, allowedGameVersion, loader);
  const potentialFiles = versions
    .filter((version) => {
      return hasTheCorrectLoader(version, loader);
    })
    .filter((version) => {
      return hasTheCorrectReleaseType(version, allowedReleaseTypes);
    })
    .filter((version) => {
      return hasTheCorrectVersion(version, allowedGameVersion, allowFallback);
    })
    .sort((versionA, versionB) => {
      return versionA.date_published < versionB.date_published ? 1 : -1;
    });

  if (potentialFiles.length === 0) {
    throw new NoRemoteFileFound(projectId, Platform.MODRINTH);
  }

  const latestFile = potentialFiles[0];

  const modData: RemoteModDetails = {
    name: name,
    fileName: latestFile.files[0].filename,
    releaseDate: latestFile.date_published,
    hash: latestFile.files[0].hashes.sha1,
    downloadUrl: latestFile.files[0].url
  };

  return modData;
};
