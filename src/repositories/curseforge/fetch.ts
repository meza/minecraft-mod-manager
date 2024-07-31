import { getNextVersionDown } from '../../lib/fallbackVersion.js';
import { Loader, Platform, ReleaseType, RemoteModDetails } from '../../lib/modlist.types.js';
import { curseForgeApiKey } from '../../env.js';
import { CouldNotFindModException } from '../../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../../errors/NoRemoteFileFound.js';
import { InvalidReleaseTypeException } from './InvalidReleaseTypeException.js';
import { Curseforge } from './index.js';
import { rateLimitingFetch } from '../../lib/rateLimiter/index.js';
import { CurseforgeDownloadUrlError } from '../../errors/CurseforgeDownloadUrlError.js';

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

  const modFiles = await rateLimitingFetch(url, {
    headers: {
      'Accept': 'application/json',
      'x-api-key': curseForgeApiKey
    }
  });

  if (!modFiles.ok) {
    throw new CouldNotFindModException(projectId, Platform.CURSEFORGE);
  }

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

const getPotentialFiles = (files: CurseforgeModFile[], allowedGameVersion: string, allowedReleaseTypes: ReleaseType[]): CurseforgeModFile[] => {
  return files.filter((file) => {
    return file.sortableGameVersions.find((gameVersion) => gameVersion.gameVersionName.toLowerCase() === allowedGameVersion.toLowerCase());
  })
    .filter((file) => {
      try {
        return file.isAvailable && allowedReleaseTypes.includes(releaseTypeFromNumber(file.releaseType)) && [4, 10].includes(file.fileStatus);
      } catch (e) {
        return false;
      }
    })
    .sort((a, b) => {
      return a.fileDate < b.fileDate ? 1 : -1;
    });
};

export const getMod = async (
  projectId: string,
  allowedReleaseTypes: ReleaseType[],
  allowedGameVersion: string,
  loader: Loader,
  allowFallback: boolean,
  fixedModVersion?: string): Promise<RemoteModDetails> => {

  performance.mark('curseforge-getmod-start');

  const url = `https://api.curseforge.com/v1/mods/${projectId}`;
  const modDetailsRequest = await rateLimitingFetch(url, {
    headers: {
      'Accept': 'application/json',
      'x-api-key': curseForgeApiKey
    }
  });

  if (!modDetailsRequest.ok) {
    throw new CouldNotFindModException(projectId, Platform.CURSEFORGE);
  }

  const modDetails = await modDetailsRequest.json();
  const files = await getFiles(projectId, allowedGameVersion, loader);

  let potentialFiles = [];

  if (fixedModVersion) {
    potentialFiles = files.filter((file) => {
      return file.fileName.toLowerCase() === fixedModVersion.toLowerCase();
    });
  } else {
    potentialFiles = getPotentialFiles(files, allowedGameVersion, allowedReleaseTypes);
  }

  if (potentialFiles.length === 0) {

    if (allowFallback) {
      const versionDown = getNextVersionDown(allowedGameVersion);
      return getMod(projectId, allowedReleaseTypes, versionDown.nextVersionToTry, loader, versionDown.canGoDown);
    }

    performance.mark('curseforge-getmod-failed');
    performance.measure(`curseforge-getmod-${projectId}-failed`, 'curseforge-getmod-start', 'curseforge-getmod-failed');

    throw new NoRemoteFileFound(modDetails.data.name, Platform.CURSEFORGE);
  }

  const latestFile = potentialFiles[0];

  if (latestFile.downloadUrl === null) {
    throw new CurseforgeDownloadUrlError(modDetails.data.name);
  }

  try {
    const modData = curseforgeFileToRemoteModDetails(latestFile, modDetails.data.name);
    performance.mark('curseforge-getmod-end');
    performance.measure(`curseforge-getmod-${projectId}`, 'curseforge-getmod-start', 'curseforge-getmod-end');
    return modData;
  } catch (e) { // Catch when the hash is not found (due to curseforge error)
    performance.mark('curseforge-getmod-failed');
    performance.measure(`curseforge-getmod-${projectId}-failed`, 'curseforge-getmod-start', 'curseforge-getmod-failed');
    throw new NoRemoteFileFound(modDetails.data.name, Platform.CURSEFORGE);
  }
};

