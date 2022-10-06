import { DefaultOptions } from '../mmm.js';
import { fileExists, readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { Mod, ModInstall } from '../lib/modlist.types.js';
import path from 'path';
import { getHash } from '../lib/hash.js';
import { updateMod } from '../lib/updater.js';
import { fetchModDetails } from '../repositories/index.js';
import { install } from './install.js';

const getInstallation = (mod: Mod, installations: ModInstall[]) => {
  return installations.findIndex((i) => i.id === mod.id && i.type === mod.type);
};

const hasInstallation = (mod: Mod, installations: ModInstall[]) => {
  return getInstallation(mod, installations) > -1;
};

export const update = async (options: DefaultOptions) => {
  await install(options);
  const configuration = await readConfigFile(options.config);
  const installations = await readLockFile(options.config);

  const installedMods = installations;
  const mods = configuration.mods;

  const processMod = async (mod: Mod, index: number) => {

    if (options.debug) {
      console.debug(`[update] Checking ${mod.name} for ${mod.type}`);
    }

    const modData = await fetchModDetails(
      mod.type,
      mod.id,
      configuration.defaultAllowedReleaseTypes,
      configuration.gameVersion,
      configuration.loader,
      configuration.allowVersionFallback
    );
    // TODO Handle the fetch failing
    mods[index].name = modData.name;

    if (!hasInstallation(mod, installations)) {
      console.error(`${mod.name} doesn't seem to be installed, please run mmm install first`);
      return;
    }

    const installedModIndex = getInstallation(mod, installedMods);
    const oldModPath = path.resolve(configuration.modsFolder, installedMods[installedModIndex].fileName);

    if (!await fileExists(oldModPath)) {
      console.error(`${mod.name} (${oldModPath}) doesn't exist, please run mmm install`);
      return;
    }

    const installedHash = await getHash(oldModPath);
    if (modData.hash !== installedHash || modData.releaseDate > installedMods[installedModIndex].releasedOn) {
      console.log(`${mod.name} has an update, downloading...`);
      await updateMod(modData, oldModPath, configuration.modsFolder);
      // TODO handle the download failing

      installedMods[installedModIndex].hash = modData.hash;
      installedMods[installedModIndex].downloadUrl = modData.downloadUrl;
      installedMods[installedModIndex].releasedOn = modData.releaseDate;
      installedMods[installedModIndex].fileName = modData.fileName;

      return;
    }
    return;
  };

  const promises = mods.map(processMod);

  await Promise.all(promises);

  await writeLockFile(installedMods, options.config);
  await writeConfigFile(configuration, options.config);
};
