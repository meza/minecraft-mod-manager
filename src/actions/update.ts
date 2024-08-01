import path from 'path';
import { Logger } from '../lib/Logger.js';
import {
  ensureConfiguration,
  fileExists,
  getModsFolder,
  readLockFile,
  writeConfigFile,
  writeLockFile
} from '../lib/config.js';
import { getHash } from '../lib/hash.js';
import { Mod } from '../lib/modlist.types.js';
import { updateMod } from '../lib/updater.js';
import { DefaultOptions, telemetry } from '../mmm.js';
import { fetchModDetails } from '../repositories/index.js';
import { install } from './install.js';

import { handleFetchErrors } from '../errors/handleFetchErrors.js';
import { getInstallation, hasInstallation } from '../lib/configurationHelper.js';

export const update = async (options: DefaultOptions, logger: Logger) => {
  performance.mark('update-start');
  await install(options, logger);
  performance.mark('update-install-success');

  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options, logger);

  const installedMods = installations;
  const mods = configuration.mods;
  const modsFolder = getModsFolder(options.config, configuration);

  const processMod = async (mod: Mod, index: number) => {
    try {
      logger.debug(`[update] Checking ${mod.name} for ${mod.type}`);

      const modData = await fetchModDetails(
        mod.type,
        mod.id,
        mod.allowedReleaseTypes || configuration.defaultAllowedReleaseTypes,
        configuration.gameVersion,
        configuration.loader,
        !!mod.allowVersionFallback,
        mod.version
      );
      mods[index].name = modData.name;

      if (!hasInstallation(mod, installations)) {
        logger.error(
          `${mod.name} doesn't seem to be installed. Please delete the lock file and the mods folder and try again.`,
          1
        );
      }

      const installedModIndex = getInstallation(mod, installedMods);
      const oldModPath = path.resolve(modsFolder, installedMods[installedModIndex].fileName);

      if (!(await fileExists(oldModPath))) {
        logger.error(
          `${mod.name} (${oldModPath}) doesn't exist. Please delete the lock file and the mods folder and try again.`,
          1
        );
      }

      const installedHash = await getHash(oldModPath);
      if (modData.hash !== installedHash || modData.releaseDate > installedMods[installedModIndex].releasedOn) {
        logger.log(`${mod.name} has an update, downloading...`);
        await updateMod(modData, oldModPath, modsFolder);

        installedMods[installedModIndex].hash = modData.hash;
        installedMods[installedModIndex].downloadUrl = modData.downloadUrl;
        installedMods[installedModIndex].releasedOn = modData.releaseDate;
        installedMods[installedModIndex].fileName = modData.fileName;

        return;
      }
      return;
    } catch (error) {
      handleFetchErrors(error as Error, mod, logger);
    }
  };

  const promises = mods.map(processMod);

  await Promise.all(promises);

  await writeLockFile(installedMods, options, logger);
  await writeConfigFile(configuration, options, logger);

  performance.mark('update-succeed');
  await telemetry.captureCommand({
    command: 'update',
    success: true,
    arguments: {
      options: options
    },
    duration: performance.measure('update-duration', 'update-start', 'update-succeed').duration
  });
};
