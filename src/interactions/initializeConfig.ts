import path from 'node:path';
import { checkbox, input, select } from '@inquirer/prompts';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import { Logger } from '../lib/Logger.js';
import { fileExists, writeConfigFile } from '../lib/config.js';
import { verifyMinecraftVersion } from '../lib/minecraftVersionVerifier.js';
import { Loader, ModsJson, ReleaseType } from '../lib/modlist.types.js';
import { DefaultOptions } from '../mmm.js';
import { configFile } from './configFileOverwrite.js';
import { getLatestMinecraftVersion } from './getLatestMinecraftVersion.js';

export interface InitializeOptions extends DefaultOptions {
  loader?: Loader;
  gameVersion?: string;
  allowVersionFallback?: boolean;
  defaultAllowedReleaseTypes?: string;
  modsFolder?: string;
}

interface AnswersInternal {
  loader?: Loader;
  gameVersion?: string;
  allowVersionFallback?: boolean;
  defaultAllowedReleaseTypes?: string[];
  modsFolder?: string;
  config: string;
}

const validateGameVersion = async (input: string): Promise<boolean | string> => {
  if (await verifyMinecraftVersion(input)) {
    return true;
  }
  return 'The game version is invalid. Please enter a valid game version';
};

const mergeOptions = (options: InitializeOptions, iq: AnswersInternal) => {
  return {
    loader: iq.loader || options.loader,
    gameVersion: iq.gameVersion || options.gameVersion,
    defaultAllowedReleaseTypes:
      iq.defaultAllowedReleaseTypes || options.defaultAllowedReleaseTypes?.replace(/\s/g, '').split(','),
    modsFolder: iq.modsFolder || options.modsFolder,
    mods: []
  };
};

const validateModsFolder = async (input: string, cwd: string) => {
  let dir = path.resolve(cwd, input);
  if (path.isAbsolute(input)) {
    dir = input;
  }

  if (!(await fileExists(dir))) {
    return `The folder: ${dir} does not exist. Please enter a valid one and try again.`;
  }
  return true;
};

const validateInput = async (options: InitializeOptions, cwd: string) => {
  /**
   * @todo Handle cli option validation better for the init function
   *
   * Currently the negative case is just throwing errors. It would be nice
   * to properly communicate the errors and offer solutions.
   */
  if (options.gameVersion) {
    if (!(await verifyMinecraftVersion(options.gameVersion))) {
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

export const initializeConfig = async (options: InitializeOptions, cwd: string, logger: Logger): Promise<ModsJson> => {
  await validateInput(options, cwd);

  options.config = await configFile(options, cwd);

  const latestMinercraftDefault = await getLatestMinecraftVersion(options, logger);

  const answers: AnswersInternal = {
    config: options.config
  };

  if (!options.loader) {
    answers.loader = await select({
      message: 'Which loader would you like to use?',
      choices: Object.values(Loader)
    });
  }

  if (!options.gameVersion) {
    answers.gameVersion = await input({
      default: latestMinercraftDefault,
      message: 'What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1)',
      validate: validateGameVersion
    });
  }

  if (!options.defaultAllowedReleaseTypes) {
    answers.defaultAllowedReleaseTypes = await checkbox({
      message: 'Which types of releases would you like to consider to download?',
      choices: getReleaseTypeChoices([ReleaseType.RELEASE, ReleaseType.BETA])
    });
  }

  if (!options.modsFolder) {
    answers.modsFolder = await input({
      default: './mods',
      message: `where is your mods folder? (full or relative path from ${cwd}):`,
      validate: async (input: string) => {
        return validateModsFolder(input, cwd);
      }
    });
  }

  const answersOut = mergeOptions(options, answers) as ModsJson;

  await writeConfigFile(answersOut, options, logger);

  return answersOut as ModsJson;
};

interface Choice {
  name: string;
  value: string;
  checked?: boolean;
}

const getReleaseTypeChoices = (defaults: ReleaseType[]): Choice[] => {
  return Object.values(ReleaseType).map((value) => {
    const choice: Choice = {
      name: value,
      value: value
    };

    if (defaults.includes(value)) {
      choice.checked = true;
    }

    return choice;
  });
};
