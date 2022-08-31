import path from 'path';
import { DefaultOptions } from '../index.js';
import { fetchModDetails } from '../repositories/index.js';
import { readConfigFile, writeConfigFile } from './config.js';
import { downloadFile } from './downloader.js';
import { ModConfig } from './modlist.types.js';

interface UpdateOptions extends DefaultOptions {}

export const update = async (options: UpdateOptions) => {
  const configuration = await readConfigFile(options.config);

  for(let i = 0; i<configuration.mods.length; i++) {
    const mod = configuration.mods[i] as ModConfig;
    const moddata = await fetchModDetails(mod.type, mod.id, configuration.defaultAllowedReleaseTypes, configuration.gameVersion, configuration.loader);
    console.log(`Updating from ${mod.type} ${moddata.name}`);
    if (!mod.installed) {
      await downloadFile(moddata.downloadUrl, path.resolve(configuration.modsFolder, moddata.fileName));
      mod.installed = {
        fileName: moddata.fileName,
        releasedOn: moddata.releaseDate,
        hash: moddata.hash
      };
    }

    if(moddata.hash !== mod.installed.hash) {
      console.log(`Downloading ${moddata.name}`);
      await downloadFile(moddata.downloadUrl, path.resolve(configuration.modsFolder, moddata.fileName));

      mod.installed = {
        fileName: moddata.fileName,
        releasedOn: moddata.releaseDate,
        hash: moddata.hash
      };
    }
    configuration.mods[i] = mod;
  }

  await writeConfigFile(configuration, options.config);
}
