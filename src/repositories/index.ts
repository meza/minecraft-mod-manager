import { Loader, Platform, ReleaseType, RemoteModDetails } from '../lib/modlist.types.js';
import { UnknownPlatformException } from '../errors/UnknownPlatformException.js';
import { Curseforge } from './curseforge/index.js';
import { Modrinth } from './modrinth/index.js';

export interface PlatformLookupResult {
  platform: Platform;
  modId: string;
  mod: RemoteModDetails;
}

export interface LookupHits {
  platform: Platform;
  mod: RemoteModDetails;
}

export interface LookupResult {
  hash: string;
  hits: LookupHits[];
}

/**
 * The hash/hashes to look up
 * The platform property is necessary to distinguish between the inputs since each platform has a unique requirement
 */
export interface LookupInput {
  platform: Platform;
  hash: string[];
}

export interface Repository {
  fetchMod: (projectId: string, allowedReleaseTypes: ReleaseType[], allowedGameVersion: string, loader: Loader, allowFallback: boolean, version?: string) => Promise<RemoteModDetails>;
  lookup: (lookup: string[]) => Promise<PlatformLookupResult[]>;
}

export interface ResultItem {
  sha1Hash: string;
  hits: PlatformLookupResult[];
}

const getRepository = (platform: Platform): Repository => {
  switch (platform) {
    case Platform.CURSEFORGE:
      return new Curseforge();
    case Platform.MODRINTH:
      return new Modrinth();
    default:
      throw new UnknownPlatformException(platform);
  }
};

/**
 * Fetches the mod's details
 *
 * @param platform
 * @param id
 * @param allowedReleaseTypes
 * @param gameVersion
 * @param loader
 * @param allowFallback
 * @param fixedModVersion
 * @throws {CouldNotFindModException} When the mod itself cannot be found
 * @throws {NoRemoteFileFound} When a suitable file for the mod cannot be found
 */
export const fetchModDetails = async (
  platform: Platform,
  id: string,
  allowedReleaseTypes: ReleaseType[],
  gameVersion: string,
  loader: Loader,
  allowFallback: boolean,
  fixedModVersion?: string
) => {

  const repository = getRepository(platform);
  return await repository.fetchMod(id, allowedReleaseTypes, gameVersion, loader, allowFallback, fixedModVersion);

};

export const lookup = async (lookup: LookupInput[]): Promise<ResultItem[]> => {
  if (lookup.length === 0) {
    return [];
  }

  const lookups: Promise<PlatformLookupResult[]>[] = [];

  Object.values(Platform).map((platform) => {

    const specificInput = lookup.find(l => l.platform === platform);

    if (!specificInput) {
      return;
    }

    const repository = getRepository(platform);
    lookups.push(repository.lookup(specificInput.hash));
  });

  const platformLookupResults = await Promise.allSettled(lookups);

  const consolidatedResult: ResultItem[] = [];

  platformLookupResults.forEach((platformLookupResult) => {
    if (platformLookupResult.status === 'rejected') {
      return;
    }

    const platformResult: PlatformLookupResult[] = platformLookupResult.value;

    platformResult.forEach((match) => {
      const hash = match.mod.hash;
      const targetIndex = consolidatedResult.findIndex((i) => i.sha1Hash === hash);
      if (targetIndex === -1) {
        consolidatedResult.push({
          sha1Hash: hash,
          hits: [match]
        });
      } else {
        consolidatedResult[targetIndex].hits.push(match);
      }
    });

  });

  return consolidatedResult;
};
