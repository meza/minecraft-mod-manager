import { curseForgeApiKey } from '../../env.js';
import { curseforgeFileToRemoteModDetails, CurseforgeModFile } from './fetch.js';
import { PlatformLookupResult } from '../index.js';
import { Platform } from '../../lib/modlist.types.js';
import { logger } from '../../mmm.js';
import { rateLimitingFetch } from '../../lib/rateLimiter/index.js';
import chalk from 'chalk';

interface CurseforgeLookupMatches {
  id: number;
  file: CurseforgeModFile;
}
interface CurseforgeLookupResult {
  data: {
    exactMatches: CurseforgeLookupMatches[];
    exactFingerprints: number[];
  }
}

export const lookup = async (fingerprints: string[]): Promise<PlatformLookupResult[]> => {
  const url = 'https://api.curseforge.com/v1/fingerprints';
  const modSearchResult = await rateLimitingFetch(url, {
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
      'x-api-key': curseForgeApiKey
    },
    method: 'POST',
    body: JSON.stringify({
      'fingerprints': fingerprints
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

  return result;
};
