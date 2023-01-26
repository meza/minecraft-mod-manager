import { Mod, ModInstall } from './modlist.types.js';
import path from 'path';

export const getInstallation = (mod: Mod, installations: ModInstall[]) => {
  return installations.findIndex((i) => i.id === mod.id && i.type === mod.type);
};

export const hasInstallation = (mod: Mod, installations: ModInstall[]) => {
  return getInstallation(mod, installations) > -1;
};

export const fileIsManaged = (file: string, installations: ModInstall[]) => {
  const filename = path.basename(file);
  const result = installations.find((install) => {
    return install.fileName === filename;
  });

  return result !== undefined;
};

// generate unit tests for all of the above
