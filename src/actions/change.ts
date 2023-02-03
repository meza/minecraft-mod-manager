import { VerifyUpgradeOptions } from '../lib/verifyUpgrade.js';
import { Logger } from '../lib/Logger.js';
import { testGameVersion } from './testGameVersion.js';
import { Mod } from '../lib/modlist.types.js';
import {
  ensureConfiguration,
  fileExists, readLockFile,
  writeConfigFile,
  writeLockFile
} from '../lib/config.js';
import path from 'path';
import fs from 'node:fs/promises';
import { install } from './install.js';
import { getInstallation, hasInstallation } from '../lib/configurationHelper.js';

export const changeGameVersion = async (gameVersion: string, options: VerifyUpgradeOptions, logger: Logger) => {
  const { version } = await testGameVersion(gameVersion, options, logger);

  const configuration = await ensureConfiguration(options.config, logger, options.quiet);
  const installations = await readLockFile(options.config);

  const installedMods = installations;
  const mods = configuration.mods;
  const removeLocalFile = async (mod: Mod) => {

    if (hasInstallation(mod, installations)) {
      const installedModIndex = getInstallation(mod, installedMods);
      const oldModPath = path.resolve(configuration.modsFolder, installedMods[installedModIndex].fileName);
      if (await fileExists(oldModPath)) {
        await fs.rm(oldModPath);
      }
    }
  };

  const promises = mods.map(removeLocalFile);

  await Promise.allSettled(promises);

  configuration.gameVersion = version;

  await writeLockFile([], options.config);
  await writeConfigFile(configuration, options.config);

  await install(options, logger);

};
