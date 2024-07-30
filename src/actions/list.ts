import chalk from 'chalk';
import { ensureConfiguration, readLockFile } from '../lib/config.js';
import { Logger } from '../lib/Logger.js';
import { Mod } from '../lib/modlist.types.js';
import { DefaultOptions, telemetry } from '../mmm.js';

export type ListOptions = DefaultOptions

export const list = async (options: ListOptions, logger: Logger) => {
  performance.mark('list-start');
  const config = await ensureConfiguration(options.config, logger);
  const installed = await readLockFile(options, logger);

  logger.log((chalk.green('Configured mods')), true);

  const sortByName = (a: Mod, b: Mod) => {
    return a.name.localeCompare(b.name);
  };

  config.mods.sort(sortByName).forEach((mod) => {
    if (installed.find((i) => i.id === mod.id && i.type === mod.type)) {
      logger.log(`${chalk.green('\u2705')} ${mod.name?.trim()} ${chalk.gray('(')}${chalk.gray(mod.id)}${chalk.gray(')')} is installed`, true);
    } else {
      logger.log(`${chalk.red('\u274c')} ${mod.name?.trim()} ${chalk.gray('(')}${chalk.gray(mod.id)}${chalk.gray(')')} is not installed`, true);
    }
  });

  performance.mark('list-succeed');

  await telemetry.captureCommand({
    command: 'list',
    success: true,
    arguments: options,
    extra: {
      numberOfMods: config.mods.length,
      perf: performance.getEntries()
    },
    duration: performance.measure('list-duration', 'list-start', 'list-succeed').duration
  });
};
