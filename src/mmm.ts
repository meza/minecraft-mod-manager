#!/usr/bin/env node
import 'dotenv/config';
import { Command } from 'commander';
import { add } from './actions/add.js';
import { list } from './actions/list.js';
import { Platform } from './lib/modlist.types.js';
import { install } from './actions/install.js';
import { version } from './version.js';
import path from 'path';

export const APP_NAME = 'Minecraft Mod Manager';
export const APP_DESCRIPTION = 'Manages mods from Modrinth and Curseforge';
export const DEFAULT_CONFIG_LOCATION = path.resolve(process.cwd(), './modlist.json');

export interface DefaultOptions {
  config?: string;
  debug?: boolean;
  quiet?: boolean;
}

const commands = [];

export const program = new Command();
program.name(APP_NAME).version(version).description(APP_DESCRIPTION);

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
  program.command('add')
    .argument('<type>', 'curseforge or modrinth')
    .argument('<id>', 'Curseforge or Modrinth Project Id')
    .action(async (type: Platform, id: string, options) => {
      await add(type, id, options);
    })
    .aliases(['a'])
);

commands.forEach((command) => {
  command.option('-c, --config <MODLIST_JSON>', 'An alternative JSON file containing the configuration');
  command.option('-q, --quiet', 'Suppress all output', false);
  command.option('-d, --debug', 'Enable debug messages', false);
});
