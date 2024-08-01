import * as path from 'path';
import inquirer from 'inquirer';
import { ConfigFileAlreadyExistsException } from '../errors/ConfigFileAlreadyExistsException.js';
import { fileExists } from '../lib/config.js';
import { DefaultOptions } from '../mmm.js';

const addNewToFilename = (filename: string) => {
  const ext = path.extname(filename);
  const basename = path.basename(filename, ext);
  return `${basename}-new${ext}`;
};

const validateConfigName = async (input: string) => {
  if (await fileExists(input)) {
    return 'The config file already exists. Please choose a different name';
  }
  return true;
};

export const configFile = async (options: DefaultOptions, cwd: string) => {
  const configExists = await fileExists(path.resolve(cwd, options.config));

  if (configExists) {
    if (options.quiet === true) {
      throw new ConfigFileAlreadyExistsException(options.config);
    }

    const newConfigName = addNewToFilename(options.config);
    const { overwrite } = await inquirer.prompt({
      type: 'confirm',
      name: 'overwrite',
      message: `The config file ${options.config} already exists. Would you like to overwrite it?`,
      default: false
    });

    if (!overwrite) {
      const { newConfig } = await inquirer.prompt({
        type: 'input',
        name: 'newConfig',
        message: 'Please enter a new config file name',
        default: newConfigName,
        validate: validateConfigName
      });
      return path.resolve(cwd, newConfig);
    }
  }
  return path.resolve(cwd, options.config);
};
