import path from 'path';
import { fetchModDetails } from '../repositories/index.js';
import { Mod, ModsJson, Platform } from '../lib/modlist.types.js';
import { initializeConfigFile, readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { downloadFile } from '../lib/downloader.js';
import { DefaultOptions } from '../mmm.js';
import { UnknownPlatformException } from '../errors/UnknownPlatformException.js';
import inquirer from 'inquirer';
import chalk from 'chalk';
import { CouldNotFindModException } from '../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../errors/NoRemoteFileFound.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { shouldCreateConfig } from '../interactions/shouldCreateConfig.js';
import { Logger } from '../lib/Logger.js';
import { modNotFound } from '../interactions/modNotFound.js';

const handleUnknownPlatformException = async (error: UnknownPlatformException, id: string, options: DefaultOptions, logger: Logger) => {
  const platformUsed = error.platform;
  const platformList = Object.values(Platform);

  if (options.quiet === true) {
    logger.error(`Unknown platform "${chalk.whiteBright(platformUsed)}". Please use one of the following: ${chalk.whiteBright(platformList.join(', '))}`);
    return;
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
    return;
  }

  // eslint-disable-next-line no-use-before-define
  await add(answers.platform, id, options, logger);

};

const getConfiguration = async (options: DefaultOptions): Promise<ModsJson> => {
  try {
    return await readConfigFile(options.config);
  } catch (error) {
    if (error instanceof ConfigFileNotFoundException && options.quiet === false) {
      if (await shouldCreateConfig(options.config)) {
        return await initializeConfigFile(options.config);
      }
    }
    throw error;
  }
};

export const add = async (platform: Platform, id: string, options: DefaultOptions, logger: Logger) => {

  const configuration = await getConfiguration(options);

  if (configuration.mods.find((mod: Mod) => (mod.id === id && mod.type === platform))) {
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

    const installations = await readLockFile(options.config);

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

    await writeConfigFile(configuration, options.config);
    await writeLockFile(installations, options.config);

  } catch (error) {
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
      logger.error(
        `Could not find a file for the version ${chalk.whiteBright(configuration.gameVersion)} `
        + `for ${chalk.whiteBright(configuration.loader)}`
      );
      // Todo handle with unified exit
    }

    logger.error((error as Error).message, 2);
  }

};
