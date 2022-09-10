#!/usr/bin/env node
import 'dotenv/config';
import { Command } from 'commander';
import { add } from './actions/add.js';
import { list } from './actions/list.js';
import { Platform } from './lib/modlist.types.js';
import { update } from './actions/update.js';
import { version } from './version.js';
import path from 'path';

export const APP_NAME = 'Minecraft Mod Updater';
export const APP_DESCRIPTION = 'Updates mods from Modrinth and Curseforge';
export const DEFAULT_CONFIG_LOCATION = path.resolve(process.cwd(), './modlist.json');

export interface DefaultOptions {
  config?: string;
  debug?: boolean;
}


const commands = [];

export const program = new Command();
program.name(APP_NAME).version(version).description(APP_DESCRIPTION);

commands.push(
  program.command('list')
    .action(async (options) => {
      await list(options);
    })
);

commands.push(
  program.command('update')
    .action(async (options) => {
      await update(options);
    })
);

commands.push(
  program.command('add')
    .argument('<type>', 'curseforge or modrinth')
    .argument('<id>', 'Curseforge or Modrinth Project Id')
    .action(async (type: Platform, id: string, options) => {
      await add(type, id, options);
    })
);

commands.forEach((command) => {
  command.option('-c, --config <MODLIST_JSON>', 'An alternative JSON file containing the configuration');
});
