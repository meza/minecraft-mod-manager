#!/usr/bin/env node
//import 'dotenv-defaults/config';
import { Command } from 'commander';
import path from 'path';
import { add } from './lib/add.js';
import { list } from './lib/list.js';
import { Platform } from './lib/modlist.types.js';
import { update } from './lib/update.js';


export const APP_NAME = 'Minecraft Mod Updater';
export const APP_DESCRIPTION = 'Updates mods from Modrinth, Curseforge and Datapacks from Vanilla Tweaks';
export const DEFAULT_CONFIG_LOCATION = path.resolve(process.cwd(), './modlist.json');

export interface DefaultOptions {
  config?: string;
}

const commands = [];

const program = new Command();
program.name(APP_NAME).version(process.env.npm_package_version).description(APP_DESCRIPTION);

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

program.parse();
