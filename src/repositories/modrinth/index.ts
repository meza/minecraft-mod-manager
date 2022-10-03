import { Platform, ReleaseType, RemoteModDetails } from '../../lib/modlist.types.js';
import { version } from '../../version.js';
import { modrinthApiKey } from '../../env.js';
import { CouldNotFindModException } from '../../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../../errors/NoRemoteFileFound.js';

interface Hash {
  sha1: string;
  sha512: string;
}

export interface ModrinthFile {
  hashes: Hash;
  url: string;
  filename: string;
}

export interface ModrinthVersion {
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

const apiHeaders = {
  'user-agent': `github_com/meza/minecraft-mod-manager/${version}`,
  'Accept': 'application/json',
  'Authorization': modrinthApiKey
};

const getName = async (projectId: string): Promise<string> => {
  const url = `https://api.modrinth.com/v2/project/${projectId}`;
  const modInfoRequest = await fetch(url, {
    headers: apiHeaders
  });

  if (modInfoRequest.status !== 200) {
    throw new CouldNotFindModException(projectId, Platform.MODRINTH);
  }

  const modInfo = await modInfoRequest.json();
  return modInfo.title;
};

const getModDetails = async (projectId: string): Promise<ModrinthMod> => {
  const name = await getName(projectId);
  const url = `https://api.modrinth.com/v2/project/${projectId}/version`;

  const modDetailsRequest = await fetch(url, {
    headers: apiHeaders
  });

  if (modDetailsRequest.status !== 200) {
    throw new CouldNotFindModException(projectId, Platform.MODRINTH);
  }

  const modVersions = await modDetailsRequest.json() as ModrinthVersion[];

  return {
    versions: modVersions,
    name: name
  };
};

export const getMod = async (
  projectId: string,
  allowedReleaseTypes: ReleaseType[],
  allowedGameVersion: string,
  loader: string,
  allowFallback: boolean): Promise<RemoteModDetails> => {

  const { name, versions } = await getModDetails(projectId);
  const potentialFiles = versions
    .filter((version) => {
      return version.loaders.map((origLoader: string) => origLoader.toLowerCase()).includes(loader.toLowerCase());
    })
    .filter((version) => {
      return allowedReleaseTypes.includes(version.version_type);
    })
    .filter((version) => {
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
    })
    .sort((a, b) => {
      return a.date_published < b.date_published ? 1 : -1;
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
