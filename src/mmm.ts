#!/usr/bin/env node
import { Command } from 'commander';
import 'dotenv/config';
import { add } from './actions/add.js';
import { changeGameVersion } from './actions/change.js';
import { install } from './actions/install.js';
import { list } from './actions/list.js';
import { prune } from './actions/prune.js';
import { removeAction } from './actions/remove.js';
import { scan } from './actions/scan.js';
import { testGameVersion } from './actions/testGameVersion.js';
import { update } from './actions/update.js';
import { helpUrl } from './env.js';
import { initializeConfig } from './interactions/initializeConfig.js';
import { Logger } from './lib/Logger.js';
import { Loader, Platform, ReleaseType } from './lib/modlist.types.js';
import { Telemetry } from './telemetry/telemetry.js';
import { version } from './version.js';

performance.mark('start');

export const APP_NAME = 'Minecraft Mod Manager';
export const APP_DESCRIPTION = 'Manages mods from Modrinth and Curseforge';
export const DEFAULT_CONFIG_LOCATION = './modlist.json';
const cwd = process.cwd();

export enum EXIT_CODE {
  SUCCESS = 0,
  GENERAL_ERROR = 1,
  SUPPLEMENTARY_ERROR = 2
}

export interface DefaultOptions {
  config: string;
  debug?: boolean;
  quiet?: boolean;
}

export const program = new Command();

export const logger: Logger = new Logger(program);
export const telemetry: Telemetry = new Telemetry();

export const stop = async (): Promise<never> => {
  await telemetry.flush();
  // eslint-disable-next-line no-process-exit
  process.exit(-1);
};

const commands = [];

program.name(APP_NAME).version(version).description(APP_DESCRIPTION);
program.addHelpText('after', '\nFor more information, visit: ' + helpUrl);

program.on('option:quiet', () => {
  logger.flagQuiet();
});

program.on('option:debug', () => {
  logger.flagDebug();
});

commands.push(
  program
    .command('list')
    .action(async (_options, cmd) => {
      await list(cmd.optsWithGlobals(), logger);
    })
    .aliases(['l', 'ls'])
);

commands.push(
  program
    .command('install')
    .action(async (_options, cmd) => {
      await install(cmd.optsWithGlobals(), logger);
    })
    .aliases(['i'])
);

commands.push(
  program
    .command('update')
    .action(async (_options, cmd) => {
      await update(cmd.optsWithGlobals(), logger);
    })
    .aliases(['u'])
);

commands.push(
  program
    .command('add')
    .argument('<type>', 'curseforge or modrinth')
    .argument('<id>', 'Curseforge or Modrinth Project Id')
    .option(
      '-v, --version <version>',
      'The version of the mod to add. If not specified, the latest version will be used'
    )
    .option(
      '-f, --allow-version-fallback',
      'Should we try to download the mod for previous Minecraft versions if they do not exists for your Minecraft Version?',
      false
    )
    .action(async (type: Platform, id: string, _options, cmd) => {
      await add(type, id, cmd.optsWithGlobals(), logger);
    })
    .aliases(['a'])
);

commands.push(
  program
    .command('init')
    .option('-l, --loader <loader>', `Which loader would you like to use? ${Object.values(Loader).join(', ')}`)
    .option(
      '-g, --game-version <gameVersion>',
      'What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1)'
    )
    .option(
      '-r, --default-allowed-release-types <defaultAllowedReleaseTypes>',
      `Which types of releases would you like to consider to download? ${Object.values(ReleaseType).join(', ')} - comma separated list`
    )
    .option('-m, --mods-folder <modsFolder>', `where is your mods folder? (full or relative path from ${cwd})`)
    .action(async (_options, cmd) => {
      await initializeConfig(cmd.optsWithGlobals(), cwd, logger);
    })
);

commands.push(
  program
    .command('test')
    .argument('[game_version]', 'The Minecraft version to test', 'latest')
    .aliases(['t'])
    .action(async (gameVersion: string, _options, cmd) => {
      await testGameVersion(gameVersion, cmd.optsWithGlobals(), logger);
    })
);

commands.push(
  program
    .command('change')
    .option(
      '-f, --force',
      'Force the change of the game version. Deletes all the mods and attempts to install with the given game version. Use at your own risk',
      false
    )
    .argument('[game_version]', 'The Minecraft version to change to', 'latest')
    .action(async (gameVersion: string, _options, cmd) => {
      await changeGameVersion(gameVersion, cmd.optsWithGlobals(), logger);
    })
);

commands.push(
  program
    .command('scan')
    .description('Scans the mod directory and attempts to find the mods on the supported mod platforms.')
    .option(
      '-p, --prefer <platform>',
      `Which platform do you prefer to use? ${Object.values(Platform).join(', ')}`,
      Platform.MODRINTH
    )
    .option('-a, --add', 'Add the mods to the modlist.json file', false)
    .action(async (_options, cmd) => {
      await scan(cmd.optsWithGlobals(), logger);
    })
);

commands.push(
  program
    .command('prune')
    .description('Prunes the mod directory from all the unmanaged files.')
    .option('-f, --force', 'Delete the files without asking', false)
    .action(async (_options, cmd) => {
      await prune(cmd.optsWithGlobals(), logger);
    })
);

commands.push(
  program
    .command('remove')
    .description('Removes one or more mods from both the config and the filesystem.')
    .option('-n, --dry-run', 'Print out the files/mods that would have been removed', false)
    .argument('<mods...>', 'A list of the mod(s) to remove. e.g: mmm remove mod1 mod2 "mod with space in its name"')
    .action(async (mods: string[], _options, cmd) => {
      await removeAction(mods, cmd.optsWithGlobals(), logger);
    })
);

program.option(
  '-c, --config <MODLIST_JSON>',
  'An alternative JSON file containing the configuration',
  DEFAULT_CONFIG_LOCATION
);
program.option('-q, --quiet', 'Suppress all output', false);
program.option('-d, --debug', 'Enable debug messages', false);
