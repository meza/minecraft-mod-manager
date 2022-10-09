import { DefaultOptions } from '../mmm.js';
import { Loader, ModsJson, Platform, ReleaseType } from '../lib/modlist.types.js';
import inquirer from 'inquirer';
import { fileExists, writeConfigFile } from '../lib/config.js';
import * as path from 'path';
import { getLatestMinecraftVersion, verifyMinecraftVersion } from '../lib/minecraftVersionVerifier.js';

interface InitializeOptions extends DefaultOptions {
  loader?: Loader,
  gameVersion?: string,
  allowVersionFallback?: boolean,
  defaultAllowedReleaseTypes?: string,
  modsFolder?: string
}

interface IQInternal extends ModsJson {
  config: string
}

const addNewToFilename = (filename: string) => {
  const ext = path.extname(filename);
  const basename = path.basename(filename, ext);
  return `${basename}-new${ext}`;
};

const validateConfigName = () => async (input: string) => {
  if (await fileExists(input)) {
    return 'The config file already exists. Please choose a different name';
  }
  return true;
};

const validateGameVersion = () => async (input: string) => {
  if (await verifyMinecraftVersion(input)) {
    return true;
  }
  return 'Invalid game version, please try again';
};

const mergeOptions = (options: InitializeOptions, iq: IQInternal) => {
  return {
    loader: iq.loader || options.loader,
    gameVersion: iq.gameVersion || options.gameVersion,
    allowVersionFallback: iq.allowVersionFallback || options.allowVersionFallback,
    defaultAllowedReleaseTypes: iq.defaultAllowedReleaseTypes || options.defaultAllowedReleaseTypes!.replace(/\s/g, '').split(','),
    modsFolder: iq.modsFolder || options.modsFolder,
    mods: []
  };
};

export const initializeConfig = async (options: InitializeOptions, cwd: string): Promise<ModsJson> => {
  const prompts = [
    {
      when: await fileExists(options.config),
      type: 'confirm',
      name: 'overwrite',
      default: false,
      message: `The config file: (${options.config}) already exists. Should we overwrite it?`
    },
    {
      when: async (answers: { overwrite?: boolean }) => {
        return await fileExists(options.config) && !answers.overwrite;
      },
      type: 'input',
      name: 'config',
      default: addNewToFilename(options.config),
      message: 'What should we name the new config file?:',
      verify: validateConfigName
    },
    {
      when: !options.loader,
      default: options.loader,
      name: 'loader',
      type: 'list',
      message: 'Which loader would you like to use?',
      choices: Object.values(Platform)
    },
    {
      when: !options.gameVersion,
      name: 'gameVersion',
      type: 'input',
      default: await getLatestMinecraftVersion(),
      message: 'What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1)',
      validate: validateGameVersion
    },
    {
      when: !options.allowVersionFallback,
      name: 'allowVersionFallback',
      type: 'confirm',
      message: 'Should we try to download mods for previous Minecraft versions if they do not exists for your Minecraft Version?'
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
      default: options.modsFolder || './mods',
      message: `where is your mods folder? (full or relative path from ${cwd}):`,
      validate: async (input: string) => {
        const dir = path.resolve(cwd, input);
        if (!await fileExists(dir)) {
          return `The folder: ${dir} does not exist. Please enter a valid one and try again.`;
        }
        return true;
      }
    }
  ];

  const iq = await inquirer.prompt(prompts) as IQInternal;

  const answers = mergeOptions(options, iq) as ModsJson;

  const configLocation = path.resolve(cwd, iq.config || options.config);

  await writeConfigFile(answers, configLocation);
  //TODO handle error

  return answers as ModsJson;
};

