import { beforeEach, describe, expect, it, vi } from 'vitest';
import { install } from './install.js';
import { fetchModDetails } from '../repositories/index.js';
import { readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';
import { downloadFile } from '../lib/downloader.js';
import { updateMod } from '../lib/updater.js';
import { getHash } from '../lib/hash.js';
import {
  assumeModFileExists,
  assumeModFileIsMissing,
  assumeSuccessfulDownload,
  assumeSuccessfulUpdate,
  emptyLockFile,
  expectModDetailsHaveBeenFetchedCorrectlyForMod,
  setupOneInstalledMod,
  setupOneUninstalledMod,
  verifyBasics
} from '../../test/setupHelpers.js';
import { Logger } from '../lib/Logger.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { ErrorTexts } from '../errors/ErrorTexts.js';
import { chance } from 'jest-chance';

vi.mock('../lib/Logger.js');
vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('inquirer');
vi.mock('../lib/config.js');
vi.mock('../lib/updater.js');
vi.mock('../lib/hash.js');

describe('The install module', () => {
  let logger: Logger;

  beforeEach(() => {
    vi.resetAllMocks();
    logger = new Logger({} as never);
  });

  it('installs a new mod with no release type override', async () => {

    const { randomConfiguration, randomUninstalledMod } = setupOneUninstalledMod();
    delete randomUninstalledMod.allowedReleaseTypes;

    // Prepare the configuration file state
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);

    // Prepare the details the mod details fetcher should return
    const remoteDetails = generateRemoteModDetails().generated;

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails);

    // Prepare the download mock
    assumeSuccessfulDownload();

    // Run the install
    await install({ config: 'config.json' }, logger);

    // Verify our expectations
    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomUninstalledMod, randomConfiguration);
    expect(logger.log).toHaveBeenCalledWith(`${randomUninstalledMod.name} doesn't exist, downloading from ${randomUninstalledMod.type}`);

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

  it('installs a new mod with a release type override', async () => {

    const { randomConfiguration, randomUninstalledMod } = setupOneUninstalledMod();

    // Prepare the configuration file state
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);

    // Prepare the details the mod details fetcher should return
    const remoteDetails = generateRemoteModDetails().generated;

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails);

    // Prepare the download mock
    assumeSuccessfulDownload();

    // Run the install
    await install({ config: 'config.json' }, logger);

    // Verify our expectations
    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomUninstalledMod, randomConfiguration);

  });

  it('downloads a missing mod', async () => {

    const { randomConfiguration, randomInstalledMod, randomInstallation } = setupOneInstalledMod();

    // Prepare the configuration file state
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([
      randomInstallation
    ]);

    // Prepare the download mock
    assumeSuccessfulDownload();

    // Prepare the file existence mock
    assumeModFileIsMissing(randomInstallation);

    // Run the install
    await install({ config: 'config.json' }, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith(
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

    // Prepare the download mock
    assumeSuccessfulUpdate(randomInstallation);

    // Prepare the file existence mock
    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce('different-hash');

    // Run the install
    await install({ config: 'config.json' }, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith(
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

  it('Sets the appropriate debug messages', async () => {
    const { randomInstalledMod, randomInstallation, randomConfiguration } = setupOneInstalledMod();

    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await install({ config: 'config.json', debug: true }, logger);

    expect(logger.debug).toHaveBeenCalledWith(`Checking ${randomInstalledMod.name} for ${randomInstalledMod.type}`);

  });

  it('handles the case when there is nothing to do', async () => {
    const { randomConfiguration, randomInstallation } = setupOneInstalledMod();

    // Prepare the configuration file state
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([
      randomInstallation
    ]);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    // Prepare the file existence mock
    assumeModFileExists(randomInstallation.fileName);

    // Run the install
    await install({ config: 'config.json' }, logger);

    // Verify our expectations
    expect(logger.log).not.toHaveBeenCalled();

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, 'config.json');
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([randomInstallation], 'config.json');

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).not.toHaveBeenCalled();

    verifyBasics();
  });

  it('shows the correct error message when the config file is missing', async () => {
    vi.mocked(readConfigFile).mockRejectedValueOnce(new ConfigFileNotFoundException('config.json'));
    await install({ config: 'config.json' }, logger);

    expect(vi.mocked(logger.error)).toHaveBeenCalledWith(ErrorTexts.configNotFound);

  });

  it('handles unexpected errors', async () => {
    const randomErrorMessage = chance.sentence();
    vi.mocked(readConfigFile).mockRejectedValueOnce(new Error(randomErrorMessage));
    await install({ config: 'config.json' }, logger);
    expect(logger.error).toHaveBeenCalledWith(randomErrorMessage, 2);
  });

});
