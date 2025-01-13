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

  //remove all directories
  for (let i = 0; i < files.length; i++) {
    const isDir = await isDirectory(files[i]);
    if (isDir) {
      files.splice(i, 1);
      i--;
    }
  }

  if (files.length === 0) {
    return [];
  }

  const notIgnoredFiles = await notIgnored(dir, files);

  return notIgnoredFiles;
};

const isDirectory = async (filePath: string): Promise<boolean> => {
  try {
    const stat = await fs.stat(filePath);
    return stat.isDirectory();
  } catch (error) {
    console.error(`Error checking if path is a directory: ${(error as Error).message}`);
    return false;
  }
};
