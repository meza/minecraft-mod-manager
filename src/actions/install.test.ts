import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';
import { generateScanResult } from '../../test/generateScanResult.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
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
import { expectCommandStartTelemetry } from '../../test/telemetryHelper.js';
import { CouldNotFindModException } from '../errors/CouldNotFindModException.js';
import { DownloadFailedException } from '../errors/DownloadFailedException.js';
import { NoRemoteFileFound } from '../errors/NoRemoteFileFound.js';
import { handleFetchErrors } from '../errors/handleFetchErrors.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, getModsFolder, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { fileIsManaged, getInstallation, hasInstallation } from '../lib/configurationHelper.js';
import { downloadFile } from '../lib/downloader.js';
import { getModFiles } from '../lib/fileHelper.js';
import { getHash } from '../lib/hash.js';
import { ModInstall, Platform } from '../lib/modlist.types.js';
import { scanFiles } from '../lib/scan.js';
import { updateMod } from '../lib/updater.js';
import { DefaultOptions } from '../mmm.js';
import { fetchModDetails } from '../repositories/index.js';
import { install } from './install.js';
import { FoundEntries, UnsureEntries, processScanResults } from './scan.js';

vi.mock('../lib/Logger.js');
vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('inquirer');
vi.mock('../lib/config.js');
vi.mock('../lib/updater.js');
vi.mock('../lib/hash.js');
vi.mock('../errors/handleFetchErrors.js');
vi.mock('../lib/fileHelper.js');
vi.mock('../lib/configurationHelper.js');
vi.mock('../lib/scan.js');
vi.mock('./scan.js');
vi.mock('../mmm.js');

interface LocalTestContext {
  options: DefaultOptions;
  logger: Logger;
}

describe('The install module', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.options = {
      config: 'config.json',
      debug: false,
      quiet: false
    };
    context.logger = new Logger({} as never);
    vi.mocked(context.logger.error).mockImplementation(() => {
      throw new Error('process.exit');
    });
    vi.mocked(handleFetchErrors).mockReturnValue();
    vi.mocked(getModFiles).mockResolvedValue([]);
  });

  it<LocalTestContext>('installs a new mod with no release type override', async ({ options, logger }) => {
    const { randomConfiguration, randomUninstalledMod } = setupOneUninstalledMod();
    delete randomUninstalledMod.allowedReleaseTypes;

    // Prepare the configuration file state
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);

    // Prepare the details the mod details fetcher should return
    const remoteDetails = generateRemoteModDetails().generated;

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails);

    // Prepare the download mock
    assumeSuccessfulDownload();
    // Run the install
    await install(options, logger);

    // Verify our expectations
    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomUninstalledMod, randomConfiguration);
    expect(logger.log).toHaveBeenCalledWith(
      `${randomUninstalledMod.name} doesn't exist, downloading from ${randomUninstalledMod.type}`
    );

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, options, logger);
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith(
      [
        {
          id: randomUninstalledMod.id,
          type: randomUninstalledMod.type,
          name: randomUninstalledMod.name,
          fileName: remoteDetails.fileName,
          releasedOn: remoteDetails.releaseDate,
          hash: remoteDetails.hash,
          downloadUrl: remoteDetails.downloadUrl
        }
      ],
      options,
      logger
    );

    expect(vi.mocked(downloadFile)).toHaveBeenCalledOnce();
    expect(vi.mocked(fetchModDetails)).toHaveBeenCalledOnce();

    verifyBasics();
  });

  it<LocalTestContext>('installs a new mod with a release type override', async ({ options, logger }) => {
    const { randomConfiguration, randomUninstalledMod } = setupOneUninstalledMod();

    // Prepare the configuration file state
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);

    // Prepare the details the mod details fetcher should return
    const remoteDetails = generateRemoteModDetails().generated;

    vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails);

    // Prepare the download mock
    assumeSuccessfulDownload();

    // Run the install
    await install(options, logger);

    // Verify our expectations
    expectModDetailsHaveBeenFetchedCorrectlyForMod(randomUninstalledMod, randomConfiguration);
  });

  it<LocalTestContext>('downloads a missing mod', async ({ options, logger }) => {
    const { randomConfiguration, randomInstalledMod, randomInstallation } = setupOneInstalledMod();

    // Prepare the configuration file state
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    // Prepare the download mock
    assumeSuccessfulDownload();

    // Prepare the file existence mock
    assumeModFileIsMissing(randomInstallation);

    vi.mocked(hasInstallation).mockReturnValue(true);
    vi.mocked(getInstallation).mockReturnValue(0);

    // Run the install
    await install(options, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith(
      `${randomInstalledMod.name} doesn't exist, downloading from ${randomInstalledMod.type}`
    );

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, options, logger);
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([randomInstallation], options, logger);

    expect(vi.mocked(downloadFile)).toHaveBeenCalledOnce();
    expect(vi.mocked(fetchModDetails)).not.toHaveBeenCalled();

    verifyBasics();
  });

  it<LocalTestContext>('downloads a mod with a different hash', async ({ options, logger }) => {
    const { randomConfiguration, randomInstalledMod, randomInstallation } = setupOneInstalledMod();

    // Prepare the configuration file state
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    vi.mocked(hasInstallation).mockReturnValueOnce(true);
    vi.mocked(getInstallation).mockReturnValueOnce(0);

    // Prepare the download mock
    assumeSuccessfulUpdate(randomInstallation);

    // Prepare the file existence mock
    assumeModFileExists(randomInstallation.fileName);

    vi.mocked(getHash).mockResolvedValueOnce('different-hash');
    // Run the install
    await install(options, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith(`${randomInstalledMod.name} has hash mismatch, downloading from source`);

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, options, logger);
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([randomInstallation], options, logger);

    expect(vi.mocked(updateMod)).toHaveBeenCalledOnce();
    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).not.toHaveBeenCalled();

    verifyBasics();
  });

  it<LocalTestContext>('Sets the appropriate debug messages for latest', async ({ options, logger }) => {
    const { randomInstalledMod, randomInstallation, randomConfiguration } = setupOneInstalledMod();

    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    options.debug = true;
    randomInstalledMod.version = undefined;
    await install(options, logger);

    expect(logger.debug).toHaveBeenCalledWith(
      `Checking ${randomInstalledMod.name}@latest for ${randomInstalledMod.type}`
    );
  });

  it<LocalTestContext>('calls the correct telemetry', async ({ options, logger }) => {
    const { randomInstallation, randomConfiguration } = setupOneInstalledMod();

    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    await install(options, logger);

    expectCommandStartTelemetry({
      command: 'install',
      success: true,
      duration: expect.any(Number),
      arguments: {
        options: options
      }
    });
  });

  it<LocalTestContext>('Sets the appropriate debug messages for specific version', async ({ options, logger }) => {
    const { randomInstalledMod, randomInstallation, randomConfiguration } = setupOneInstalledMod();

    randomInstalledMod.version = '1.1.0';

    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    options.debug = true;
    randomInstalledMod.version = undefined;
    await install(options, logger);

    expect(logger.debug).toHaveBeenCalledWith(
      `Checking ${randomInstalledMod.name}@latest for ${randomInstalledMod.type}`
    );
  });

  it<LocalTestContext>('Sets the appropriate debug messages for specific version', async ({ options, logger }) => {
    const { randomInstalledMod, randomInstallation, randomConfiguration } = setupOneInstalledMod();

    randomInstalledMod.version = '1.1.0';

    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);
    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    options.debug = true;
    await install(options, logger);

    expect(logger.debug).toHaveBeenCalledWith(
      `Checking ${randomInstalledMod.name}@1.1.0 for ${randomInstalledMod.type}`
    );
  });

  it<LocalTestContext>('handles the case when there is nothing to do', async ({ options, logger }) => {
    const { randomConfiguration, randomInstallation } = setupOneInstalledMod();

    // Prepare the configuration file state
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
    vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
    vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

    vi.mocked(hasInstallation).mockReturnValueOnce(true);
    vi.mocked(getInstallation).mockReturnValueOnce(0);

    vi.mocked(getHash).mockResolvedValueOnce(randomInstallation.hash);

    // Prepare the file existence mock
    assumeModFileExists(randomInstallation.fileName);

    // Run the install
    await install(options, logger);

    // Verify our expectations
    expect(logger.log).toHaveBeenCalledWith('✅ all mods are installed!');

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledWith(randomConfiguration, options, logger);
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledWith([randomInstallation], options, logger);

    expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fetchModDetails)).not.toHaveBeenCalled();

    verifyBasics();
  });

  describe('when there are unknown files', () => {
    it<LocalTestContext>('checks if all files are managed', async ({ options, logger }) => {
      // Prepare the configuration file state
      const emptyConfiguration = generateModsJson().generated;
      const emptyInstallations: ModInstall[] = [];
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(emptyConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(emptyConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce(emptyInstallations);

      const file1 = chance.word();
      const file2 = chance.word();

      vi.mocked(getModFiles).mockReset();
      vi.mocked(getModFiles).mockResolvedValueOnce([file1, file2]);
      vi.mocked(fileIsManaged).mockReturnValue(true); //exit quickly

      // Run the install
      await install(options, logger);

      expect(fileIsManaged).toHaveBeenCalledTimes(2);
      expect(fileIsManaged).toHaveBeenNthCalledWith(1, file1, emptyInstallations);
      expect(fileIsManaged).toHaveBeenNthCalledWith(2, file2, emptyInstallations);
    });

    it<LocalTestContext>('scans and processes only the non managed files', async ({ options, logger }) => {
      const emptyConfiguration = generateModsJson().generated;
      const emptyInstallations: ModInstall[] = [];
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(emptyConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(emptyConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce(emptyInstallations);

      const file1 = chance.word();
      const file2 = chance.word();
      const file3 = chance.word();

      vi.mocked(getModFiles).mockReset();
      vi.mocked(getModFiles).mockResolvedValueOnce([file1, file2, file3]);
      vi.mocked(fileIsManaged).mockReturnValueOnce(true); //file1 is managed
      vi.mocked(fileIsManaged).mockReturnValueOnce(false); //file2 is non-managed
      vi.mocked(fileIsManaged).mockReturnValueOnce(false); //file3 is non-managed

      const scanResults = [generateScanResult().generated];

      vi.mocked(scanFiles).mockResolvedValueOnce(scanResults); // not processing anything
      vi.mocked(processScanResults).mockReturnValue({ unsure: [] } as never);
      // Run the install
      await install(options, logger);

      expect(scanFiles).toHaveBeenCalledOnce();
      expect(scanFiles).toHaveBeenCalledWith([file2, file3], emptyInstallations, Platform.MODRINTH, emptyConfiguration);
      expect(processScanResults).toHaveBeenCalledWith(scanResults, emptyConfiguration, emptyInstallations, logger);
    });

    it<LocalTestContext>('exits as expected when there are unsure matches', async ({ options, logger }) => {
      const emptyConfiguration = generateModsJson().generated;
      const emptyInstallations: ModInstall[] = [];
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(emptyConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(emptyConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce(emptyInstallations);

      vi.mocked(hasInstallation).mockReturnValueOnce(false);

      vi.mocked(getModFiles).mockReset();
      vi.mocked(getModFiles).mockResolvedValueOnce([chance.word()]);
      vi.mocked(fileIsManaged).mockReturnValue(false); //don't care about the files

      vi.mocked(scanFiles).mockResolvedValueOnce([]); // not processing anything

      const processResult = {
        unsure: [
          chance.word() //doesn't matter
        ] as unknown as UnsureEntries[],
        unmanaged: [] as unknown as FoundEntries[]
      };

      vi.mocked(processScanResults).mockReturnValue(processResult);

      await expect(install(options, logger)).rejects.toThrow('process.exit');

      const errorMessage = vi.mocked(logger.error).mock.calls[0][0];

      expect(errorMessage).toMatchInlineSnapshot(`
        "
        Please fix the unresolved issues above manually or by running mmm scan, then try again."
      `);
    });
  });

  describe('when fetching a missing mod file fails', () => {
    it<LocalTestContext>('passes the correct error', async ({ options, logger }) => {
      const url = chance.url({ protocol: 'https' });
      const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

      // Prepare the configuration file state
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

      vi.mocked(hasInstallation).mockReturnValueOnce(true);
      vi.mocked(getInstallation).mockReturnValueOnce(0);

      // Prepare the file existence mock
      assumeModFileIsMissing(randomInstallation);
      const error = new DownloadFailedException(url);
      vi.mocked(downloadFile).mockRejectedValueOnce(error);

      await install(options, logger);
      expect(handleFetchErrors).toHaveBeenCalledWith(error, randomInstalledMod, logger);
    });
  });

  describe('when the download fails during an update', () => {
    it<LocalTestContext>('shows the correct message', async ({ options, logger }) => {
      const url = chance.url({ protocol: 'https' });
      const { randomConfiguration, randomInstallation, randomInstalledMod } = setupOneInstalledMod();

      // Prepare the configuration file state
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce([randomInstallation]);

      vi.mocked(hasInstallation).mockReturnValueOnce(true);
      vi.mocked(getInstallation).mockReturnValueOnce(0);

      // Prepare the file existence mock
      assumeModFileExists(randomInstallation.fileName);

      vi.mocked(getHash).mockResolvedValueOnce('different-hash');
      const error = new DownloadFailedException(url);
      vi.mocked(updateMod).mockRejectedValueOnce(error);

      await install(options, logger);

      expect(handleFetchErrors).toHaveBeenCalledWith(error, randomInstalledMod, logger);
    });
  });

  describe('when fetching a missing installation fails', () => {
    it<LocalTestContext>('reports the correct error', async ({ options, logger }) => {
      const url = chance.url({ protocol: 'https' });
      const { randomConfiguration, randomUninstalledMod } = setupOneUninstalledMod();

      // Prepare the configuration file state
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
      vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);

      // Prepare the details the mod details fetcher should return
      const remoteDetails = generateRemoteModDetails().generated;

      vi.mocked(fetchModDetails).mockResolvedValueOnce(remoteDetails);
      const error = new DownloadFailedException(url);
      vi.mocked(downloadFile).mockRejectedValueOnce(error);

      await install(options, logger);
      expect(handleFetchErrors).toHaveBeenCalledWith(error, randomUninstalledMod, logger);
    });
  });

  describe('when a mod cannot be found', () => {
    it<LocalTestContext>('reports the correct message', async ({ options, logger }) => {
      const aModName = 'a mod name';
      const { randomConfiguration, randomInstalledMod } = setupOneInstalledMod();

      randomConfiguration.mods[0].id = 'id';
      randomConfiguration.mods[0].name = aModName;
      randomConfiguration.mods[0].type = Platform.MODRINTH;

      // Prepare the configuration file state
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);

      const error = new CouldNotFindModException('id', Platform.MODRINTH);
      vi.mocked(fetchModDetails).mockRejectedValueOnce(error);

      await install(options, logger);

      expect(handleFetchErrors).toHaveBeenCalledWith(error, randomInstalledMod, logger);
    });
  });

  describe('when a remote file is not found', () => {
    it<LocalTestContext>('reports the correct message', async ({ options, logger }) => {
      const aModName = 'another mod name';
      const { randomConfiguration, randomInstalledMod } = setupOneInstalledMod();

      randomConfiguration.mods[0].id = 'id2';
      randomConfiguration.mods[0].name = aModName;
      randomConfiguration.mods[0].type = Platform.CURSEFORGE;

      // Prepare the configuration file state
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);
      const error = new NoRemoteFileFound(aModName, Platform.CURSEFORGE);
      vi.mocked(fetchModDetails).mockRejectedValueOnce(error);

      await install(options, logger);

      expect(handleFetchErrors).toHaveBeenCalledWith(error, randomInstalledMod, logger);
    });
  });

  describe('when an unexpected error occurs', () => {
    it<LocalTestContext>('throws it on', async ({ options, logger }) => {
      const error = chance.word();
      vi.mocked(handleFetchErrors).mockReset();
      vi.mocked(handleFetchErrors).mockImplementation(() => {
        throw new Error(error);
      });

      const { randomConfiguration } = setupOneInstalledMod();

      // Prepare the configuration file state
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(randomConfiguration);
      vi.mocked(getModsFolder).mockReturnValue(randomConfiguration.modsFolder);
      vi.mocked(readLockFile).mockResolvedValueOnce(emptyLockFile);

      vi.mocked(fetchModDetails).mockRejectedValueOnce(error);

      await expect(install(options, logger)).rejects.toThrow(error);
    });
  });
});
