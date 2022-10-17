import { Logger } from '../lib/Logger.js';
import { DefaultOptions } from '../mmm.js';
import chalk from 'chalk';
import { ModsJson, Platform } from '../lib/modlist.types.js';
import inquirer from 'inquirer';

interface NoRemoteFileFoundInteractionResult {
  platform: Platform;
  id: string;
}

export const noRemoteFileFound = async (
  modName: string,
  platform: Platform,
  configuration: ModsJson,
  logger: Logger,
  options: DefaultOptions
): Promise<NoRemoteFileFoundInteractionResult> => {
  const errorText = `Could not find a file for ${modName} and the Minecraft version ${chalk.whiteBright(configuration.gameVersion)} `
    + `for ${chalk.whiteBright(configuration.loader)} loader`;

  if (options.quiet === true) {
    logger.error(errorText);
    return {} as NoRemoteFileFoundInteractionResult; // needs a return for testing purposes because the above line terminates the process in production
  }

  const newPlatform = platform === Platform.CURSEFORGE ? Platform.MODRINTH : Platform.CURSEFORGE;

  const answers = await inquirer.prompt([
    {
      name: 'confirm',
      type: 'confirm',
      message: `${errorText}. Would you like to try on ${newPlatform}?`,
      default: true
    }
  ]);

  if (answers.confirm === false) {
    logger.error('Aborting', 0);
    return {} as NoRemoteFileFoundInteractionResult; // needs a return for testing purposes because the above line terminates the process in production
  }

  const { newModName } = await inquirer.prompt([
    {
      type: 'input',
      name: 'newModName',
      message: `What is the project id of the mod you want to add on ${newPlatform}?`
    }
  ]);

  return {
    id: newModName,
    platform: newPlatform
  };

};
