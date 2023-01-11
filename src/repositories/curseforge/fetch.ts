import { Loader, Platform, ReleaseType, RemoteModDetails } from '../../lib/modlist.types.js';
import { curseForgeApiKey } from '../../env.js';
import { CouldNotFindModException } from '../../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../../errors/NoRemoteFileFound.js';
import { InvalidReleaseTypeException } from './InvalidReleaseTypeException.js';
import { Curseforge } from './index.js';

export enum HashFunctions {
  // eslint-disable-next-line no-unused-vars
  sha1 = 1,
  // eslint-disable-next-line no-unused-vars
  md5 = 2,
}

interface Hash {
  algo: HashFunctions;
  value: string;
}

interface CurseForgeGameVersion {
  gameVersionName: string,
  gameVersion: string
}

export interface CurseforgeModFile {
  displayName: string;
  fileDate: string;
  releaseType: number;
  fileName: string;
  downloadUrl: string;
  fileStatus: number;
  isAvailable: boolean;
  hashes: Hash[];
  sortableGameVersions: CurseForgeGameVersion[];
  fileFingerprint: number;
}

const getHash = (hashes: Hash[], algo: HashFunctions): string => {
  const hash = hashes.find((h) => h.algo === algo);
  if (!hash) {
    throw new Error('Hash not found');
  }
  return hash.value;
};

const releaseTypeFromNumber = (curseForgeReleaseType: number): ReleaseType => {
  switch (curseForgeReleaseType) {
    case 1:
      return ReleaseType.RELEASE;
    case 2:
      return ReleaseType.BETA;
    case 3:
      return ReleaseType.ALPHA;
    default:
      throw new InvalidReleaseTypeException(curseForgeReleaseType);
  }
};

const getFiles = async (projectId: string, gameVersion: string, loader: Loader): Promise<CurseforgeModFile[]> => {
  const cfLoader = Curseforge.curseforgeLoaderFromLoader(loader);
  const url = `https://api.curseforge.com/v1/mods/${projectId}/files?gameVersion=${gameVersion}&modLoaderType=${cfLoader}`;

  const modFiles = await fetch(url, {
    headers: {
      'Accept': 'application/json',
      'x-api-key': curseForgeApiKey
    }
  });

  // if (modFiles.status !== 200) {
  //   throw new CouldNotFindModException(projectId, Platform.CURSEFORGE);
  // }
  // TODO fix this

  const filesData = await modFiles.json();
  return filesData.data as CurseforgeModFile[];
};

export const curseforgeFileToRemoteModDetails = (file: CurseforgeModFile, name: string): RemoteModDetails => {
  return {
    name: name,
    fileName: file.fileName,
    releaseDate: file.fileDate,
    hash: getHash(file.hashes, HashFunctions.sha1),
    downloadUrl: file.downloadUrl
  };
};

export const getMod = async (projectId: string, allowedReleaseTypes: ReleaseType[], allowedGameVersion: string, loader: Loader, allowFallback: boolean) => {
  const url = `https://api.curseforge.com/v1/mods/${projectId}`;

  const modDetailsRequest = await fetch(url, {
    headers: {
      'Accept': 'application/json',
      'x-api-key': curseForgeApiKey
    }
  });

  if (modDetailsRequest.status !== 200) {
    throw new CouldNotFindModException(projectId, Platform.CURSEFORGE);
  }

  const modDetails = await modDetailsRequest.json();
  const files = await getFiles(projectId, allowedGameVersion, loader);
  const potentialFiles = files
    .filter((file) => {
      return file.sortableGameVersions.find((gameVersion) => gameVersion.gameVersionName.toLowerCase() === loader.toLowerCase());
    })
    .filter((file) => {
      try {
        return file.isAvailable && allowedReleaseTypes.includes(releaseTypeFromNumber(file.releaseType)) && [4, 10].includes(file.fileStatus);
      } catch (e) {
        return false;
      }
    })
    .filter((file) => {
      return file.sortableGameVersions.some((gameVersion) => {
        if (gameVersion.gameVersion === allowedGameVersion) {
          return true;
        }
        if (allowFallback) {
          const [major, minor, patch] = allowedGameVersion.split('.');
          const decreasedVersion = `${major}.${minor}.${parseInt(patch, 10) - 1}`;

          if (gameVersion.gameVersion === decreasedVersion) {
            return true;
          }

          if (gameVersion.gameVersion === `${major}.${minor}`) {
            return true;
          }
        }

        return false;
      });
    })
    .sort((a, b) => {
      return a.fileDate < b.fileDate ? 1 : -1;
    });

  if (potentialFiles.length === 0) {
    throw new NoRemoteFileFound(modDetails.data.name, Platform.CURSEFORGE);
  }

  const latestFile = potentialFiles[0];

  try {
    const modData = curseforgeFileToRemoteModDetails(latestFile, modDetails.data.name);

    return modData;
  } catch (e) { // Catch when the hash is not found (due to curseforge error)
    throw new NoRemoteFileFound(modDetails.data.name, Platform.CURSEFORGE);
  }
};

