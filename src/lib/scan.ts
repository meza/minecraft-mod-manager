import { ModInstall, ModsJson, Platform } from './modlist.types.js';
import { fetchModDetails, lookup, LookupInput, ResultItem } from '../repositories/index.js';
import { getHash } from './hash.js';
import { Modrinth } from '../repositories/modrinth/index.js';
import curseforge from '@meza/curseforge-fingerprint';
import { ScanResults } from '../actions/scan.js';
import { fileIsManaged } from './configurationHelper.js';
import { getModFiles } from './fileHelper.js';

export const scan = async (configLocation: string, prefer: Platform, configuration: ModsJson, installations: ModInstall[]) => {
  const files = await getModFiles(configLocation, configuration.modsFolder);

  const cfInput: LookupInput = {
    platform: Platform.CURSEFORGE,
    hash: []
  };
  const modrinthInput: LookupInput = {
    platform: Platform.MODRINTH,
    hash: []
  };
  let found = 0;
  const all = files.map(async (filePath) => {
    if (fileIsManaged(filePath, installations)) {
      return;
    }
    found++;
    const fingerprint = curseforge.fingerprint(filePath);
    const fileSha1Hash = await getHash(filePath, Modrinth.PREFERRED_HASH);

    cfInput.hash.push(String(fingerprint));
    modrinthInput.hash.push(fileSha1Hash);
  });

  await Promise.all(all);
  if (found === 0) {
    return [];
  }

  const lookupResults: ResultItem[] = await lookup([cfInput, modrinthInput]);

  const normalizers: Promise<ScanResults>[] = [];

  const normalizeResults = async (lookupResult: ResultItem): Promise<ScanResults> => {
    let hit = lookupResult.hits[0];
    const preferredHitIndex = lookupResult.hits.findIndex((hit) => hit.platform.toString() === prefer);
    if (preferredHitIndex >= 0) {
      hit = lookupResult.hits[preferredHitIndex];
    }

    const modDetails = await fetchModDetails(hit.platform, hit.modId, configuration.defaultAllowedReleaseTypes, configuration.gameVersion, configuration.loader, configuration.allowVersionFallback);
    return {
      resolvedDetails: modDetails,
      localDetails: hit
    };
  };

  lookupResults.forEach((lookupResult) => {
    normalizers.push(normalizeResults(lookupResult));
  });

  const normalizedResults = await Promise.allSettled(normalizers);

  const result: ScanResults[] = [];
  normalizedResults.forEach((normalizeResult) => {
    if (normalizeResult.status === 'rejected') {
      return;
    }

    result.push(normalizeResult.value);
  });

  return result;
};
