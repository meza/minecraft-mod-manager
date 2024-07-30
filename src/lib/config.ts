import fs from 'node:fs/promises';
import path from 'path';
import { ModInstall, ModsJson } from './modlist.types.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { initializeConfig } from '../interactions/initializeConfig.js';
import { shouldCreateConfig } from '../interactions/shouldCreateConfig.js';
import { Logger } from './Logger.js';
import { fileToWrite } from '../interactions/fileToWrite.js';
import { DefaultOptions } from '../mmm.js';

export const fileExists = async (configPath: string) => {
  return await fs.access(configPath).then(
    () => true,
    () => false
  );
};

const getLockfileName = (configPath: string) => {
  return path.resolve(path.basename(configPath, path.extname(configPath)) + '-lock.json');
};

export const writeConfigFile = async (config: ModsJson, options: DefaultOptions, logger: Logger) => {
  const configLocation = path.resolve(options.config);
  const fileToUse = await fileToWrite(configLocation, options, logger);
  await fs.writeFile(fileToUse, JSON.stringify(config, null, 2));
};

export const writeLockFile = async (config: ModInstall[], options: DefaultOptions, logger: Logger) => {
  const configLocation = getLockfileName(path.resolve(options.config));
  const fileToUse = await fileToWrite(configLocation, options, logger);
  await fs.writeFile(fileToUse, JSON.stringify(config, null, 2));
};

export const readLockFile = async (options: DefaultOptions, logger: Logger): Promise<ModInstall[]> => {
  const lockFileLocation = getLockfileName(path.resolve(options.config));
  const lockFileExists = await fileExists(lockFileLocation);

  if (lockFileExists) {
    const configContents = await fs.readFile(lockFileLocation, {
      encoding: 'utf8'
    });
    return JSON.parse(configContents);
  }

  const emptyModLock: ModInstall[] = [];

  await writeLockFile(emptyModLock, options, logger);

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
  performance.mark('ensure-configuration-start');
  try {
    const result = await readConfigFile(configPath);
    performance.mark('ensure-configuration-succeed');
    return result;
  } catch (error) {
    if (error instanceof ConfigFileNotFoundException && !quiet) {
      performance.mark('ensure-configuration-fail');
      if (await shouldCreateConfig(configPath)) {
        return await initializeConfigFile(configPath, logger);
      }
    }
    throw error;
  }
};

export const getModsFolder = (configLocation: string, config: ModsJson): string => {
  const realConfigLocation = path.resolve(configLocation);
  const configFolder = path.dirname(realConfigLocation);
  const configuredModsFolder = config.modsFolder;

  if (path.isAbsolute(configuredModsFolder)) {
    return configuredModsFolder;
  }

  return path.resolve(configFolder, configuredModsFolder);

};
