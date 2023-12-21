import { DefaultOptions } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, getModsFolder, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { findLocalMods, getInstallation, hasInstallation } from '../lib/configurationHelper.js';
import fs from 'fs/promises';
import path from 'path';
import chalk from 'chalk';

export interface RemoveOptions extends DefaultOptions {
  dryRun: boolean;
}

export const removeAction = async (mods: string[], options: RemoveOptions, logger: Logger) => {
  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options, logger);
  const matches = findLocalMods(mods, configuration);
  const modsDir = getModsFolder(options.config, configuration);

  if (options.dryRun) {
    logger.log(chalk.yellow('Running in dry-run mode. Nothing will actually be removed.'));
  }

  for (const modToBeDeleted of matches) {
    const name = structuredClone(modToBeDeleted.name);

    if (options.dryRun) {
      logger.log(`Would have removed ${name}`);
      continue;
    }

    if (hasInstallation(modToBeDeleted, installations)) {
      const installationIndex = getInstallation(modToBeDeleted, installations);
      const filename = installations[installationIndex].fileName;

      if (!options.dryRun) {
        await fs.rm(path.resolve(modsDir, filename), { force: true });
        installations.splice(installationIndex, 1);
        /**
         * We're using structuredClone here to avoid weird reference changes under the hood.
         * This way we write an accurate snapshot of what should be written.
         * The tests have caught a weird race condition that isn't present this way.
         */
        await writeLockFile(structuredClone(installations), options, logger);
      }
    }

    const modIndex = configuration.mods.findIndex((mod) => {
      return (mod.id === modToBeDeleted.id) && (mod.type === modToBeDeleted.type);
    });

    configuration.mods.splice(modIndex, 1);
    await writeConfigFile(configuration, options, logger);

    logger.log(`Removed ${name}`);
  }
};
