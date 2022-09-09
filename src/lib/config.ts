import fs from 'node:fs/promises';
import path from 'path';
import { DEFAULT_CONFIG_LOCATION } from '../index.js';
import inquirer from 'inquirer';
import { Loader, ModlistConfig, ReleaseType } from './modlist.types.js';

export const fileExists = async (configPath: string) => {
  return await fs.access(configPath).then(
    () => true,
    () => false
  );
};

export const writeConfigFile = async (config: ModlistConfig, configPath?: string) => {
  const configLocation = path.resolve(configPath || DEFAULT_CONFIG_LOCATION);
  await fs.writeFile(configLocation, JSON.stringify(config, null, 2));
};

export const readConfigFile = async (configPath?: string): Promise<ModlistConfig> => {
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
