import fs from 'node:fs/promises';
import path from 'path';
import { downloadFile } from './downloader.js';
import { ModInstall } from './modlist.types.js';

export const updateMod = async (
  mod: ModInstall,
  modPath: string,
  modsFolder: string
): Promise<ModInstall> => {

  await fs.rename(modPath, `${modPath}.bak`);

  try {
    const newPath = path.resolve(modsFolder, mod.fileName);
    await downloadFile(mod.downloadUrl, newPath);
    await fs.rm(`${modPath}.bak`);

  } catch {
    console.log(`Download of ${mod.name} failed, restoring the original`);
    await fs.rename(`${modPath}.bak`, modPath);
  }

  return mod;

};
