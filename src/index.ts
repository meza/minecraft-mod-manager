#!/usr/bin/env node

import { Command } from 'commander';
import { version } from './version.js';

const APP_NAME = 'Minecraft Mod Updater';
const APP_DESCRIPTION = 'Updates mods from Modrinth, Curseforge and Datapacks from Vanilla Tweaks';

const program = new Command();
program.name(APP_NAME).version(version).description(APP_DESCRIPTION);

program.parse();
