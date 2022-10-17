import { Logger } from '../lib/Logger.js';
import { DefaultOptions } from '../mmm.js';
import chalk from 'chalk';
import { Platform } from '../lib/modlist.types.js';
import inquirer from 'inquirer';

interface ModNofFoundInteractionResult {
  platform: Platform;
  id: string;
}

export const modNotFound = async (modName: string, platform: Platform, logger: Logger, options: DefaultOptions): Promise<ModNofFoundInteractionResult> => {
  const errorText = `Mod "${chalk.whiteBright(modName)}" for ${chalk.whiteBright(platform)} does not exist`;
  if (options.quiet === true) {
    logger.error(errorText);
    return {} as ModNofFoundInteractionResult; // needs a return for testing purposes because the above line terminates the process in production
  }

  const answers = await inquirer.prompt([
    {
      name: 'confirm',
      type: 'confirm',
      message: `${errorText}. Would you like to modify your search?`,
      default: true
    }
  ]);

  if (answers.confirm === false) {
    logger.error('Aborting', 0);
    return {} as ModNofFoundInteractionResult; // needs a return for testing purposes because the above line terminates the process in production
  }

  const { newModName, newPlatform } = await inquirer.prompt([
    {
      name: 'newPlatform',
      type: 'list',
      message: 'Which platform would you like to use?',
      choices: Object.values(Platform),
      default: platform
    },
    {
      type: 'input',
      name: 'newModName',
      message: 'What is the project id of the mod you want to add?'
    }
  ]);

  return {
    id: newModName,
    platform: newPlatform
  };

};
