import { confirm, input, select } from '@inquirer/prompts';
import chalk from 'chalk';
import { Logger } from '../lib/Logger.js';
import { Platform } from '../lib/modlist.types.js';
import { DefaultOptions } from '../mmm.js';

interface ModNofFoundInteractionResult {
  platform: Platform;
  id: string;
}

export const modNotFound = async (
  modName: string,
  platform: Platform,
  logger: Logger,
  options: DefaultOptions
): Promise<ModNofFoundInteractionResult> => {
  const errorText = `Mod "${chalk.whiteBright(modName)}" for ${chalk.whiteBright(platform)} does not exist`;
  if (options.quiet === true) {
    logger.error(errorText);
  }

  const answer: boolean = await confirm({
    message: `${errorText}. Would you like to modify your search?`,
    default: true
  });

  if (answer === false) {
    logger.error('Aborting', 0);
  }

  const newPlatform: Platform = await select({
    message: 'Which platform would you like to use?',
    choices: Object.values(Platform),
    default: platform
  });

  const newModName = await input({
    message: 'What is the project id of the mod you want to add?'
  });

  return {
    id: newModName,
    platform: newPlatform
  };
};
