import fs from 'node:fs/promises';
import path from 'path';
import { downloadFile } from './downloader.js';
import { ModInstall, RemoteModDetails } from './modlist.types.js';
import { Logger } from './Logger.js';

export const updateMod = async (
  mod: ModInstall | RemoteModDetails,
  modPath: string,
  modsFolder: string,
  logger: Logger
): Promise<ModInstall | RemoteModDetails> => {

  await fs.rename(modPath, `${modPath}.bak`);
  // Todo handle bak file existing or rename not working

  try {
    const newPath = path.resolve(modsFolder, mod.fileName);
    await downloadFile(mod.downloadUrl, newPath);
    await fs.rm(`${modPath}.bak`);

  } catch {
    logger.log(`Download of ${mod.name} failed, restoring the original`);
    // Todo handle the error
    await fs.rename(`${modPath}.bak`, modPath);
    // Todo handle bak file existing or rename not working
  }

  return mod;

};
