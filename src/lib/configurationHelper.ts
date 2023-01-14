import { Mod, ModInstall } from './modlist.types.js';

export const getInstallation = (mod: Mod, installations: ModInstall[]) => {
  return installations.findIndex((i) => i.id === mod.id && i.type === mod.type);
};

export const hasInstallation = (mod: Mod, installations: ModInstall[]) => {
  return getInstallation(mod, installations) > -1;
};

export const fileIsManaged = (file: string, installations: ModInstall[]) => {
  const result = installations.find((install) => {
    return install.fileName === file;
  });

  return result !== undefined;
};
