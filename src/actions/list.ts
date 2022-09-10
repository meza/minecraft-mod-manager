import { readConfigFile } from '../lib/config.js';
import chalk from 'chalk';
import { DefaultOptions } from '../mmu.js';

export interface ListOptions extends DefaultOptions {}

export const list = async (options: ListOptions) => {
  const config = await readConfigFile(options.config);

  console.log((chalk.green('Configured mods')));

  config.mods.forEach((mod) => {
    console.log(` ${mod.name}`);
    if (mod.installed) {
      console.log('   ', chalk.green('\u2705'), 'Installed');
    } else {
      console.log('   ', chalk.green('\u274c'), 'Not installed');
    }

  });
};
