import path from 'path';
import * as fs from 'fs/promises';
import { getModsFolder } from './config.js';
import { notIgnored } from './ignore.js';
import { ModsJson } from './modlist.types.js';

export const getModFiles = async (configLocation: string, configuration: ModsJson) => {
  const dir = path.resolve(path.dirname(configLocation));
  const modsDir = getModsFolder(configLocation, configuration);
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
