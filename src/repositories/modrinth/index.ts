import { getMod } from './fetch.js';
import { Loader, ReleaseType, RemoteModDetails } from '../../lib/modlist.types.js';
import { PlatformLookupResult, Repository } from '../index.js';
import { version } from '../../version.js';
import { modrinthApiKey } from '../../env.js';
import { lookup as modrinthLookup } from './lookup.js';

export class Modrinth implements Repository {
  static PREFERRED_HASH = 'sha1';
  static API_HEADERS = {
    'user-agent': `github_com/meza/minecraft-mod-manager/${version}`,
    'Accept': 'application/json',
    'Authorization': modrinthApiKey
  };

  fetchMod(projectId: string, allowedReleaseTypes: ReleaseType[], allowedGameVersion: string, loader: Loader, allowFallback: boolean): Promise<RemoteModDetails> {
    return getMod(projectId, allowedReleaseTypes, allowedGameVersion, loader, allowFallback);
  }

  lookup(lookup: string[]): Promise<PlatformLookupResult[]> {
    return modrinthLookup(lookup);
  }

}
