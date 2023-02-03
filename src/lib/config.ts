import fs from 'node:fs/promises';
import path from 'path';
import { ModInstall, ModsJson } from './modlist.types.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { initializeConfig } from '../interactions/initializeConfig.js';
import { shouldCreateConfig } from '../interactions/shouldCreateConfig.js';
import { Logger } from './Logger.js';

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
  const lockFileLocation = getLockfileName(path.resolve(configPath));
  const lockFileExists = await fileExists(lockFileLocation);

  if (lockFileExists) {
    const configContents = await fs.readFile(lockFileLocation, {
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

export const initializeConfigFile = async (configPath: string, logger: Logger): Promise<ModsJson> => {
  const runPath = process.cwd();
  const emptyModJson = await initializeConfig({
    config: configPath
  }, runPath, logger);

  return emptyModJson;
};

export const ensureConfiguration = async (configPath: string, logger: Logger, quiet = false): Promise<ModsJson> => {
  try {
    return await readConfigFile(configPath);
  } catch (error) {
    if (error instanceof ConfigFileNotFoundException && !quiet) {
      if (await shouldCreateConfig(configPath)) {
        return await initializeConfigFile(configPath, logger);
      }
    }
    throw error;
  }
};
