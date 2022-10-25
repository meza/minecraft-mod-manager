#!/usr/bin/env node
import 'dotenv/config';
import { Command } from 'commander';
import { add } from './actions/add.js';
import { list } from './actions/list.js';
import { Platform, ReleaseType } from './lib/modlist.types.js';
import { install } from './actions/install.js';
import { version } from './version.js';
import { update } from './actions/update.js';
import { initializeConfig } from './interactions/initializeConfig.js';
import { Logger } from './lib/Logger.js';

export const APP_NAME = 'Minecraft Mod Manager';
export const APP_DESCRIPTION = 'Manages mods from Modrinth and Curseforge';
export const DEFAULT_CONFIG_LOCATION = './modlist.json';
const cwd = process.cwd();

export interface DefaultOptions {
  config: string;
  debug?: boolean;
  quiet?: boolean;
}

export const program = new Command();

export const logger: Logger = new Logger(program);

export const stop = (): never => {
  // eslint-disable-next-line no-process-exit
  process.exit(-1);
};

const commands = [];

program.name(APP_NAME).version(version).description(APP_DESCRIPTION);

program.on('option:quiet', () => {
  logger.flagQuiet();
});

program.on('option:debug', () => {
  logger.flagDebug();
});

commands.push(
  program.command('list')
    .action(async (_options, cmd) => {
      await list(cmd.optsWithGlobals(), logger);
    })
    .aliases(['l', 'ls'])
);

commands.push(
  program.command('install')
    .action(async (_options, cmd) => {
      await install(cmd.optsWithGlobals(), logger);
    })
    .aliases(['i'])
);

commands.push(
  program.command('update')
    .action(async (_options, cmd) => {
      await update(cmd.optsWithGlobals(), logger);
    })
    .aliases(['u'])
);

commands.push(
  program.command('add')
    .argument('<type>', 'curseforge or modrinth')
    .argument('<id>', 'Curseforge or Modrinth Project Id')
    .action(async (type: Platform, id: string, _options, cmd) => {
      await add(type, id, cmd.optsWithGlobals(), logger);
    })
    .aliases(['a'])
);

commands.push(
  program.command('init')
    .option('-l, --loader <loader>', `Which loader would you like to use? ${Object.values(Platform).join(', ')}`)
    .option('-g, --game-version <gameVersion>', 'What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1)')
    .option('-f, --allow-version-fallback', 'Should we try to download mods for previous Minecraft versions if they do not exists for your Minecraft Version?')
    .option('-r, --default-allowed-release-types <defaultAllowedReleaseTypes>',
      `Which types of releases would you like to consider to download? ${Object.values(ReleaseType).join(', ')} - comma separated list`)
    .option('-m, --mods-folder <modsFolder>', `where is your mods folder? (full or relative path from ${cwd})`)
    .action(async (_options, cmd) => {
      await initializeConfig(cmd.optsWithGlobals(), cwd);
    })
);

program.option('-c, --config <MODLIST_JSON>', 'An alternative JSON file containing the configuration', DEFAULT_CONFIG_LOCATION);
program.option('-q, --quiet', 'Suppress all output', false);
program.option('-d, --debug', 'Enable debug messages', false);

