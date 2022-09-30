import path from 'path';
import { fetchModDetails } from '../repositories/index.js';
import { fileExists, readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { downloadFile } from '../lib/downloader.js';
import { Mod, ModInstall, RemoteModDetails } from '../lib/modlist.types.js';
import { getHash } from '../lib/hash.js';
import { DefaultOptions } from '../mmm.js';
import { updateMod } from '../lib/updater.js';

interface UpdateOptions extends DefaultOptions {
}

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

export const install = async (options: UpdateOptions) => {
  const configuration = await readConfigFile(options.config);
  const installations = await readLockFile(options.config);

  const installedMods = installations;
  const mods = configuration.mods;

  const processMod = async (mod: Mod, index: number) => {

    const modData = await fetchModDetails(
      mod.type,
      mod.id,
      configuration.defaultAllowedReleaseTypes,
      configuration.gameVersion,
      configuration.loader,
      configuration.allowVersionFallback
    );

    mods[index].name = modData.name;

    if (!hasInstallation(mod, installedMods)) { // no installation exists
      console.log(`${mod.id} doesn't exist, downloading from ${mod.type}`);
      const dlData = await getMod(modData, configuration.modsFolder);
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
    }

    // Installation exists

    const installedModIndex = getInstallation(mod, installedMods);

    const modPath = path.resolve(configuration.modsFolder, installations[installedModIndex].fileName);

    if (!await fileExists(modPath)) {
      console.log(`${installations[installedModIndex].name} doesn't exist, downloading from ${installations[installedModIndex].type}`);
      const dlData = await getMod(modData, configuration.modsFolder);

      installations[installedModIndex].fileName = dlData.fileName;
      installations[installedModIndex].releasedOn = dlData.releasedOn;
      installations[installedModIndex].hash = dlData.hash;
      installations[installedModIndex].downloadUrl = modData.downloadUrl;
      return;
    }

    const installedHash = await getHash(modPath);

    if (modData.hash !== installedHash) {
      console.log(`${installations[installedModIndex].name}  has hash mismatch, downloading from source`);
      const freshMod = await updateMod(installations[installedModIndex], modPath, modData, configuration.modsFolder);

      installations[installedModIndex].fileName = freshMod.fileName;
      installations[installedModIndex].releasedOn = freshMod.releasedOn;
      installations[installedModIndex].hash = freshMod.hash;
      installations[installedModIndex].downloadUrl = freshMod.downloadUrl;
      installations[installedModIndex].name = freshMod.name;

      return;
    }

  };

  const promises = mods.map(processMod);

  await Promise.all(promises);

  await writeLockFile(installedMods, options.config);
  await writeConfigFile(configuration, options.config);
};
