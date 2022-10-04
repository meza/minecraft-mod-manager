import { DefaultOptions } from '../mmm.js';
import { fileExists, readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { Mod, ModInstall, RemoteModDetails } from '../lib/modlist.types.js';
import path from 'path';
import { downloadFile } from '../lib/downloader.js';
import { getHash } from '../lib/hash.js';
import { updateMod } from '../lib/updater.js';
import { fetchModDetails } from '../repositories/index.js';

const getMod = async (moddata: RemoteModDetails, modsFolder: string) => {
  await downloadFile(moddata.downloadUrl, path.resolve(modsFolder, moddata.fileName));
  return {
    fileName: moddata.fileName,
    releasedOn: moddata.releaseDate,
    hash: moddata.hash,
    downloadUrl: moddata.downloadUrl
  };
};
const getInstallation = (mod: Mod, installations: ModInstall[]) => {
  return installations.findIndex((i) => i.id === mod.id && i.type === mod.type);
};

const hasInstallation = (mod: Mod, installations: ModInstall[]) => {
  return getInstallation(mod, installations) > -1;
};

export const update = async (options: DefaultOptions) => {
  const configuration = await readConfigFile(options.config);
  const installations = await readLockFile(options.config);

  const installedMods = installations;
  const mods = configuration.mods;

  const processMod = async (mod: Mod, index: number) => {

    if (options.debug) {
      console.debug(`Checking ${mod.name} for ${mod.type}`);
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
    const x = hasInstallation(mod, installations);
    if (x) {
      const installedModIndex = getInstallation(mod, installedMods);

      const modPath = path.resolve(configuration.modsFolder, installedMods[installedModIndex].fileName);

      if (!await fileExists(modPath)) {
        console.log(`${mod.name} (${modPath}) doesn't exist, downloading from ${installedMods[installedModIndex].type}`);
        await downloadFile(modData.downloadUrl, modPath);
        // TODO handle the download failing

        installedMods[installedModIndex].hash = modData.hash;
        installedMods[installedModIndex].downloadUrl = modData.downloadUrl;
        installedMods[installedModIndex].releasedOn = modData.releaseDate;
        installedMods[installedModIndex].fileName = modData.fileName;
        return;
      }

      const installedHash = await getHash(modPath);
      if (modData.hash !== installedHash || modData.releaseDate > installedMods[installedModIndex].releasedOn) {
        console.log(`${mod.name} has an update, downloading...`);
        await updateMod(modData, modPath, configuration.modsFolder);
        // TODO handle the download failing

        installedMods[installedModIndex].hash = modData.hash;
        installedMods[installedModIndex].downloadUrl = modData.downloadUrl;
        installedMods[installedModIndex].releasedOn = modData.releaseDate;
        installedMods[installedModIndex].fileName = modData.fileName;

        return;
      }
      return;
    }

    mods[index].name = modData.name;

    // no installation exists
    console.log(`${mod.name} (in the lock file) doesn't exist, downloading from ${mod.type}`);
    const dlData = await getMod(modData, configuration.modsFolder);
    // TODO handle the download failing
    installedMods.push({
      name: modData.name,
      type: mod.type,
      id: mod.id,
      fileName: dlData.fileName,
      releasedOn: dlData.releasedOn,
      hash: dlData.hash,
      downloadUrl: dlData.downloadUrl
    });
    return;

  };

  const promises = mods.map(processMod);

  await Promise.all(promises);

  await writeLockFile(installedMods, options.config);
  await writeConfigFile(configuration, options.config);
};
