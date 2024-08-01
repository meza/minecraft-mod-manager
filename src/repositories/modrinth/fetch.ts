import { CouldNotFindModException } from '../../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../../errors/NoRemoteFileFound.js';
import { getNextVersionDown } from '../../lib/fallbackVersion.js';
import { Loader, Platform, ReleaseType, RemoteModDetails } from '../../lib/modlist.types.js';
import { rateLimitingFetch } from '../../lib/rateLimiter/index.js';
import { Modrinth } from './index.js';

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
  version_number: string;
  version_type: ReleaseType;
  files: ModrinthFile[];
}

interface ModrinthMod {
  name: string;
  versions: ModrinthVersion[];
}

const getName = async (projectId: string): Promise<string> => {
  performance.mark('modrinth-getname-start');
  const url = `https://api.modrinth.com/v2/project/${projectId}`;
  const modInfoRequest = await rateLimitingFetch(url, {
    headers: Modrinth.API_HEADERS
  });

  performance.mark('modrinth-getname-end');
  performance.measure(`modrinth-getname-${projectId}`, 'modrinth-getname-start', 'modrinth-getname-end');
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

  const modVersions = (await modDetailsRequest.json()) as ModrinthVersion[];

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

const hasTheCorrectVersion = (version: ModrinthVersion, allowedGameVersion: string) => {
  return version.game_versions.includes(allowedGameVersion);
};

const getPotentialFiles = (
  versions: ModrinthVersion[],
  loader: Loader,
  allowedReleaseTypes: ReleaseType[],
  allowedGameVersion: string
) => {
  return versions
    .filter((version) => {
      return hasTheCorrectLoader(version, loader);
    })
    .filter((version) => {
      return hasTheCorrectReleaseType(version, allowedReleaseTypes);
    })
    .filter((version) => {
      return hasTheCorrectVersion(version, allowedGameVersion);
    })
    .sort((versionA, versionB) => {
      return versionA.date_published < versionB.date_published ? 1 : -1;
    });
};

export const getMod = async (
  projectId: string,
  allowedReleaseTypes: ReleaseType[],
  allowedGameVersion: string,
  loader: Loader,
  allowFallback: boolean,
  fixedModVersion?: string
): Promise<RemoteModDetails> => {
  performance.mark('modrinth-getmod-start');
  const { name, versions } = await getModDetails(projectId, allowedGameVersion, loader);
  let potentialFiles = [];
  if (fixedModVersion) {
    potentialFiles = versions.filter((file) => {
      return file.version_number === fixedModVersion;
    });
  } else {
    potentialFiles = getPotentialFiles(versions, loader, allowedReleaseTypes, allowedGameVersion);
  }

  if (potentialFiles.length === 0) {
    if (allowFallback) {
      const versionDown = getNextVersionDown(allowedGameVersion);
      return getMod(projectId, allowedReleaseTypes, versionDown.nextVersionToTry, loader, versionDown.canGoDown);
    }

    performance.mark('modrinth-getmod-failed');
    performance.measure(`modrinth-getmod-${projectId}-failed`, 'modrinth-getmod-start', 'modrinth-getmod-failed');
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

  performance.mark('modrinth-getmod-end');
  performance.measure(`modrinth-getmod-${projectId}`, 'modrinth-getmod-start', 'modrinth-getmod-end');

  return modData;
};
