import chalk from 'chalk';
import { curseForgeApiKey } from '../../env.js';
import { Platform } from '../../lib/modlist.types.js';
import { rateLimitingFetch } from '../../lib/rateLimiter/index.js';
import { logger } from '../../mmm.js';
import { PlatformLookupResult } from '../index.js';
import { CurseforgeModFile, curseforgeFileToRemoteModDetails } from './fetch.js';

interface CurseforgeLookupMatches {
  id: number;
  file: CurseforgeModFile;
}
interface CurseforgeLookupResult {
  data: {
    exactMatches: CurseforgeLookupMatches[];
    exactFingerprints: number[];
  };
}

export const lookup = async (fingerprints: string[]): Promise<PlatformLookupResult[]> => {
  const url = 'https://api.curseforge.com/v1/fingerprints';
  performance.mark('curseforge-lookup-start');
  const modSearchResult = await rateLimitingFetch(url, {
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'x-api-key': curseForgeApiKey
    },
    method: 'POST',
    body: JSON.stringify({
      fingerprints: fingerprints
    })
  });

  if (!modSearchResult.ok) {
    logger.log(chalk.whiteBright(chalk.bgRed('Could not reach Curseforge, please try again')));
    return [];
  }

  const data: CurseforgeLookupResult = await modSearchResult.json();

  const result: PlatformLookupResult[] = [];

  data.data.exactMatches.forEach((match) => {
    result.push({
      modId: String(match.id),
      platform: Platform.CURSEFORGE,
      mod: curseforgeFileToRemoteModDetails(match.file, match.file.displayName)
    });
  });

  performance.mark('curseforge-lookup-end');
  performance.measure('curseforge-lookup', 'curseforge-lookup-start', 'curseforge-lookup-end');

  return result;
};
