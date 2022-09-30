import { readConfigFile, readLockFile } from '../lib/config.js';
import chalk from 'chalk';
import { DefaultOptions } from '../mmm.js';

export interface ListOptions extends DefaultOptions {}

export const list = async (options: ListOptions) => {
  const config = await readConfigFile(options.config);
  const installed = await readLockFile(options.config);

  console.log((chalk.green('Configured mods')));

  config.mods.forEach((mod) => {
    if (installed.find((i) => i.id === mod.id && i.type === mod.type)) {
      console.log(chalk.green('\u2705'), mod.name?.trim(), 'is installed');
    } else {
      console.log(chalk.red('\u274c'), mod.name?.trim(), 'is not installed');
    }
  });
};
