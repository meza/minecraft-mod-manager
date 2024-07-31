import chalk from 'chalk';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import { RedundantVersionException } from '../errors/RedundantVersionException.js';
import { Logger } from '../lib/Logger.js';
import { UpgradeVerificationResult, verifyUpgradeIsPossible, VerifyUpgradeOptions } from '../lib/verifyUpgrade.js';
import { EXIT_CODE } from '../mmm.js';

// disabling due to https://github.com/typescript-eslint/typescript-eslint/issues/1277
// eslint-disable-next-line consistent-return
export const testGameVersion = async (gameVersion: string, options: VerifyUpgradeOptions, logger: Logger): Promise<UpgradeVerificationResult> | never => {
  performance.mark('test-version-start');
  try {
    const verified = await verifyUpgradeIsPossible(gameVersion, options, logger);
    if (!verified.canUpgrade) {
      logger.log(`Some mods are missing support for ${verified.version}.`);
      verified.modsInError.forEach((mod) => {
        logger.log(`${chalk.red('\u274c')} ${mod.name?.trim()} ${chalk.gray('(')}${chalk.gray(mod.id)}${chalk.gray(')')}`);
      });
      logger.error(`You cannot upgrade to ${verified.version} just yet.`, 1);
    }

    logger.log(chalk.green(`All mods have support for ${verified.version}. You can safely upgrade.`));
    performance.mark('test-version-succeed');
    return verified;

  } catch (e) {
    performance.mark('test-version-fail');
    if (e instanceof IncorrectMinecraftVersionException) {
      logger.error(e.message, EXIT_CODE.GENERAL_ERROR);
    }
    if (e instanceof RedundantVersionException) {
      logger.error(e.message, EXIT_CODE.SUPPLEMENTARY_ERROR);
    }
    logger.error((e as Error).message, EXIT_CODE.GENERAL_ERROR);
  }
};
