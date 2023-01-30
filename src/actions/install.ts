import {
  fileExists,
  readConfigFile,
  readLockFile,
  writeConfigFile,
  writeLockFile
} from '../lib/config.js';
import path from 'path';
import { fetchModDetails } from '../repositories/index.js';
import { downloadFile } from '../lib/downloader.js';
import { Mod, RemoteModDetails } from '../lib/modlist.types.js';
import { getHash } from '../lib/hash.js';
import { DefaultOptions } from '../mmm.js';
import { updateMod } from '../lib/updater.js';
import { Logger } from '../lib/Logger.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { ErrorTexts } from '../errors/ErrorTexts.js';
import { getInstallation, hasInstallation } from '../lib/configurationHelper.js';
import { DownloadFailedException } from '../errors/DownloadFailedException.js';

const getMod = async (moddata: RemoteModDetails, modsFolder: string) => {
  await downloadFile(moddata.downloadUrl, path.resolve(modsFolder, moddata.fileName));
  return {
    fileName: moddata.fileName,
    releasedOn: moddata.releaseDate,
    hash: moddata.hash,
    downloadUrl: moddata.downloadUrl
  };
};

export const install = async (options: DefaultOptions, logger: Logger) => {
  try {
    const configuration = await readConfigFile(options.config);
    const installations = await readLockFile(options.config);

    const installedMods = installations;
    const mods = configuration.mods;

    const processMod = async (mod: Mod, index: number) => {

      logger.debug(`Checking ${mod.name} for ${mod.type}`);

      if (hasInstallation(mod, installations)) {
        const installedModIndex = getInstallation(mod, installedMods);

        const modPath = path.resolve(configuration.modsFolder, installedMods[installedModIndex].fileName);

        if (!await fileExists(modPath)) {
          logger.log(`${mod.name} doesn't exist, downloading from ${installedMods[installedModIndex].type}`);
          await downloadFile(installedMods[installedModIndex].downloadUrl, modPath);
          return;
        }

        const installedHash = await getHash(modPath);
        if (installedMods[installedModIndex].hash !== installedHash) {
          logger.log(`${mod.name} has hash mismatch, downloading from source`);
          await updateMod(installedMods[installedModIndex], modPath, configuration.modsFolder);
          return;
        }
        return;
      }

      const modData = await fetchModDetails(
        mod.type,
        mod.id,
        mod.allowedReleaseTypes || configuration.defaultAllowedReleaseTypes,
        configuration.gameVersion,
        configuration.loader,
        configuration.allowVersionFallback
      );
      // TODO Handle the fetch failing

      mods[index].name = modData.name;

      // no installation exists
      logger.log(`${mod.name} doesn't exist, downloading from ${mod.type}`);
      const dlData = await getMod(modData, configuration.modsFolder);

      installedMods.push({
        name: modData.name,
        type: mod.type,
        id: mod.id,
        fileName: dlData.fileName,
        releasedOn: dlData.releasedOn,
        hash: dlData.hash,
        downloadUrl: dlData.downloadUrl
      });
      return;

    };

    const promises = mods.map(processMod);

    await Promise.all(promises);

    await writeLockFile(installedMods, options.config);
    await writeConfigFile(configuration, options.config);
  } catch (error) {
    if (error instanceof DownloadFailedException) {
      logger.error(error.message, 1);
    }
    if (error instanceof ConfigFileNotFoundException) {
      logger.error(ErrorTexts.configNotFound);
    }

    logger.error((error as Error).message, 2);
  }
};
