import fs from 'node:fs/promises';
import path from 'node:path';
import chalk from 'chalk';
import inquirer from 'inquirer';
import { Logger } from '../lib/Logger.js';
import { fileExists } from '../lib/config.js';
import { DefaultOptions } from '../mmm.js';

const isFileWritable = async (filePath: string) => {
  let absolutePath = path.resolve(filePath);

  if (!(await fileExists(filePath))) {
    absolutePath = path.resolve(path.dirname(filePath));
  }

  const permissions = fs.constants.W_OK;

  try {
    await fs.access(absolutePath, permissions);
    return true;
  } catch {
    return false;
  }
};

export const fileToWrite = async (filePath: string, options: DefaultOptions, logger: Logger) => {
  logger.debug(`Checking if ${filePath} is writable`);
  const result = await isFileWritable(filePath);

  if (result === true) {
    return filePath;
  }

  if (options.quiet) {
    logger.error(`${filePath} is not writable. Aborting.`, 1);
  }

  const answers = await inquirer.prompt([
    {
      name: 'filePath',
      type: 'input',
      message: chalk.bgRed(chalk.whiteBright(`${filePath} is not writable, please choose another one`)),
      default: filePath,
      validationText: 'Checking if file is writable',
      validate: async (input: string) => {
        if (await isFileWritable(input)) {
          return true;
        }
        return chalk.bgRed(chalk.whiteBright(`${input} is not writable, please choose another one`));
      }
    }
  ]);

  return answers.filePath;
};
