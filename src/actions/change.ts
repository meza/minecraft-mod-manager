import { VerifyUpgradeOptions } from '../lib/verifyUpgrade.js';
import { Logger } from '../lib/Logger.js';
import { telemetry } from '../mmm.js';
import { testGameVersion } from './testGameVersion.js';
import { Mod } from '../lib/modlist.types.js';
import {
  ensureConfiguration,
  fileExists, getModsFolder, readLockFile,
  writeConfigFile,
  writeLockFile
} from '../lib/config.js';
import path from 'path';
import fs from 'node:fs/promises';
import { install } from './install.js';
import { getInstallation, hasInstallation } from '../lib/configurationHelper.js';

export const changeGameVersion = async (gameVersion: string, options: VerifyUpgradeOptions, logger: Logger) => {
  performance.mark('change-start');
  const { version } = await testGameVersion(gameVersion, options, logger);
  performance.mark('change-version-succeed');

  const configuration = await ensureConfiguration(options.config, logger, options.quiet);
  const installations = await readLockFile(options, logger);

  const installedMods = installations;
  const mods = configuration.mods;
  const removeLocalFile = async (mod: Mod) => {

    if (hasInstallation(mod, installations)) {
      const installedModIndex = getInstallation(mod, installedMods);
      const oldModPath = path.resolve(getModsFolder(options.config, configuration), installedMods[installedModIndex].fileName);
      if (await fileExists(oldModPath)) {
        await fs.rm(oldModPath);
      }
    }
  };

  const promises = mods.map(removeLocalFile);

  await Promise.allSettled(promises);

  configuration.gameVersion = version;

  await writeLockFile([], options, logger);
  await writeConfigFile(configuration, options, logger);

  await install(options, logger);

  performance.mark('change-succeed');

  await telemetry.captureCommand({
    command: 'change',
    success: true,
    arguments: {
      options: options,
      gameVersion: gameVersion
    },
    extra: {
      versionTestDuration: performance.measure('change-version-duration', 'change-start', 'change-version-succeed').duration
    },
    duration: performance.measure('change-duration', 'change-start', 'change-succeed').duration
  });

};
