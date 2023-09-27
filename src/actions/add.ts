import path from 'path';
import { fetchModDetails } from '../repositories/index.js';
import { Mod, Platform } from '../lib/modlist.types.js';
import {
  ensureConfiguration,
  readLockFile,
  writeConfigFile,
  writeLockFile
} from '../lib/config.js';
import { downloadFile } from '../lib/downloader.js';
import { DefaultOptions, stop } from '../mmm.js';
import { UnknownPlatformException } from '../errors/UnknownPlatformException.js';
import inquirer from 'inquirer';
import chalk from 'chalk';
import { CouldNotFindModException } from '../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../errors/NoRemoteFileFound.js';
import { Logger } from '../lib/Logger.js';
import { modNotFound } from '../interactions/modNotFound.js';
import { noRemoteFileFound } from '../interactions/noRemoteFileFound.js';
import { DownloadFailedException } from '../errors/DownloadFailedException.js';

const handleUnknownPlatformException = async (error: UnknownPlatformException, id: string, options: DefaultOptions, logger: Logger) => {
  const platformUsed = error.platform;
  const platformList = Object.values(Platform);

  if (options.quiet === true) {
    logger.error(`Unknown platform "${chalk.whiteBright(platformUsed)}". Please use one of the following: ${chalk.whiteBright(platformList.join(', '))}`);
  }

  const answers = await inquirer.prompt([
    {
      type: 'list',
      name: 'platform',
      default: false,
      choices: [...platformList, 'cancel'],
      message: chalk.redBright(`The platform you entered (${chalk.whiteBright(platformUsed)}) is not a valid platform.\n`)
        + chalk.whiteBright('Would you like to retry with a valid one?')
    }
  ]);

  if (answers.platform === 'cancel') {
    stop();
  }

  // eslint-disable-next-line no-use-before-define
  await add(answers.platform, id, options, logger);

};

export const add = async (platform: Platform, id: string, options: DefaultOptions, logger: Logger) => {

  const configuration = await ensureConfiguration(options.config, logger, options.quiet);
  const modConfig = configuration.mods.find((mod: Mod) => (mod.id === id && mod.type === platform));

  if (modConfig) {
    logger.debug(`Mod ${id} for ${platform} already exists in the configuration`);
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

    const installations = await readLockFile(options, logger);

    configuration.mods.push({
      type: platform,
      id: id,
      name: modData.name
    });

    installations.push({
      name: modData.name,
      type: platform,
      id: id,
      fileName: modData.fileName,
      releasedOn: modData.releaseDate,
      hash: modData.hash,
      downloadUrl: modData.downloadUrl
    });

    await writeConfigFile(configuration, options, logger);
    await writeLockFile(installations, options, logger);

    logger.log(`${chalk.green('\u2705')} Added ${modData.name} (${id}) for ${platform}`);

  } catch (error) {
    if (error instanceof DownloadFailedException) {
      logger.error(error.message, 1);
    }
    if (error instanceof UnknownPlatformException) {
      await handleUnknownPlatformException(error, id, options, logger);
      return;
    }

    if (error instanceof CouldNotFindModException) {
      const { id: newId, platform: newPlatform } = await modNotFound(id, platform, logger, options);
      await add(newPlatform, newId, options, logger);
      return;
    }

    if (error instanceof NoRemoteFileFound) {
      const {
        id: newId,
        platform: newPlatform
      } = await noRemoteFileFound(id, platform, configuration, logger, options);
      await add(newPlatform, newId, options, logger);
      return;
    }

    logger.error((error as Error).message, 2);
  }

};
