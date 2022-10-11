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
import { hasUpdate } from './lib/mmmVersionCheck.js';

export const APP_NAME = 'Minecraft Mod Manager';
export const APP_DESCRIPTION = 'Manages mods from Modrinth and Curseforge';
export const DEFAULT_CONFIG_LOCATION = './modlist.json';

export interface DefaultOptions {
  config: string;
  debug?: boolean;
  quiet?: boolean;
}

const commands = [];
const cwd = process.cwd();
export const program = new Command();
program.name(APP_NAME).version(version).description(APP_DESCRIPTION);

await hasUpdate(version);

commands.push(
  program.command('list')
    .action(async (options) => {
      await list(options);
    })
    .aliases(['l', 'ls'])
);

commands.push(
  program.command('install')
    .action(async (options) => {
      await install(options);
    })
    .aliases(['i'])
);

commands.push(
  program.command('update')
    .action(async (options) => {
      await update(options);
    })
    .aliases(['u'])
);

commands.push(
  program.command('add')
    .argument('<type>', 'curseforge or modrinth')
    .argument('<id>', 'Curseforge or Modrinth Project Id')
    .action(async (type: Platform, id: string, options) => {
      await add(type, id, options);
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
    .action(async (options) => {
      await initializeConfig(options, cwd);
    })
);

commands.forEach((command) => {
  command.option('-c, --config <MODLIST_JSON>', 'An alternative JSON file containing the configuration', DEFAULT_CONFIG_LOCATION);
  command.option('-q, --quiet', 'Suppress all output', false);
  command.option('-d, --debug', 'Enable debug messages', false);
});
