import inquirer from 'inquirer';
import { Logger } from '../lib/Logger.js';
import { getLatestMinecraftVersion as getLatestMinecraftVersionLib } from '../lib/minecraftVersionVerifier.js';
import { DefaultOptions } from '../mmm.js';

const askForLatestVersion = async (): Promise<string> => {
  const answer = await inquirer.prompt([
    {
      name: 'gameVersion',
      type: 'input',
      message: 'The Minecraft APIs are down. What is the latest Minecraft version? (for example: 1.19.3, 1.20)'
    }
  ]);

  return answer.gameVersion;
};

export const getLatestMinecraftVersion = async (options: DefaultOptions, logger: Logger): Promise<string> => {
  try {
    return await getLatestMinecraftVersionLib();
  } catch (_e) {
    if (options.quiet) {
      logger.error('The Minecraft APIs are down and the latest minecraft version could not be determined.', 1);
    }

    return askForLatestVersion();
  }
};
