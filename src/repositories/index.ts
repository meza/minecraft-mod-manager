import { Loader, Platform, ReleaseType } from '../lib/modlist.types.js';
import { getMod as cfMod } from './curseforge/index.js';
import { getMod as mMod } from './modrinth/index.js';

export const fetchModDetails = async (platform: Platform, id: string, allowedReleaseTypes: ReleaseType[], gameVersion: string, loader: Loader) => {
  switch (platform) {
    case Platform.CURSEFORGE: return await cfMod(id, allowedReleaseTypes, gameVersion, loader);
    case Platform.MODRINTH: return await mMod(id, allowedReleaseTypes, gameVersion, loader);
    default: throw new Error('Unknown platform');
  }
};
