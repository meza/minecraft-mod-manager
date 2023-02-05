import { DefaultOptions } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { PlatformLookupResult } from '../repositories/index.js';
import { Mod, ModInstall, ModsJson, Platform, RemoteModDetails } from '../lib/modlist.types.js';
import { scan as scanLib } from '../lib/scan.js';
import chalk from 'chalk';
import { shouldAddScanResults } from '../interactions/shouldAddScanResults.js';
import { getModFiles } from '../lib/fileHelper.js';
import { fileIsManaged, getModsDir } from '../lib/configurationHelper.js';
import path from 'path';

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

const processForeignFiles = async (options: ScanOptions, configuration: ModsJson, installations: ModInstall[], scanResults: ScanResults[], hasResults: boolean, logger: Logger) => {
  const modsFolder = getModsDir(options.config, configuration.modsFolder);
  const allFiles = await getModFiles(options.config, configuration.modsFolder);
  const nonManagedFiles = allFiles.filter((filePath) => {
    return !fileIsManaged(filePath, installations);
  });

  const nonMatchedFiles = nonManagedFiles.filter((filePath) => {
    const foundIndex = scanResults.findIndex((scanResult) => {
      return path.resolve(modsFolder, scanResult.localDetails.mod.fileName) === filePath;
    });
    return foundIndex < 0;
  });
  const hasForeignFiles = (nonMatchedFiles.length > 0);

  if ((!hasResults) && (!hasForeignFiles)) {
    logger.log(`${chalk.green('\u2705')} All of your mods are managed by mmm.`);
    return;
  }

  if (!hasResults) {
    logger.log(`${chalk.green('\u2705')} Every mod that can be matched are managed by mmm.`);
  }

  if (hasForeignFiles) {
    logger.log('\nThe following files cannot be matched to any mod on any of the platforms:\n');
    nonMatchedFiles.forEach((file) => {
      logger.log(`  ${chalk.red('\u274c')} ${file}`);
    });
  }
};

export const scan = async (options: ScanOptions, logger: Logger) => {
  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options, logger);
  let scanResults: ScanResults[] = [];

  try {
    scanResults = await scanLib(options.config, options.prefer, configuration, installations);
  } catch (error) {
    logger.error((error as Error).message, 2);
  }

  const hasResults = (scanResults.length > 0);
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

  if (hasResults && await shouldAddScanResults(options, logger)) {

    found.forEach(({ mod, install }) => {
      configuration.mods.push(mod);
      installations.push(install);
    });

    await writeConfigFile(configuration, options, logger);
    await writeLockFile(installations, options, logger);
  }

  await processForeignFiles(options, configuration, installations, scanResults, hasResults, logger);

  // scan during init
  // deduce loader from the mods
  // if the loader doesn't match, error

};
