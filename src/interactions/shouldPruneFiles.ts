import { confirm } from '@inquirer/prompts';
import chalk from 'chalk';
import { PruneOptions } from '../actions/prune.js';
import { Logger } from '../lib/Logger.js';

export const shouldPruneFiles = async (options: PruneOptions, logger: Logger) => {
  if (options.force) {
    return true;
  }

  if (options.quiet) {
    logger.log('There are files to prune but you are using --quiet.', true);
    logger.log(
      chalk.yellowBright('Use mmm prune --quiet --force to prune all the files without any interaction'),
      true
    );
    return false;
  }

  return confirm({
    message: 'Do you want to delete these files?',
    default: true
  });
};
