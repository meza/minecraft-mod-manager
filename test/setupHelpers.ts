import { Mod, ModInstall, ModsJson } from '../src/lib/modlist.types.js';
import { fetchModDetails } from '../src/repositories/index.js';
import { generateModsJson } from './modlistGenerator.js';
import { generateModConfig } from './modConfigGenerator.js';
import { generateModInstall } from './modInstallGenerator.js';
import { expect, vi } from 'vitest';
import { fileExists, readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../src/lib/config.js';
import { downloadFile } from '../src/lib/downloader.js';
import { updateMod } from '../src/lib/updater.js';
import path from 'node:path';

export const emptyLockFile: ModInstall[] = [];

export const verifyBasics = () => {
  expect(vi.mocked(writeConfigFile)).toHaveBeenCalledOnce();
  expect(vi.mocked(writeLockFile)).toHaveBeenCalledOnce();
  expect(vi.mocked(readConfigFile)).toHaveBeenCalledOnce();
  expect(vi.mocked(readLockFile)).toHaveBeenCalledOnce();
};

export const expectModDetailsHaveBeenFetchedCorrectlyForMod = (
  mod: Mod, modsJson: ModsJson, call = 1) => {
  if (mod.allowedReleaseTypes) {
    expect(vi.mocked(fetchModDetails)).toHaveBeenNthCalledWith(
      call,
      mod.type,
      mod.id,
      mod.allowedReleaseTypes,
      modsJson.gameVersion,
      modsJson.loader,
      modsJson.allowVersionFallback
    );
  } else {
    expect(vi.mocked(fetchModDetails)).toHaveBeenNthCalledWith(
      call,
      mod.type,
      mod.id,
      modsJson.defaultAllowedReleaseTypes,
      modsJson.gameVersion,
      modsJson.loader,
      modsJson.allowVersionFallback
    );
  }
};

export const setupOneInstalledMod = () => {
  const randomConfiguration = generateModsJson().generated;
  const randomInstalledMod = generateModConfig().generated;
  const randomInstallation = generateModInstall({
    type: randomInstalledMod.type,
    id: randomInstalledMod.id
  }).generated;

  randomConfiguration.mods = [randomInstalledMod];

  return {
    randomConfiguration: randomConfiguration,
    randomInstalledMod: randomInstalledMod,
    randomInstallation: randomInstallation
  };
};

export const setupOneUninstalledMod = () => {
  const randomConfiguration = generateModsJson().generated;
  const randomUninstalledMod = generateModConfig().generated;

  randomConfiguration.mods = [randomUninstalledMod];

  return {
    randomConfiguration: randomConfiguration,
    randomUninstalledMod: randomUninstalledMod
  };
};

export const assumeSuccessfulDownload = () => {
  vi.mocked(downloadFile).mockResolvedValue();
};

export const assumeSuccessfulUpdate = (modToUpdate: ModInstall) => {
  vi.mocked(updateMod).mockResolvedValueOnce(modToUpdate);
};

export const assumeModFileIsMissing = (randomInstallation: ModInstall) => {
  vi.mocked(fileExists).mockImplementation(async (modPath: string) => {
    return path.basename(modPath) !== randomInstallation.fileName;
  });
};

export const assumeModFileExists = (filename: string) => {
  vi.mocked(fileExists).mockImplementation(async (modPath: string) => {
    return path.basename(modPath) === filename;
  });
};
