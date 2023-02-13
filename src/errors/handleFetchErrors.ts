import { Mod } from '../lib/modlist.types.js';
import { Logger } from '../lib/Logger.js';
import { CouldNotFindModException } from './CouldNotFindModException.js';
import { NoRemoteFileFound } from './NoRemoteFileFound.js';
import { DownloadFailedException } from './DownloadFailedException.js';
import chalk from 'chalk';

export const handleFetchErrors = (error: Error, mod: Mod, logger: Logger) => {
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
