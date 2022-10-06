import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  assumeModFileExists,
  assumeModFileIsMissing,
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

vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('../lib/config.js');
vi.mock('../lib/updater.js');
vi.mock('../lib/hash.js');
vi.mock('./install.js');

describe('The update action', () => {
  beforeEach(() => {
    vi.mocked(install).mockResolvedValue();
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('does nothing when there are no updates', async () => {
    const { randomConfiguration, randomInstallation } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    const consoleSpy = vi.spyOn(console, 'log');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });
    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json' });

    // Verify our expectations
    expect(consoleSpy).not.toHaveBeenCalled();

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, 'config.json');
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([randomInstallation], 'config.json');

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();

    verifyBasics();
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

    const consoleSpy = vi.spyOn(console, 'log');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });
    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json' });

    // Verify our expectations
    expect(consoleSpy).toHaveBeenCalledWith(`${remoteDetails.generated.name} has an update, downloading...`);
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

    const consoleSpy = vi.spyOn(console, 'log');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });
    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json' });

    // Verify our expectations
    expect(consoleSpy).toHaveBeenCalledWith(`${remoteDetails.generated.name} has an update, downloading...`);
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

  it('logs the update checks when in debug mode', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    const consoleSpy = vi.spyOn(console, 'debug');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });
    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await update({ config: 'config.json', debug: true });

    // Verify our expectations
    expect(consoleSpy).toHaveBeenCalledWith(`[update] Checking ${randomInstalledMod.name} for ${randomInstalledMod.type}`);

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

    const consoleSpy = vi.spyOn(console, 'error');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });

    await update({ config: 'config.json' });

    // Verify our expectations
    expect(consoleSpy).toHaveBeenCalledWith(`${randomInstalledMod.name} doesn't seem to be installed, please run mmm install first`);

  });

  it('prints the correct error when the original mod file doe not exist', async () => {
    const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

    const remoteDetails = generateRemoteModDetails({
      hash: randomInstallation.hash,
      releaseDate: randomInstallation.releasedOn,
      name: randomInstalledMod.name!
    });

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails.generated);
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    const consoleSpy = vi.spyOn(console, 'error');
    vi.mocked(consoleSpy).mockImplementation(() => {
    });
    assumeModFileIsMissing(randomInstallation);
    await update({ config: 'config.json' });
    const expectedPath = path.resolve(randomConfiguration.modsFolder, randomInstallation.fileName);

    // Verify our expectations
    expect(consoleSpy).toHaveBeenCalledWith(`${randomInstalledMod.name} (${expectedPath}) doesn't exist, please run mmm install`);

  });

});
