import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  assumeModFileExists,
  assumeModFileIsMissing,
  expectModDetailsHaveBeenFetchedCorrectlyForMod,
  setupOneInstalledMod,
  verifyBasics
} from '../../test/setupHelpers.js';
import { update } from './update.js';
import { getHash } from '../lib/hash.js';
import { readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { downloadFile } from '../lib/downloader.js';
import { fetchModDetails } from '../repositories/index.js';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';
import { install } from './install.js';
import { chance } from 'jest-chance';
import { updateMod } from '../lib/updater.js';
import * as path from 'path';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { Logger } from '../lib/Logger.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { ErrorTexts } from '../errors/ErrorTexts.js';

vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('../lib/config.js');
vi.mock('../lib/updater.js');
vi.mock('../lib/hash.js');
vi.mock('./install.js');
vi.mock('../lib/Logger.js');

describe('The update action', () => {
  let logger: Logger;
  beforeEach(() => {
    logger = new Logger({} as never);
    vi.mocked(install).mockResolvedValue();
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('does nothing when there are no updates', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();
    delete randomInstalledMod.allowedReleaseTypes;

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json' }, logger);

    // Verify our expectations
    expect(logger.log).not.toHaveBeenCalled();

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, 'config.json');
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([randomInstallation], 'config.json');

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();
    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomInstalledMod, randomConfiguration);

    verifyBasics();
  });

  it('can use the release type override', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json' }, logger);

    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomInstalledMod, randomConfiguration);

  });

  it('can update based on hashes', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();
    const oldFilename = randomInstallation.fileName;
    const newFilename = chance.word();
    const newHash = chance.hash();

    const newInstallation = generateModInstall({
      name: randomInstallation.name,
      releasedOn: randomInstallation.releasedOn,
      type: randomInstalledMod.type,
      fileName: newFilename,
      hash: newHash,
      id: randomInstallation.id
    }).generated;

    const remoteDetails = generateRemoteModDetails({
      hash: newHash,
      releaseDate: randomInstallation.releasedOn,
      fileName: newInstallation.fileName,
      downloadUrl: newInstallation.downloadUrl,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(updateMod).mockResolvedValueOnce(remoteDetails.generated);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json' }, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith(`${remoteDetails.generated.name} has an update, downloading...`);
    expect(vi.mocked(updateMod)).toHaveBeenCalledWith(
      remoteDetails.generated,
      path.resolve(randomConfiguration.modsFolder, oldFilename),
      randomConfiguration.modsFolder
    );

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, 'config.json');
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([newInstallation], 'config.json');

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();

    verifyBasics();
  });

  it('can update based on release date only', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();
    const oldFilename = randomInstallation.fileName;
    const newFilename = chance.word();
    const oldDate = '2022-01-01';
    const newDate = '2022-01-02';

    randomInstallation.releasedOn = oldDate;

    const newInstallation = generateModInstall({
      name: randomInstallation.name,
      releasedOn: newDate,
      type: randomInstalledMod.type,
      fileName: newFilename,
      hash: randomInstallation.hash,
      id: randomInstallation.id
    }).generated;

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: newDate,
      fileName: newInstallation.fileName,
      downloadUrl: newInstallation.downloadUrl,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(updateMod).mockResolvedValueOnce(remoteDetails.generated);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json' }, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith(`${remoteDetails.generated.name} has an update, downloading...`);
    expect(vi.mocked(updateMod)).toHaveBeenCalledWith(
      remoteDetails.generated,
      path.resolve(randomConfiguration.modsFolder, oldFilename),
      randomConfiguration.modsFolder
    );

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, 'config.json');
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([newInstallation], 'config.json');

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();

    verifyBasics();
  });

  it('logs the update checks for debug mode', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json', debug: chance.bool() }, logger);

    // Verify our expectations
    expect(logger.debug).toHaveBeenCalledWith(`[update] Checking ${randomInstalledMod.name} for ${randomInstalledMod.type}`);

  });

  it('prints the correct error when an installation is not found', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([]);

    await update({ config: 'config.json' }, logger);

    // Verify our expectations
    expect(logger.error).toHaveBeenCalledWith(`${randomInstalledMod.name} doesn't seem to be installed, please run mmm install first`);

  });

  it('prints the correct error when the original mod file does not exist', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    assumeModFileIsMissing(randomInstallation);
    const expectedPath = path.resolve(randomConfiguration.modsFolder, randomInstallation.fileName);

    await update({ config: 'config.json' }, logger);
    // Verify our expectations
    expect(logger.error).toHaveBeenCalledWith(`${randomInstalledMod.name} (${expectedPath}) doesn't exist, please run mmm install`);

  });

  it('shows the correct error message when the config file is missing', async () => {
    vi.mocked(readConfigFile).mockRejectedValueOnce(new ConfigFileNotFoundException('config.json'));
    await update({ config: 'config.json' }, logger);

    expect(vi.mocked(logger.error)).toHaveBeenCalledWith(ErrorTexts.configNotFound);

  });

  it('handles unexpected errors', async () => {
    const randomErrorMessage = chance.sentence();
    vi.mocked(readConfigFile).mockRejectedValueOnce(new Error(randomErrorMessage));
    await update({ config: 'config.json' }, logger);
    expect(logger.error).toHaveBeenCalledWith(randomErrorMessage, 2);
  });

});
