import fs from 'node:fs/promises';
import path from 'path';
import { downloadFile } from './downloader.js';
import { ModInstall, RemoteModDetails } from './modlist.types.js';

export const updateMod = async (
  mod: ModInstall | RemoteModDetails,
  modPath: string,
  modsFolder: string
): Promise<ModInstall | RemoteModDetails> => {

  const newPath = path.resolve(modsFolder, mod.fileName);
  await downloadFile(mod.downloadUrl, newPath);
  if (modPath !== newPath) {
    await fs.rm(modPath);
  }
  return mod;

};
