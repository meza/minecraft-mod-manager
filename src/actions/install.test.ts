import { describe, it, expect, vi } from 'vitest';
import { afterEach } from 'vitest';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { Mod, ModInstall, ModsJson } from '../lib/modlist.types.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { install } from './install.js';
import { fetchModDetails } from '../repositories/index.js';
import { fileExists, readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { generateRemoteModDetails } from '../../test/modDetailsGenerator.js';
import { downloadFile } from '../lib/downloader.js';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import path from 'node:path';
import { updateMod } from '../lib/updater.js';
import { getHash } from '../lib/hash.js';

vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('inquirer');
vi.mock('../lib/config.js');
vi.mock('../lib/updater.js');
vi.mock('../lib/hash.js');

const emptyLockFile: ModInstall[] = [];

const expectModDetailsHaveBeenFetchedCorrectlyForMod = (
  mod: Mod, modsJson: ModsJson, call: number = 1) => {
  expect(vi.mocked(fetchModDetails)).toHaveBeenNthCalledWith(
    call,
    mod.type,
    mod.id,
    modsJson.defaultAllowedReleaseTypes,
    modsJson.gameVersion,
    modsJson.loader,
    modsJson.allowVersionFallback
  );
};

const setupOneInstalledMod = () => {
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

const setupOneUninstalledMod = () => {
  const randomConfiguration = generateModsJson().generated;
  const randomUninstalledMod = generateModConfig().generated;

  randomConfiguration.mods = [randomUninstalledMod];

  return {
    randomConfiguration: randomConfiguration,
    randomUninstalledMod: randomUninstalledMod
  };
};

const verifyBasics = () => {
  expect(vi.mocked(writeConfigFile)).toHaveBeenCalledOnce();
  expect(vi.mocked(writeLockFile)).toHaveBeenCalledOnce();
  expect(vi.mocked(readConfigFile)).toHaveBeenCalledOnce();
  expect(vi.mocked(readLockFile)).toHaveBeenCalledOnce();
};

const assumeSuccessfulDownload = () => {
  vi.mocked(downloadFile).mockResolvedValue();
};

const assumeSuccessfulUpdate = (modToUpdate: ModInstall) => {
  vi.mocked(updateMod).mockResolvedValueOnce(modToUpdate);
};

const assumeModFileIsMissing = (randomInstallation: ModInstall) => {
  vi.mocked(fileExists).mockImplementation(async (modPath: string) => {
    return path.basename(modPath) !== randomInstallation.fileName;
  });
};

const assumeModFileExists = (randomInstallation: ModInstall) => {
  vi.mocked(fileExists).mockImplementation(async (modPath: string) => {
    return path.basename(modPath) === randomInstallation.fileName;
  });
};

describe('The install module', () => {

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('installs a new mod', async () => {

    const { randomConfiguration, randomUninstalledMod } = setupOneUninstalledMod();

    // Prepare the configuration file state
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);

    // Prepare the details the mod details fetcher should return
    const remoteDetails = generateRemoteModDetails().generated;
    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails);

    // Prepare the console log mock
    const consoleSpy = vi.spyOn(console, 'log');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });

    // Prepare the download mock
    assumeSuccessfulDownload();

    // Run the install
    await install({ config: 'config.json' });

    // Verify our expectations
    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomUninstalledMod, randomConfiguration);
    expect(consoleSpy).toHaveBeenCalledWith(`${randomUninstalledMod.name} doesn't exist, downloading from ${randomUninstalledMod.type}`);

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, 'config.json');
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([
      {
        id: randomUninstalledMod.id,
        type: randomUninstalledMod.type,
        name: randomUninstalledMod.name,
        fileName: remoteDetails.fileName,
        releasedOn: remoteDetails.releaseDate,
        hash: remoteDetails.hash,
        downloadUrl: remoteDetails.downloadUrl
      }
    ], 'config.json');

    expect(vi.mocked(downloadFile)).toHaveBeenCalledOnce();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();

    verifyBasics();

  });

  it('downloads a missing mod', async () => {

    const { randomConfiguration, randomInstalledMod, randomInstallation } = setupOneInstalledMod();

    // Prepare the configuration file state
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([
      randomInstallation
    ]);

    // Prepare the console log mock
    const consoleSpy = vi.spyOn(console, 'log');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });

    // Prepare the download mock
    assumeSuccessfulDownload();

    // Prepare the file existence mock
    assumeModFileIsMissing(randomInstallation);

    // Run the install
    await install({ config: 'config.json' });

    // Verify our expectations
    expect(consoleSpy).toHaveBeenCalledWith(
      `${randomInstalledMod.name} doesn't exist, downloading from ${randomInstalledMod.type}`
    );

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, 'config.json');
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([
      randomInstallation
    ], 'config.json');

    expect(vi.mocked(downloadFile)).toHaveBeenCalledOnce();
    expect(vi.mocked(fetchModDetails)).not.toHaveBeenCalled();

    verifyBasics();

  });

  it('downloads a mod with a different hash', async () => {
    const { randomConfiguration, randomInstalledMod, randomInstallation } = setupOneInstalledMod();

    // Prepare the configuration file state
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([
      randomInstallation
    ]);

    // Prepare the console log mock
    const consoleSpy = vi.spyOn(console, 'log');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });

    // Prepare the download mock
    assumeSuccessfulUpdate(randomInstallation);

    // Prepare the file existence mock
    assumeModFileExists(randomInstallation);

    vi.mocked(getHash).mockResolvedValueOnce('different-hash');

    // Run the install
    await install({ config: 'config.json' });

    // Verify our expectations
    expect(consoleSpy).toHaveBeenCalledWith(
      `${randomInstalledMod.name} has hash mismatch, downloading from source`
    );

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, 'config.json');
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([
      randomInstallation
    ], 'config.json');

    expect(vi.mocked(updateMod)).toHaveBeenCalledOnce();
    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).not.toHaveBeenCalled();

    verifyBasics();
  });

  it('Shows the debug messages when it is enabled', async () => {
    const { randomInstalledMod, randomInstallation, randomConfiguration } = setupOneInstalledMod();

    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    const consoleSpy = vi.spyOn(console, 'debug');
    vi.mocked(consoleSpy).mockImplementation(() => {});

    await install({ config: 'config.json', debug: true });

    expect(consoleSpy).toHaveBeenCalledWith(`Checking ${randomInstalledMod.name} for ${randomInstalledMod.type}`);

  });

});
