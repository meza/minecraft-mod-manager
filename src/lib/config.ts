import fs from 'node:fs/promises';
import path from 'path';
import { Loader, ModInstall, ModsJson, ReleaseType } from './modlist.types.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';

export const fileExists = async (configPath: string) => {
  return await fs.access(configPath).then(
    () => true,
    () => false
  );
};

const getLockfileName = (configPath: string) => {
  return path.resolve(path.basename(configPath, path.extname(configPath)) + '-lock.json');
};

export const writeConfigFile = async (config: ModsJson, configPath: string) => {
  const configLocation = path.resolve(configPath);
  await fs.writeFile(configLocation, JSON.stringify(config, null, 2));
  // TODO handle failed write
};

export const writeLockFile = async (config: ModInstall[], configPath: string) => {
  const configLocation = getLockfileName(path.resolve(configPath));
  await fs.writeFile(configLocation, JSON.stringify(config, null, 2));
  // TODO handle failed write
};

export const readLockFile = async (configPath: string): Promise<ModInstall[]> => {
  const configLocation = getLockfileName(path.resolve(configPath));
  const configFileExists = await fileExists(configLocation);

  if (configFileExists) {
    const configContents = await fs.readFile(configLocation, {
      encoding: 'utf8'
    });
    return JSON.parse(configContents);
  }

  const emptyModLock: ModInstall[] = [];

  await writeLockFile(emptyModLock, configPath);

  return emptyModLock;

};

export const readConfigFile = async (configPath: string): Promise<ModsJson> => {
  const configLocation = path.resolve(configPath);

  if (!await fileExists(configLocation)) {
    throw new ConfigFileNotFoundException(configLocation);
  }

  const configContents = await fs.readFile(configLocation, {
    encoding: 'utf8'
  });
  return JSON.parse(configContents);
};

export const initializeConfigFile = async (configPath: string) => {
  const configLocation = path.resolve(configPath);
  const runPath = process.cwd();
  const emptyModJson = {
    loader: Loader.FABRIC,
    gameVersion: '1.19.2',
    allowVersionFallback: true,
    defaultAllowedReleaseTypes: [ReleaseType.RELEASE, ReleaseType.BETA],
    modsFolder: path.relative(runPath, './mods'),
    mods: []
  };

  await writeConfigFile(emptyModJson, configLocation);

  return emptyModJson;
};
