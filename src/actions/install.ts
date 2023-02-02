import {
  ensureConfiguration,
  fileExists,
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
import { getInstallation, hasInstallation } from '../lib/configurationHelper.js';
import { DownloadFailedException } from '../errors/DownloadFailedException.js';
import { CouldNotFindModException } from '../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../errors/NoRemoteFileFound.js';
import chalk from 'chalk';

const getMod = async (moddata: RemoteModDetails, modsFolder: string) => {
  await downloadFile(moddata.downloadUrl, path.resolve(modsFolder, moddata.fileName));
  return {
    fileName: moddata.fileName,
    releasedOn: moddata.releaseDate,
    hash: moddata.hash,
    downloadUrl: moddata.downloadUrl
  };
};

const handleError = (error: Error, mod: Mod, logger: Logger) => {
  if (error instanceof CouldNotFindModException) {
    logger.log(`${chalk.red('\u274c')} ${mod.name}${chalk.gray('(' + mod.id + ')')} cannot be found on ${mod.type} anymore. Was the mod revoked?`, true);
    return;
  }

  if (error instanceof NoRemoteFileFound) {
    logger.log(`${chalk.red('\u274c')} ${mod.type} doesn't serve the required file for ${mod.name}${chalk.gray('(' + mod.id + ')')} anymore. Please update it.`, true);
    return;
  }

  if (error instanceof DownloadFailedException) {
    logger.error(error.message, 1);
  }

  throw error;
};

export const install = async (options: DefaultOptions, logger: Logger) => {

  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options.config);

  const installedMods = installations;
  const mods = configuration.mods;

  const processMod = async (mod: Mod, index: number) => {
    try {
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
    } catch (error) {
      handleError(error as Error, mod, logger);
    }

  };

  const promises = mods.map(processMod);

  await Promise.all(promises);

  await writeLockFile(installedMods, options.config);
  await writeConfigFile(configuration, options.config);
};
