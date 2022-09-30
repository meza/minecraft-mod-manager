import fs from 'node:fs/promises';
import path from 'path';
import { downloadFile } from './downloader.js';
import { ModInstall, RemoteModDetails } from './modlist.types.js';

export const updateMod = async (
  mod: ModInstall,
  modPath: string,
  modData: RemoteModDetails,
  modsFolder: string
): Promise<ModInstall> => {

  await fs.rename(modPath, `${modPath}.bak`);

  try {
    const newPath = path.resolve(modsFolder, modData.fileName);
    await downloadFile(modData.downloadUrl, newPath);

    mod.name = modData.name;
    mod.fileName = modData.fileName;
    mod.releasedOn = modData.releaseDate;
    mod.hash = modData.hash;
    mod.downloadUrl = modData.downloadUrl;

    await fs.rm(`${modPath}.bak`);

  } catch {
    console.log('Download failed, restoring the original');
    await fs.rename(`${modPath}.bak`, modPath);
  }

  return mod;

};
