import { confirm, input } from '@inquirer/prompts';
import chalk from 'chalk';
import { Logger } from '../lib/Logger.js';
import { ModsJson, Platform } from '../lib/modlist.types.js';
import { DefaultOptions } from '../mmm.js';

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
  const errorText =
    `Could not find a file for ${modName} and the Minecraft version ${chalk.whiteBright(configuration.gameVersion)} ` +
    `for ${chalk.whiteBright(configuration.loader)} loader`;

  if (options.quiet === true) {
    logger.error(errorText);
  }

  const newPlatform = platform === Platform.CURSEFORGE ? Platform.MODRINTH : Platform.CURSEFORGE;

  const answer = await confirm({
    message: `${errorText}. Would you like to try on ${newPlatform}?`,
    default: true
  });

  if (answer === false) {
    logger.error('Aborting', 0);
  }

  const newModName = await input({
    message: `What is the project id of the mod you want to add on ${newPlatform}?`
  });

  return {
    id: newModName,
    platform: newPlatform
  };
};
