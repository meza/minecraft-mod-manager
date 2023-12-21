import chalk from 'chalk';
import {
  ensureConfiguration,
  fileExists,
  getModsFolder,
  readLockFile,
  writeConfigFile,
  writeLockFile
} from '../lib/config.js';
import path from 'path';
import { fetchModDetails } from '../repositories/index.js';
import { downloadFile } from '../lib/downloader.js';
import { Mod, ModInstall, ModsJson, Platform, RemoteModDetails } from '../lib/modlist.types.js';
import { getHash } from '../lib/hash.js';
import { DefaultOptions } from '../mmm.js';
import { updateMod } from '../lib/updater.js';
import { Logger } from '../lib/Logger.js';
import { fileIsManaged, getInstallation, hasInstallation } from '../lib/configurationHelper.js';
import { handleFetchErrors } from '../errors/handleFetchErrors.js';
import { getModFiles } from '../lib/fileHelper.js';
import { scanFiles } from '../lib/scan.js';
import { processScanResults } from './scan.js';

const getMod = async (moddata: RemoteModDetails, modsFolder: string) => {
  await downloadFile(moddata.downloadUrl, path.resolve(modsFolder, moddata.fileName));
  return {
    fileName: moddata.fileName,
    releasedOn: moddata.releaseDate,
    hash: moddata.hash,
    downloadUrl: moddata.downloadUrl
  };
};

const handleUnknownFiles = async (options: DefaultOptions, configuration: ModsJson, installations: ModInstall[], logger: Logger) => {
  //const modsFolder = getModsDir(options.config, configuration.modsFolder);
  const allFiles = await getModFiles(options.config, configuration);
  const nonManagedFiles = allFiles.filter((filePath) => {
    return !fileIsManaged(filePath, installations);
  });

  if (nonManagedFiles.length === 0) {
    return;
  }

  const scanResults = await scanFiles(nonManagedFiles, installations, Platform.MODRINTH, configuration);
  const { unsure } = processScanResults(scanResults, configuration, installations, logger);

  if (unsure.length > 0) {
    logger.error('\nPlease fix the unresolved issues above manually or by running mmm scan, then try again.', 1);
  }
};

export const install = async (options: DefaultOptions, logger: Logger) => {

  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options, logger);
  await handleUnknownFiles(options, configuration, installations, logger);
  const installedMods = installations;
  const mods = configuration.mods;
  const modsFolder = getModsFolder(options.config, configuration);

  const processMod = async (mod: Mod, index: number) => {
    const canonVersion = mod.version || 'latest';
    try {
      logger.debug(`Checking ${mod.name}@${canonVersion} for ${mod.type}`);

      if (hasInstallation(mod, installations)) {
        const installedModIndex = getInstallation(mod, installedMods);

        const modPath = path.resolve(getModsFolder(options.config, configuration), installedMods[installedModIndex].fileName);

        if (!await fileExists(modPath)) {
          logger.log(`${mod.name} doesn't exist, downloading from ${installedMods[installedModIndex].type}`);
          await downloadFile(installedMods[installedModIndex].downloadUrl, modPath);
          return;
        }

        const installedHash = await getHash(modPath);
        if (installedMods[installedModIndex].hash !== installedHash) {
          logger.log(`${mod.name} has hash mismatch, downloading from source`);
          await updateMod(installedMods[installedModIndex], modPath, modsFolder);
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
        !!mod.allowVersionFallback,
        mod.version
      );

      mods[index].name = modData.name;

      // no installation exists
      logger.log(`${mod.name} doesn't exist, downloading from ${mod.type}`);
      const dlData = await getMod(modData, modsFolder);

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
      handleFetchErrors(error as Error, mod, logger);
    }

  };

  const promises = mods.map(processMod);

  await Promise.all(promises);

  await writeLockFile(installedMods, options, logger);
  await writeConfigFile(configuration, options, logger);
  logger.log(`${chalk.green('\u2705')} all mods are installed!`);
};
