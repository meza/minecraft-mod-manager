import path from 'path';
import * as fs from 'fs/promises';
import { notIgnored } from './ignore.js';
import { getModsDir } from './configurationHelper.js';

export const getModFiles = async (configLocation: string, modsFolder: string) => {
  const dir = path.resolve(path.dirname(configLocation));
  const modsDir = getModsDir(configLocation, modsFolder);
  const modFileNames = await fs.readdir(modsDir);
  const files = modFileNames.map((file) => {
    return path.resolve(modsDir, file);
  });

  if (files.length === 0) {
    return [];
  }

  const notIgnoredFiles = await notIgnored(dir, files);

  return notIgnoredFiles;
};
