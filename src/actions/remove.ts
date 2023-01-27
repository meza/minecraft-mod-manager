import { DefaultOptions } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { findLocalMods, getInstallation, getModsDir, hasInstallation } from '../lib/configurationHelper.js';
import fs from 'fs/promises';
import path from 'path';
import chalk from 'chalk';

export interface RemoveOptions extends DefaultOptions {
  dryRun: boolean;
}

export const removeAction = async (mods: string[], options: RemoveOptions, logger: Logger) => {
  const configuration = await ensureConfiguration(options.config);
  const installations = await readLockFile(options.config);
  const matches = findLocalMods(mods, configuration);
  const modsDir = getModsDir(options.config, configuration.modsFolder);

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
        await writeLockFile(installations, options.config);
      }
    }

    const modIndex = configuration.mods.findIndex((mod) => {
      return (mod.id === modToBeDeleted.id) && (mod.type === modToBeDeleted.type);
    });

    configuration.mods.splice(modIndex, 1);
    await writeConfigFile(configuration, options.config);

    logger.log(`Removed ${name}`);
  }
};
