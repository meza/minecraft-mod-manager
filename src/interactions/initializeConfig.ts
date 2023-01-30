import { DefaultOptions } from '../mmm.js';
import { Loader, ModsJson, ReleaseType } from '../lib/modlist.types.js';
import inquirer from 'inquirer';
import { fileExists, writeConfigFile } from '../lib/config.js';
import * as path from 'path';
import { getLatestMinecraftVersion, verifyMinecraftVersion } from '../lib/minecraftVersionVerifier.js';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import { configFile } from './configFileOverwrite.js';

export interface InitializeOptions extends DefaultOptions {
  loader?: Loader,
  gameVersion?: string,
  allowVersionFallback?: boolean,
  defaultAllowedReleaseTypes?: string,
  modsFolder?: string
}

interface IQInternal extends ModsJson {
  config: string
}

const validateGameVersion = async (input: string): Promise<boolean | string> => {
  if (await verifyMinecraftVersion(input)) {
    return true;
  }
  return 'The game version is invalid. Please enter a valid game version';
};

const mergeOptions = (options: InitializeOptions, iq: IQInternal) => {
  return {
    loader: iq.loader || options.loader,
    gameVersion: iq.gameVersion || options.gameVersion,
    allowVersionFallback: iq.allowVersionFallback || options.allowVersionFallback,
    defaultAllowedReleaseTypes: iq.defaultAllowedReleaseTypes || options.defaultAllowedReleaseTypes?.replace(/\s/g, '').split(','),
    modsFolder: iq.modsFolder || options.modsFolder,
    mods: []
  };
};

const validateModsFolder = async (input: string, cwd: string) => {
  const dir = path.resolve(cwd, input);
  if (!await fileExists(dir)) {
    return `The folder: ${dir} does not exist. Please enter a valid one and try again.`;
  }
  return true;
};

const validateInput = async (options: InitializeOptions, cwd: string) => {
  if (options.gameVersion) {
    if (!await verifyMinecraftVersion(options.gameVersion)) {
      throw new IncorrectMinecraftVersionException(options.gameVersion);
    }
  }
  if (options.modsFolder) {
    const result = await validateModsFolder(options.modsFolder, cwd);
    if (result !== true) {
      throw new Error(result);
    }
  }
};

export const initializeConfig = async (options: InitializeOptions, cwd: string): Promise<ModsJson> => {
  await validateInput(options, cwd);

  options.config = await configFile(options, cwd);

  const prompts = [
    {
      when: !options.loader,
      name: 'loader',
      type: 'list',
      message: 'Which loader would you like to use?',
      choices: Object.values(Loader)
    },
    {
      when: !options.gameVersion,
      name: 'gameVersion',
      type: 'input',
      default: await getLatestMinecraftVersion(),
      message: 'What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1)',
      validateText: 'Verifying the game version',
      validate: validateGameVersion
    },
    {
      when: !Object.hasOwn(options, 'allowVersionFallback'),
      name: 'allowVersionFallback',
      type: 'confirm',
      message: 'Should we try to download mods for previous Minecraft versions if they do not exist for your Minecraft Version?'
    },
    {
      when: !options.defaultAllowedReleaseTypes,
      name: 'defaultAllowedReleaseTypes',
      type: 'checkbox',
      choices: Object.values(ReleaseType),
      default: [ReleaseType.RELEASE, ReleaseType.BETA],
      message: 'Which types of releases would you like to consider to download?'
    },
    {
      when: !options.modsFolder,
      name: 'modsFolder',
      type: 'input',
      default: './mods',
      message: `where is your mods folder? (full or relative path from ${cwd}):`,
      validate: async (input: string) => {
        return validateModsFolder(input, cwd);
      }
    }
  ];
  const iq = await inquirer.prompt(prompts) as IQInternal;
  const answers = mergeOptions(options, iq) as ModsJson;

  await writeConfigFile(answers, options.config);

  return answers as ModsJson;
};

