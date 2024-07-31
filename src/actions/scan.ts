import { DefaultOptions, telemetry } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, getModsFolder, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { PlatformLookupResult } from '../repositories/index.js';
import { Mod, ModInstall, ModsJson, Platform, RemoteModDetails } from '../lib/modlist.types.js';
import { scan as scanLib } from '../lib/scan.js';
import chalk from 'chalk';
import { shouldAddScanResults } from '../interactions/shouldAddScanResults.js';

import { fileIsManaged, getInstallation } from '../lib/configurationHelper.js';
import { getModFiles } from '../lib/fileHelper.js';
import path from 'path';

export interface ScanOptions extends DefaultOptions {
  prefer: Platform;
  add: boolean;
}

export interface ScanResults {
  preferredDetails: RemoteModDetails;
  allRemoteDetails: RemoteModDetails[];
  localDetails: PlatformLookupResult[];
}

export interface FoundEntries {
  mod: Mod;
  install: ModInstall;
}

export interface UnsureEntries {
  configuredMod: number;
  configuredInstallation: number;
  newMod: Mod;
  installation: ModInstall;
}

const findInConfiguration = (platform: Platform, modId: string, configuration: ModsJson) => {
  return configuration.mods.findIndex((configuredMod) => {
    return (configuredMod.type === platform) && (configuredMod.id === modId);
  });
};

const processForeignFiles = async (
  options: ScanOptions,
  configuration: ModsJson,
  installations: ModInstall[],
  dealtWith: string[],
  hasResults: boolean,
  logger: Logger
) => {
  const modsFolder = getModsFolder(options.config, configuration);
  const allFiles = await getModFiles(options.config, configuration);

  const nonMatchedFiles = allFiles.filter((filePath) => {
    if (fileIsManaged(filePath, installations)) {
      return false;
    }
    const foundIndex = dealtWith.findIndex((dealtWithFile) => {
      return path.resolve(modsFolder, dealtWithFile) === filePath;
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

export const processScanResults = (scanResults: ScanResults[], configuration: ModsJson, installations: ModInstall[], logger: Logger) => {
  const unmanaged: FoundEntries[] = [];
  const unsure: UnsureEntries[] = [];

  scanResults.forEach((hit) => {

    // ================================================================================================================
    // We know that `hit` has no direct matches in the lockfile, so we need to look for a configuration match

    const halfMatching = hit.localDetails.map((local, index) => {
      const modIndex = findInConfiguration(local.platform, local.modId, configuration);
      if (modIndex < 0) {
        return {
          found: false,
          configured: null,
          remote: hit.allRemoteDetails[index],
          local: local
        };
      }

      return {
        found: true,
        configured: configuration.mods[modIndex],
        remote: hit.allRemoteDetails[index],
        local: local
      };
    }).filter((result) => {
      return result.found === true;
    });

    if (halfMatching.length > 0 && halfMatching[0].configured) {
      const installationIndex = getInstallation(halfMatching[0].configured, installations);

      if (installationIndex > -1) {
        const installation = installations[installationIndex];
        if (halfMatching[0].local.mod.hash !== installation.hash) {
          logger.log(`${chalk.red('\u274c')} ${halfMatching[0].remote.name} has a different version locally than what is in the lockfile`);
        }
      } else {
        logger.log(`${chalk.red('\u274c')} ${halfMatching[0].remote.name} has a local file that isn't in the lockfile.`);
      }

      unsure.push({
        configuredMod: findInConfiguration(halfMatching[0].configured?.type, halfMatching[0].configured?.id, configuration),
        configuredInstallation: installationIndex,
        newMod: {
          type: halfMatching[0].local.platform,
          id: halfMatching[0].local.modId,
          name: halfMatching[0].remote.name
        } as Mod,
        installation: {
          hash: halfMatching[0].local.mod.hash,
          fileName: halfMatching[0].local.mod.fileName,
          name: halfMatching[0].remote.name,
          type: halfMatching[0].local.platform,
          id: halfMatching[0].local.modId,
          releasedOn: halfMatching[0].local.mod.releaseDate,
          downloadUrl: halfMatching[0].local.mod.downloadUrl
        } as ModInstall
      });
    }

    // ================================================================================================================

    if (halfMatching.length === 0) {
      const message = chalk.green('\u2705') + `Found unmanaged mod: ${chalk.bold(chalk.whiteBright(hit.preferredDetails.name))}`;
      logger.log(message, true);

      unmanaged.push(
        {
          mod: {
            type: hit.localDetails[0].platform,
            id: hit.localDetails[0].modId,
            name: hit.preferredDetails.name
          },
          install: {
            name: hit.preferredDetails.name,
            type: hit.localDetails[0].platform,
            id: hit.localDetails[0].modId,
            fileName: hit.localDetails[0].mod.fileName,
            hash: hit.localDetails[0].mod.hash,
            downloadUrl: hit.localDetails[0].mod.downloadUrl,
            releasedOn: hit.localDetails[0].mod.releaseDate
          }
        });
    }
  });
  return {
    unmanaged: unmanaged,
    unsure: unsure
  };
};

export const scan = async (options: ScanOptions, logger: Logger) => {
  performance.mark('scan-start');
  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options, logger);
  let scanResults: ScanResults[] = [];

  try {
    scanResults = await scanLib(options.config, options.prefer, configuration, installations);
  } catch (error) {
    logger.error((error as Error).message, 2);
  }

  const { unmanaged, unsure } = processScanResults(scanResults, configuration, installations, logger);

  const hasResults = (scanResults.length > 0);
  if (hasResults && await shouldAddScanResults(options, logger)) {

    unmanaged.forEach(({ mod, install }) => {
      configuration.mods.push(mod);
      installations.push(install);
    });

    unsure.forEach((result) => {
      configuration.mods[result.configuredMod] = result.newMod;

      if (result.configuredInstallation > -1) {
        installations[result.configuredInstallation] = result.installation;
      } else {
        installations.push(result.installation);
      }

      logger.log(`${chalk.green('\u2705')} Updated ${configuration.mods[result.configuredMod].name} to match the installed file`);

    });

    await writeConfigFile(configuration, options, logger);
    await writeLockFile(installations, options, logger);
  }

  const dealtWith: string[] = [];

  unsure.forEach((managed) => {
    dealtWith.push(managed.installation.fileName);
  });

  unmanaged.forEach((managed) => {
    dealtWith.push(managed.install.fileName);
  });

  await processForeignFiles(options, configuration, installations, dealtWith, hasResults, logger);

  performance.mark('scan-succeed');

  await telemetry.captureCommand({
    command: 'scan',
    success: true,
    arguments: {
      options: options
    },
    duration: performance.measure('scan-duration', 'scan-start', 'scan-succeed').duration
  });

};
