import { DefaultOptions } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { PlatformLookupResult } from '../repositories/index.js';
import { Platform, RemoteModDetails } from '../lib/modlist.types.js';
import { scan as scanLib } from '../lib/scan.js';
import chalk from 'chalk';

export interface ScanOptions extends DefaultOptions {
  prefer: Platform;
  add: boolean;
}

export interface ScanResults {
  resolvedDetails: RemoteModDetails;
  localDetails: PlatformLookupResult;
}

export const scan = async (options: ScanOptions, logger: Logger) => {
  const configuration = await ensureConfiguration(options.config);
  const installations = await readLockFile(options.config);

  const scanResults = await scanLib(options.prefer, configuration, installations);

  if (scanResults.length === 0) {
    logger.log('You have no unmanaged mods in your mods folder.');
    return;
  }

  scanResults.forEach((hit) => {
    if (!options.quiet && !options.add) {
      const message = chalk.green('\u2705') + `Found unmanaged mod: ${chalk.bold(chalk.whiteBright(hit.resolvedDetails.name))}`;
      logger.log(message);
    }

    if (options.add) {
      configuration.mods.push({
        type: hit.localDetails.platform,
        id: hit.localDetails.modId,
        name: hit.resolvedDetails.name
      });

      installations.push({
        name: hit.resolvedDetails.name,
        type: hit.localDetails.platform,
        id: hit.localDetails.modId,
        fileName: hit.localDetails.mod.fileName,
        hash: hit.localDetails.mod.hash,
        downloadUrl: hit.localDetails.mod.downloadUrl,
        releasedOn: hit.localDetails.mod.releaseDate
      });

      logger.log(`${chalk.green('\u2705')} Added ${chalk.bold(chalk.cyanBright(hit.resolvedDetails.name))} from ${chalk.bold(chalk.yellow(hit.localDetails.platform))}`);
    }

  });
  if (!options.add) {
    logger.log('\n' + chalk.yellow('Use the --add flag to add these mod to your modlist.'));
    return;
  }

  await writeConfigFile(configuration, options.config);
  await writeLockFile(installations, options.config);

  // scan during init
  // scan when there is a config
  // deduce loader from the mods
  // if the loader doesn't match, error
  // add the existing file as local installation (can't just use add)

};
