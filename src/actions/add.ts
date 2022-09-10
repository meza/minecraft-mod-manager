import path from 'path';
import { fetchModDetails } from '../repositories/index.js';
import { ModConfig, Platform } from '../lib/modlist.types.js';
import { readConfigFile, writeConfigFile } from '../lib/config.js';
import { downloadFile } from '../lib/downloader.js';
import { DefaultOptions } from '../mmu.js';

export const add = async (platform: Platform, id: string, options: DefaultOptions) => {
  const configuration = await readConfigFile(options.config);

  if (configuration.mods.find((mod: ModConfig) => (mod.id === id && mod.type === platform))) {
    if (options.debug) {
      console.debug(`Mod ${id} for ${platform} already exists in the configuration`);
    }

    return;
  }

  try {
    const modData = await fetchModDetails(
      platform,
      id,
      configuration.defaultAllowedReleaseTypes,
      configuration.gameVersion,
      configuration.loader,
      configuration.allowVersionFallback);

    await downloadFile(modData.downloadUrl, path.resolve(configuration.modsFolder, modData.fileName));

    configuration.mods.push({
      type: platform,
      id: id,
      name: modData.name,
      installed: {
        fileName: modData.fileName,
        releasedOn: modData.releaseDate,
        hash: modData.hash
      },
      allowedReleaseTypes: configuration.defaultAllowedReleaseTypes
    });

    await writeConfigFile(configuration, options.config);
  } catch (error) {
    console.error(error);
  }

};
