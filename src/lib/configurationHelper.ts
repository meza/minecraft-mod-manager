import { Mod, ModInstall, ModsJson } from './modlist.types.js';
import path from 'path';
import { minimatch } from 'minimatch';

export const findLocalMods = (lookup: string[], configuration: ModsJson) => {
  const matches: Set<Mod> = new Set<Mod>();

  lookup.forEach((modLookup) => {
    const pattern: RegExp = minimatch.makeRe(modLookup.toLowerCase()) as RegExp;
    const found = configuration.mods.filter((mod) => {
      if (mod.id.toLowerCase().match(pattern)) {
        return true;
      }

      if (mod.name.toLowerCase().match(pattern)) {
        return true;
      }

      return false;
    });

    found.forEach((mod) => {
      matches.add(mod);
    });
  });
  return matches;
};

export const fileIsManaged = (file: string, installations: ModInstall[]) => {
  const filename = path.basename(file);
  const result = installations.find((install) => {
    return install.fileName === filename;
  });

  return result !== undefined;
};

export const getModsDir = (configPath: string, modsFolder: string) => {
  const dir = path.resolve(path.dirname(configPath));
  return path.isAbsolute(modsFolder) ? modsFolder : path.resolve(dir, modsFolder);
};

export const getInstallation = (mod: Mod, installations: ModInstall[]) => {
  return installations.findIndex((i) => i.id === mod.id && i.type === mod.type);
};

export const hasInstallation = (mod: Mod, installations: ModInstall[]) => {
  return getInstallation(mod, installations) > -1;
};

// generate unit tests for all of the above
