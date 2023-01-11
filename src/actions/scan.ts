import { DefaultOptions } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { readConfigFile, readLockFile } from '../lib/config.js';
import { ResultItem } from '../repositories/index.js';
import { Platform, RemoteModDetails } from '../lib/modlist.types.js';
import { scan as scanLib } from '../lib/scan.js';
import chalk from 'chalk';

export interface ScanOptions extends DefaultOptions {
  prefer: Platform;
  add: boolean;
}

export interface ScanResults {
  resolvedDetails: RemoteModDetails;
  localDetails: ResultItem;
}

export const scan = async (options: ScanOptions, logger: Logger) => {
  const configuration = await readConfigFile(options.config);
  const installations = await readLockFile(options.config);

  const scanResults = await scanLib(options.prefer, configuration, installations);

  scanResults.forEach((hit) => {
    if (!options.quiet) {
      const message = chalk.green('\u2705') + `Found unmanaged mod: ${chalk.bold(chalk.whiteBright(hit.resolvedDetails.name))}`;
      logger.log(message);
    }

  });
  if (!options.add) {
    logger.log('\n' + chalk.yellow('Use the --add flag to add these mod to your modlist.'));
  }

  // scan during init
  // scan when there is a config
  // deduce loader from the mods
  // if the loader doesn't match, error
  // add the existing file as local installation (can't just use add)

};
