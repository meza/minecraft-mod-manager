import { readConfigFile, readLockFile } from '../lib/config.js';
import chalk from 'chalk';
import { DefaultOptions } from '../mmm.js';
import { Mod } from '../lib/modlist.types.js';
import { Logger } from '../lib/Logger.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { ErrorTexts } from '../errors/ErrorTexts.js';

export type ListOptions = DefaultOptions

export const list = async (options: ListOptions, logger: Logger) => {
  try {
    const config = await readConfigFile(options.config);
    const installed = await readLockFile(options.config);

    logger.log((chalk.green('Configured mods')), true);

    const sortByName = (a: Mod, b: Mod) => {
      return a.name.localeCompare(b.name);
    };

    config.mods.sort(sortByName).forEach((mod) => {
      if (installed.find((i) => i.id === mod.id && i.type === mod.type)) {
        logger.log(`${chalk.green('\u2705')} ${mod.name?.trim()} is installed`, true);
      } else {
        logger.log(`${chalk.red('\u274c')} ${mod.name?.trim()} is not installed`, true);
      }
    });
  } catch (error) {
    if (error instanceof ConfigFileNotFoundException) {
      logger.error(ErrorTexts.configNotFound);
    }
  }
};
