import { ModInstall, ModsJson, Platform } from './modlist.types.js';
import { fetchModDetails, lookup, LookupInput, ResultItem } from '../repositories/index.js';
import { getHash } from './hash.js';
import { Modrinth } from '../repositories/modrinth/index.js';
import curseforge from '@meza/curseforge-fingerprint';
import { ScanResults } from '../actions/scan.js';
import { fileIsManaged } from './configurationHelper.js';
import { getModFiles } from './fileHelper.js';

const getScanResults = async (files: string[], installations: ModInstall[]) => {
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
  return lookupResults;
};

export const scan = async (configLocation: string, prefer: Platform, configuration: ModsJson, installations: ModInstall[]) => {
  const files = await getModFiles(configLocation, configuration.modsFolder);

  const lookupResults = await getScanResults(files, installations);

  const normalizers: Promise<ScanResults>[] = [];

  const normalizeResults = async (lookupResult: ResultItem): Promise<ScanResults> => {
    lookupResult.hits.sort((hit1, hit2) => {
      if (hit1.platform === prefer && hit2.platform !== prefer) { // there can't be 2 identical hits from the same platform
        return -1;
      }
      return 1;
    });

    const modDetails = await fetchModDetails(
      lookupResult.hits[0].platform,
      lookupResult.hits[0].modId,
      configuration.defaultAllowedReleaseTypes,
      configuration.gameVersion,
      configuration.loader,
      configuration.allowVersionFallback
    );
    return {
      resolvedDetails: modDetails,
      localDetails: lookupResult.hits
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
