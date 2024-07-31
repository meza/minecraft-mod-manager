import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  assumeModFileExists,
  assumeModFileIsMissing,
  expectModDetailsHaveBeenFetchedCorrectlyForMod,
  setupOneInstalledMod,
  verifyBasics
} from '../../test/setupHelpers.js';
import { expectCommandStartTelemetry } from '../../test/telemetryHelper.js';
import { update } from './update.js';
import { getHash } from '../lib/hash.js';
import { ensureConfiguration, getModsFolder, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { downloadFile } from '../lib/downloader.js';
import { fetchModDetails } from '../repositories/index.js';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';
import { install } from './install.js';
import { chance } from 'jest-chance';
import { updateMod } from '../lib/updater.js';
import * as path from 'path';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { Logger } from '../lib/Logger.js';
import { DefaultOptions } from '../mmm.js';
import { handleFetchErrors } from '../errors/handleFetchErrors.js';

vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('../lib/config.js');
vi.mock('../lib/updater.js');
vi.mock('../lib/hash.js');
vi.mock('./install.js');
vi.mock('../lib/Logger.js');
vi.mock('../errors/handleFetchErrors.js');
vi.mock('../mmm.js');

interface LocalTestContext {
  options: DefaultOptions;
  logger: Logger;
}

describe('The update action', () => {

  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.options = {
      config: 'config.json',
      debug: false,
      quiet: false
    };
    vi.mocked(install).mockResolvedValue();
    vi.mocked(context.logger.error).mockImplementation(() => {
      throw new Error('process.exit');
    });
    vi.mocked(handleFetchErrors).mockReturnValue();
  });

  it<LocalTestContext>('does nothing when there are no updates', async ({ options, logger }) => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();
    delete randomInstalledMod.allowedReleaseTypes;

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update(options, logger);

    // Verify our expectations
    expect(logger.log).not.toHaveBeenCalled();

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, options, logger);
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([randomInstallation], options, logger);

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();
    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomInstalledMod, randomConfiguration);

    verifyBasics();
  });

  it<LocalTestContext>('calls the correct telemetry', async ({ options, logger }) => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();
    delete randomInstalledMod.allowedReleaseTypes;

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update(options, logger);

    expectCommandStartTelemetry({
      command: 'update',
      success: true,
      duration: expect.any(Number),
      arguments: {
        options: options
      }
    });
  });

  it<LocalTestContext>('can use the release type override', async ({ options, logger }) => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update(options, logger);

    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomInstalledMod, randomConfiguration);

  });

  it<LocalTestContext>('can update based on hashes', async ({ options, logger }) => {
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
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(updateMod).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update(options, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith(`${remoteDetails.generated.name} has an update, downloading...`);
    expect(vi.mocked(updateMod)).toHaveBeenCalledWith(
      remoteDetails.generated,
      path.resolve(randomConfiguration.modsFolder, oldFilename),
      randomConfiguration.modsFolder
    );

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, options, logger);
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([newInstallation], options, logger);

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();

    verifyBasics();
  });

  it<LocalTestContext>('can update based on release date only', async ({ options, logger }) => {
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
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(updateMod).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update(options, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith(`${remoteDetails.generated.name} has an update, downloading...`);
    expect(vi.mocked(updateMod)).toHaveBeenCalledWith(
      remoteDetails.generated,
      path.resolve(randomConfiguration.modsFolder, oldFilename),
      randomConfiguration.modsFolder
    );

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, options, logger);
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([newInstallation], options, logger);

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();

    verifyBasics();
  });

  it<LocalTestContext>('logs the update checks for debug mode', async ({ options, logger }) => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    options.debug = chance.bool();
    await update(options, logger);

    // Verify our expectations
    expect(logger.debug).toHaveBeenCalledWith(`[update] Checking ${randomInstalledMod.name} for ${randomInstalledMod.type}`);

  });

  it<LocalTestContext>('prints the correct error when an installation is not found', async ({ options, logger }) => {
    const { randomConfiguration, randomInstallation } = setupOneInstalledMod();

    randomInstallation.name = 'random mod name';
    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn,
      name: randomInstallation.name
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([]);

    await update(options, logger);

    // Verify our expectations
    const message = vi.mocked(logger.error).mock.calls[0][0];
    const errorCode = vi.mocked(logger.error).mock.calls[0][1];

    expect(message).toMatchInlineSnapshot('"random mod name doesn\'t seem to be installed. Please delete the lock file and the mods folder and try again."');
    expect(errorCode).toEqual(1);

  });

  it<LocalTestContext>('prints the correct error when the original mod file does not exist', async ({ options, logger }) => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);

    assumeModFileIsMissing(randomInstallation);
    const expectedPath = path.resolve(randomConfiguration.modsFolder, randomInstallation.fileName);

    await update(options, logger);
    // Verify our expectations
    expect(logger.error).toHaveBeenCalledWith(`${randomInstalledMod.name} (${expectedPath}) doesn't exist. Please delete the lock file and the mods folder and try again.`, 1);

  });

  it<LocalTestContext>('handles unexpected errors', async ({ options, logger }) => {
    const randomErrorMessage = chance.sentence();
    const { randomConfiguration } = setupOneInstalledMod();

    vi.mocked(handleFetchErrors).mockReset();
    vi.mocked(handleFetchErrors).mockImplementation(() => {
      throw new Error(randomErrorMessage);
    });

    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    await expect(update(options, logger)).rejects.toThrow(randomErrorMessage);
  });

});
