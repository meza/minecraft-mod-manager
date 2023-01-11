import { PlatformLookupResult } from '../index.js';

import { Modrinth } from './index.js';
import { ModrinthFile, ModrinthVersion } from './fetch.js';
import { Platform, RemoteModDetails } from '../../lib/modlist.types.js';

const startLookup = async (hash: string) => {
  const url = `https://api.modrinth.com/v2/version_file/${hash}?algorithm=sha1`;

  const response = await fetch(url, {
    headers: Modrinth.API_HEADERS
  });

  if (!response.ok) {
    throw new Error(response.statusText);
  }

  return await response.json() as ModrinthVersion;
};

export const lookup = async (hashes: string[]): Promise<PlatformLookupResult[]> => {
  const lookupQueue: Promise<ModrinthVersion>[] = [];

  hashes.forEach((hash) => {
    lookupQueue.push(startLookup(hash));
  });

  const settledQueue = await Promise.allSettled(lookupQueue);

  const results: PlatformLookupResult[] = [];

  settledQueue.forEach((lookup) => {

    if (lookup.status === 'rejected') {
      return;
    }

    const data: ModrinthVersion = lookup.value;
    const matchingFile: ModrinthFile = data.files[0];

    const modData: RemoteModDetails = {
      name: data.name,
      fileName: matchingFile.filename,
      releaseDate: data.date_published,
      hash: matchingFile.hashes.sha1,
      downloadUrl: matchingFile.url
    };

    results.push({
      modId: data.project_id,
      platform: Platform.MODRINTH,
      mod: modData
    });
  });

  return results;

};
