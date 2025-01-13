import { confirm } from '@inquirer/prompts';
import chalk from 'chalk';
import { ScanOptions } from '../actions/scan.js';
import { Logger } from '../lib/Logger.js';

export const shouldAddScanResults = async (options: ScanOptions, logger: Logger): Promise<boolean> => {
  if (options.add) {
    return true;
  }

  if (options.quiet) {
    logger.log('\n' + chalk.yellow('Use the --add flag to add these mod to your modlist.'), true);
    return false;
  }

  return confirm({
    message: 'Do you want to add these mods and/or make changes to your config?',
    default: true
  });
};
