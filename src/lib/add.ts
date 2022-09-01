import path from 'path';
import { DefaultOptions } from '../index.js';
import { fetchModDetails } from '../repositories/index.js';
import { ModConfig, Platform } from './modlist.types.js';
import { readConfigFile, writeConfigFile } from './config.js';
import { downloadFile } from './downloader.js';

export interface AddOptions extends DefaultOptions {
}

export const add = async (platform: Platform, id: string, options: AddOptions) => {
  const configuration = await readConfigFile(options.config);

  const moddata = await fetchModDetails(platform, id, configuration.defaultAllowedReleaseTypes, configuration.gameVersion, configuration.loader);

  await downloadFile(moddata.downloadUrl, path.resolve(configuration.modsFolder, moddata.fileName));

  if (configuration.mods.find((mod: ModConfig) => (mod.id === id && mod.type === platform))) {
    return;
  }

  configuration.mods.push({
    type: platform,
    id: id,
    name: moddata.name,
    installed: {
      fileName: moddata.fileName,
      releasedOn: moddata.releaseDate,
      hash: moddata.hash
    },
    allowedReleaseTypes: configuration.defaultAllowedReleaseTypes
  });

  await writeConfigFile(configuration, options.config);

};
