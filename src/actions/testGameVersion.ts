import chalk from 'chalk';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import { RedundantVersionException } from '../errors/RedundantVersionException.js';
import { Logger } from '../lib/Logger.js';
import { verifyUpgradeIsPossible, VerifyUpgradeOptions } from '../lib/verifyUpgrade.js';
import { EXIT_CODE } from '../mmm.js';

export const testGameVersion = async (gameVersion: string, options: VerifyUpgradeOptions, logger: Logger) => {
  try {
    const { canUpgrade, version, modsInError } = await verifyUpgradeIsPossible(gameVersion, options, logger);
    if (!canUpgrade) {
      logger.log(`Some mods are missing support for ${version}.`);
      modsInError.forEach((mod) => {
        logger.log(`${chalk.red('\u274c')} ${mod.name?.trim()}`);
      });
      logger.error(`You cannot upgrade to ${version} just yet.`, 1);
    } else {
      logger.log(chalk.green(`All mods have support for ${version}. You can safely upgrade.`));
    }
  } catch (e) {
    if (e instanceof IncorrectMinecraftVersionException) {
      logger.error(e.message, EXIT_CODE.GENERAL_ERROR);
    }
    if (e instanceof RedundantVersionException) {
      logger.error(e.message, EXIT_CODE.SUPPLEMENTARY_ERROR);
    }
  }
};
