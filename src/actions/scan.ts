import { DefaultOptions } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { PlatformLookupResult } from '../repositories/index.js';
import { Mod, ModInstall, Platform, RemoteModDetails } from '../lib/modlist.types.js';
import { scan as scanLib } from '../lib/scan.js';
import chalk from 'chalk';
import { shouldAddScanResults } from '../interactions/shouldAddScanResults.js';

export interface ScanOptions extends DefaultOptions {
  prefer: Platform;
  add: boolean;
}

export interface ScanResults {
  resolvedDetails: RemoteModDetails;
  localDetails: PlatformLookupResult;
}

interface FoundEntries {
  mod: Mod;
  install: ModInstall;
}

export const scan = async (options: ScanOptions, logger: Logger) => {
  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options.config);
  let scanResults: ScanResults[] = [];

  try {
    scanResults = await scanLib(options.config, options.prefer, configuration, installations);
  } catch (error) {
    logger.error((error as Error).message, 2);
  }

  if (scanResults.length === 0) {
    logger.log('You have no unmanaged mods in your mods folder.');
    return;
  }

  const found: FoundEntries[] = [];

  scanResults.forEach((hit) => {
    const message = chalk.green('\u2705') + `Found unmanaged mod: ${chalk.bold(chalk.whiteBright(hit.resolvedDetails.name))}`;
    logger.log(message, true);

    found.push(
      {
        mod: {
          type: hit.localDetails.platform,
          id: hit.localDetails.modId,
          name: hit.resolvedDetails.name
        },
        install: {
          name: hit.resolvedDetails.name,
          type: hit.localDetails.platform,
          id: hit.localDetails.modId,
          fileName: hit.localDetails.mod.fileName,
          hash: hit.localDetails.mod.hash,
          downloadUrl: hit.localDetails.mod.downloadUrl,
          releasedOn: hit.localDetails.mod.releaseDate
        }
      });
  });

  if (await shouldAddScanResults(options, logger)) {

    found.forEach(({ mod, install }) => {
      configuration.mods.push(mod);
      installations.push(install);
    });

    await writeConfigFile(configuration, options.config);
    await writeLockFile(installations, options.config);
  }

  // scan during init
  // deduce loader from the mods
  // if the loader doesn't match, error

};
