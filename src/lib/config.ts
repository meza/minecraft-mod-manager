import fs from 'node:fs/promises';
import path from 'path';
import inquirer from 'inquirer';
import { Loader, ModInstall, ModsJson, ReleaseType } from './modlist.types.js';
import { DEFAULT_CONFIG_LOCATION } from '../mmm.js';

export const fileExists = async (configPath: string) => {
  return await fs.access(configPath).then(
    () => true,
    () => false
  );
};

const getLockfileName = (configPath: string) => {
  return path.basename(configPath, path.extname(configPath)) + '-lock.json';
};

export const writeConfigFile = async (config: ModsJson, configPath?: string) => {
  const configLocation = path.resolve(configPath || DEFAULT_CONFIG_LOCATION);
  await fs.writeFile(configLocation, JSON.stringify(config, null, 2));
};

export const writeLockFile = async (config: ModInstall[], configPath?: string) => {
  const configLocation = getLockfileName(path.resolve(configPath || DEFAULT_CONFIG_LOCATION));
  await fs.writeFile(configLocation, JSON.stringify(config, null, 2));
};

export const readLockFile = async (configPath?: string): Promise<ModInstall[]> => {
  const configLocation = getLockfileName(path.resolve(configPath || DEFAULT_CONFIG_LOCATION));
  const configFileExists = await fileExists(configLocation);

  if (configFileExists) {
    const configContents = await fs.readFile(configLocation, {
      encoding: 'utf8'
    });
    return JSON.parse(configContents);
  }

  const emptyModLock: ModInstall[] = [];

  await fs.writeFile(configLocation, JSON.stringify(emptyModLock, null, 2));

  return emptyModLock;

};

export const readConfigFile = async (configPath?: string): Promise<ModsJson> => {
  const runPath = process.cwd();
  const configLocation = path.resolve(configPath || DEFAULT_CONFIG_LOCATION);
  const configFileExists = await fileExists(configLocation);

  if (configFileExists) {
    const configContents = await fs.readFile(configLocation, {
      encoding: 'utf8'
    });
    return JSON.parse(configContents);
  }

  const answers = await inquirer.prompt([
    {
      type: 'confirm',
      name: 'create',
      default: false,
      message: `The config file: (${configLocation}) does not exist. Should we create it?`
    }
  ]);

  if (answers.create === false) {
    throw new Error('Config File does not exist');
  }

  const emptyModConfig = {
    loader: Loader.FABRIC,
    gameVersion: '1.19.2',
    allowVersionFallback: true,
    defaultAllowedReleaseTypes: [ReleaseType.RELEASE, ReleaseType.BETA],
    modsFolder: path.relative(runPath, './mods'),
    mods: []
  };

  await fs.writeFile(configLocation, JSON.stringify(emptyModConfig, null, 2));

  return emptyModConfig;

};
