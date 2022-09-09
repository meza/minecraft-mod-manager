import path from 'path';
import { DefaultOptions } from '../index.js';
import { fetchModDetails } from '../repositories/index.js';
import { fileExists, readConfigFile, writeConfigFile } from './config.js';
import { downloadFile } from './downloader.js';
import { ModConfig, ModDetails } from './modlist.types.js';
import { getHash } from './hash.js';
import fs from 'node:fs/promises';

interface UpdateOptions extends DefaultOptions {
}

const getMod = async (moddata: ModDetails, modsFolder: string) => {
  await downloadFile(moddata.downloadUrl, path.resolve(modsFolder, moddata.fileName));
  return {
    fileName: moddata.fileName,
    releasedOn: moddata.releaseDate,
    hash: moddata.hash
  };
};

export const update = async (options: UpdateOptions) => {
  const configuration = await readConfigFile(options.config);

  for (let i = 0; i < configuration.mods.length; i++) {
    const mod = configuration.mods[i] as ModConfig;

    const moddata = await fetchModDetails(
      mod.type,
      mod.id,
      configuration.defaultAllowedReleaseTypes,
      configuration.gameVersion,
      configuration.loader,
      configuration.allowVersionFallback
    );

    mod.name = moddata.name;

    if (!mod.installed) {
      console.log(`Installing ${mod.name}`);
      const dlData = await getMod(moddata, configuration.modsFolder);
      mod.installed = {
        fileName: dlData.fileName,
        releasedOn: dlData.releasedOn,
        hash: dlData.hash
      };
    }

    const modPath = path.resolve(configuration.modsFolder, mod.installed.fileName);

    if (!await fileExists(modPath)) {
      console.log(`${mod.name} doesn't exist, downloading from ${mod.type}`);
      const dlData = await getMod(moddata, configuration.modsFolder);
      mod.installed = {
        fileName: dlData.fileName,
        releasedOn: dlData.releasedOn,
        hash: dlData.hash
      };
    }

    const installedHash = await getHash(modPath);

    if (moddata.hash !== installedHash) {
      console.log(`${mod.name}  has hash mismatch, downloading from source`);
      await fs.rename(modPath, `${modPath}.bak`);
      try {
        const newPath = path.resolve(configuration.modsFolder, moddata.fileName);
        await downloadFile(moddata.downloadUrl, newPath);
        mod.name = moddata.name;
        mod.installed = {
          fileName: moddata.fileName,
          releasedOn: moddata.releaseDate,
          hash: moddata.hash
        };

        await fs.rm(`${modPath}.bak`);
      } catch {
        console.log('Download failed, restoring the original');
        await fs.rename(`${modPath}.bak`, modPath);
      }
    }
    configuration.mods[i] = mod;
  }

  await writeConfigFile(configuration, options.config);
};
