import { ModDetails, ReleaseType } from '../../lib/modlist.types.js';

export enum HashFunctions {
  sha1 = 1,
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

interface CurseforgeModFile {
  displayName: string
  fileDate: string
  releaseType: number
  fileName: string
  downloadUrl: string
  fileStatus: number
  isAvailable: boolean
  hashes: Hash[],
  sortableGameVersions: CurseForgeGameVersion[]
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
      throw new Error('Cannot determine release type');
  }
};

export const getMod = async (projectId: string, allowedReleaseTypes: ReleaseType[], allowedGameVersion: string, loader: string) => {
  const url = `https://api.curseforge.com/v1/mods/${projectId}`;

  const modDetailsRequest = await fetch(url, {
    headers: {
      'Accept': 'application/json',
      'x-api-key': process.env.CURSEFORGE_API_KEY
    }
  });

  if (modDetailsRequest.status !== 200) {
    throw new Error('Could not find the given mod');
  }

  const modDetails = await modDetailsRequest.json();
  const files = modDetails.data.latestFiles as CurseforgeModFile[];

  const potentialFiles = files
    .filter((file) => {
      return file.sortableGameVersions.find((gameVersion) => gameVersion.gameVersionName.toLowerCase() === loader.toLowerCase());
    })
    .filter((file) => {
      return file.isAvailable && allowedReleaseTypes.includes(releaseTypeFromNumber(file.releaseType)) && [4, 10].includes(file.fileStatus);
    })
    .filter((file) => {
      return file.sortableGameVersions.some((gameVersion) => {
        return gameVersion.gameVersion.includes(allowedGameVersion);
      });
    })
    .sort((a, b) => {
      return a.fileDate < b.fileDate ? 1 : -1;
    });

  if (potentialFiles.length === 0) {
    throw new Error('No files found for the given mod');
  }

  const latestFile = potentialFiles[0];

  const modData: ModDetails = {
    name: latestFile.displayName,
    fileName: latestFile.fileName,
    releaseDate: latestFile.fileDate,
    hash: getHash(latestFile.hashes, HashFunctions.sha1),
    downloadUrl: latestFile.downloadUrl
  };

  return modData;
};
