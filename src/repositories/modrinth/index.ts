import { ModDetails, ReleaseType } from '../../lib/modlist.types.js';

interface Hash {
  sha1: string;
  sha512: string;
}

interface ModrinthFile {
  hashes: Hash;
  url: string;
  filename: string;
}

interface ModrinthVersion {
  name: string;
  loaders: string[];
  game_versions: string[];
  date_published: string;
  version_type: ReleaseType;
  files: ModrinthFile[];
}

export const getMod = async (
  projectId: string,
  allowedReleaseTypes: ReleaseType[],
  allowedGameVersion: string,
  loader: string,
  allowFallback: boolean): Promise<ModDetails> => {
  const url = `https://api.modrinth.com/v2/project/${projectId}/version`;

  const modDetailsRequest = await fetch(url, {
    headers: {
      'user-agent': `github_com/meza/minecraft-mod-updater/${process.env.npm_package_version}`,
      'Accept': 'application/json',
      'Authorization': process.env.MODRINTH_API_KEY
    }
  });

  const modVersions = await modDetailsRequest.json() as ModrinthVersion[];

  const potentialFiles = modVersions
    .filter((version) => {
      return version.loaders.map((origLoader: string) => origLoader.toLowerCase()).includes(loader.toLowerCase());
    })
    .filter((version) => {
      const hasPerfectMatch = version.game_versions.includes(allowedGameVersion);
      if (!hasPerfectMatch && allowFallback) {
        const gameVersionsWithOnlyMajorAndMinor = version.game_versions.map((gameVersion) => {
          const [major, minor] = gameVersion.split('.');
          return `${major}.${minor}`;
        });

        const [major, minor] = allowedGameVersion.split('.');

        return gameVersionsWithOnlyMajorAndMinor.includes(`${major}.${minor}`);
      }
      return hasPerfectMatch;
    })
    .filter((version) => {
      return allowedReleaseTypes.includes(version.version_type);
    })
    .sort((a, b) => {
      return a.date_published < b.date_published ? 1 : -1;
    })
  ;

  if (potentialFiles.length === 0) {
    throw new Error(`No files found for the given mod: Modrinth ${projectId}`);
  }
  const latestFile = potentialFiles[0];

  const modData: ModDetails = {
    name: latestFile.name,
    fileName: latestFile.files[0].filename,
    releaseDate: latestFile.date_published,
    hash: latestFile.files[0].hashes.sha1,
    downloadUrl: latestFile.files[0].url
  };

  return modData;
};
