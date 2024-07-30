import inquirer from 'inquirer';
import { chance } from 'jest-chance';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { generateRandomPlatform } from '../../test/generateRandomPlatform.js';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { expectCommandStartTelemetry, expectCommandStartTelemetryNth } from '../../test/telemetryHelper.js';
import { GeneratorResult } from '../../test/test.types.js';
import { CouldNotFindModException } from '../errors/CouldNotFindModException.js';
import { DownloadFailedException } from '../errors/DownloadFailedException.js';
import { NoRemoteFileFound } from '../errors/NoRemoteFileFound.js';
import { UnknownPlatformException } from '../errors/UnknownPlatformException.js';
import { modNotFound } from '../interactions/modNotFound.js';
import { noRemoteFileFound } from '../interactions/noRemoteFileFound.js';
import { ensureConfiguration, getModsFolder, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { downloadFile } from '../lib/downloader.js';
import { Logger } from '../lib/Logger.js';
import { ModInstall, ModsJson, Platform, RemoteModDetails } from '../lib/modlist.types.js';
import { fetchModDetails } from '../repositories/index.js';
import { add } from './add.js';

vi.mock('../lib/Logger.js');
vi.mock('../mmm.js');
vi.mock('../lib/config.js');
vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('inquirer');
vi.mock('../interactions/shouldCreateConfig.js');
vi.mock('../interactions/modNotFound.ts');
vi.mock('../interactions/noRemoteFileFound.js');

interface LocalTestContext {
  randomConfiguration: GeneratorResult<ModsJson>;
  randomModDetails: GeneratorResult<RemoteModDetails>;
}

const assumeDownloadIsSuccessful = () => {
  vi.mocked(downloadFile).mockResolvedValueOnce();
};

const assumeWrongPlatform = (override?: string) => {
  const randomPlatform = override || chance.word();
  vi.mocked(fetchModDetails).mockReset();
  vi.mocked(fetchModDetails).mockRejectedValueOnce(new UnknownPlatformException(randomPlatform));
  return randomPlatform;
};

const assumeModNotFound = (override?: string) => {
  const randomPlatform = getRandomPlatform();
  const randomMod = override || chance.word();

  vi.mocked(fetchModDetails).mockReset();
  vi.mocked(fetchModDetails).mockRejectedValueOnce(new CouldNotFindModException(randomPlatform, randomMod));
  return { randomPlatform: randomPlatform, randomMod: randomMod };
};

const getRandomPlatform = () => {
  return generateRandomPlatform();
};

describe('The add module', async () => {
  let logger: Logger;
  beforeEach<LocalTestContext>((context) => {
    logger = new Logger({} as never);
    context.randomConfiguration = generateModsJson();

    // the main configuration to work with
    vi.mocked(ensureConfiguration).mockResolvedValue(context.randomConfiguration.generated);
    vi.mocked(getModsFolder).mockReturnValue(context.randomConfiguration.generated.modsFolder);
    vi.mocked(readLockFile).mockResolvedValue([]);

    // the mod details returned from the repository
    context.randomModDetails = generateRemoteModDetails();
    vi.mocked(fetchModDetails).mockResolvedValueOnce(context.randomModDetails.generated);
    vi.mocked(logger.error).mockImplementation(() => {
      throw new Error('process.exit');
    });
  });

  afterEach(() => {
    vi.resetAllMocks();
    vi.clearAllMocks();
  });

  it<LocalTestContext>('should add a mod to the configuration', async (
    { randomConfiguration, randomModDetails }
  ) => {

    const randomPlatform = chance.pickone(['fabric', 'forge']);
    const randomModId = chance.word();

    assumeDownloadIsSuccessful();
    const options = { config: 'config.json' };
    await add(randomPlatform, randomModId, options, logger);

    expectCommandStartTelemetry({
      command: 'add',
      success: true,
      duration: expect.any(Number),
      arguments: {
        options: options,
        platform: randomPlatform,
        id: randomModId
      }
    });

    expect(
      vi.mocked(ensureConfiguration),
      'did not read the configuration file'
    ).toHaveBeenCalledTimes(1);

    expect(
      vi.mocked(fetchModDetails),
      'fetching the mod details during adding didn\'t happen'
    ).toHaveBeenCalledTimes(1);

    const expectedConfiguration = {
      ...randomConfiguration.expected,
      mods: [
        {
          type: randomPlatform,
          id: randomModId,
          name: randomModDetails.expected.name
        }
      ]
    };

    const expectedLockFile: ModInstall[] = [
      {
        type: randomPlatform,
        id: randomModId,
        name: randomModDetails.expected.name,
        fileName: randomModDetails.expected.fileName,
        releasedOn: randomModDetails.expected.releaseDate,
        hash: randomModDetails.expected.hash,
        downloadUrl: randomModDetails.expected.downloadUrl
      }
    ];

    expect(
      vi.mocked(writeConfigFile),
      'Writing the configuration file after adding a mod has failed'
    ).toHaveBeenCalledWith(expectedConfiguration, options, logger);

    expect(
      vi.mocked(writeLockFile),
      'Writing the lock file after adding a mod has failed'
    ).toHaveBeenCalledWith(expectedLockFile, options, logger);

    expect(
      logger.log,
      'The log message was not logged'
    ).toHaveBeenCalledWith(`✅ Added ${randomModDetails.expected.name} (${randomModId}) for ${randomPlatform}`);

  });

  it<LocalTestContext>('should skip the download if the mod already exists', async (context) => {
    const randomPlatform = getRandomPlatform();
    const randomModId = chance.word();

    const randomModDetails = generateModConfig({
      type: randomPlatform,
      id: randomModId
    });

    context.randomConfiguration.generated.mods = [randomModDetails.generated];

    await add(randomPlatform, randomModId, { config: 'config.json' }, logger);

    expect(
      vi.mocked(fetchModDetails),
      'Fetched the mod details even though the mod already exists'
    ).toHaveBeenCalledTimes(0);

    expect(
      vi.mocked(downloadFile),
      'The download was called even though the mod already exists'
    ).toHaveBeenCalledTimes(0);

    expectCommandStartTelemetry({
      command: 'add',
      success: true,
      duration: expect.any(Number),
      arguments: {
        options: { config: 'config.json' },
        platform: randomPlatform,
        id: randomModId
      },
      extra: {
        flag: 'already-exists'
      }
    });
  });

  it<LocalTestContext>('should send the correct debug message', async (context) => {

    const randomPlatform = Platform.MODRINTH;
    const randomModId = 'another-mod-id';

    const randomModDetails = generateModConfig({
      type: randomPlatform,
      id: randomModId
    });

    context.randomConfiguration.generated.mods = [randomModDetails.generated];

    await add(randomPlatform, randomModId, { config: 'config.json', debug: chance.bool() }, logger);

    expect(
      logger.debug,
      'The debug message was not logged'
    ).toHaveBeenCalledWith('Mod another-mod-id for modrinth already exists in the configuration');
  });

  it<LocalTestContext>('should report when a file cannot be found for the version and exit', async ({ randomConfiguration }) => {
    const randomPlatform = Platform.CURSEFORGE;
    const randomModId = chance.word();
    const randomAllowVersion = chance.bool();
    let randomVersion: string | undefined;

    if (chance.bool()) {
      randomVersion = chance.word();
    }

    vi.mocked(fetchModDetails).mockReset();
    vi.mocked(fetchModDetails).mockRejectedValueOnce(new NoRemoteFileFound(randomModId, randomPlatform));
    vi.mocked(fetchModDetails).mockRejectedValueOnce(new Error('test-error'));
    vi.mocked(noRemoteFileFound).mockResolvedValueOnce({
      id: 'another-mod-id',
      platform: Platform.MODRINTH
    });

    await expect(add(randomPlatform, randomModId, { config: 'config.json', allowVersionFallback: randomAllowVersion, version: randomVersion }, logger)).rejects.toThrow(new Error('process.exit'));

    expect(fetchModDetails).toHaveBeenNthCalledWith(2, Platform.MODRINTH, 'another-mod-id',
      randomConfiguration.expected.defaultAllowedReleaseTypes,
      randomConfiguration.expected.gameVersion,
      randomConfiguration.expected.loader,
      randomAllowVersion,
      randomVersion);
    expect(noRemoteFileFound).toHaveBeenCalledWith(randomModId, randomPlatform, randomConfiguration.expected, logger, {
      config: 'config.json',
      allowVersionFallback: randomAllowVersion,
      version: randomVersion
    });
    expect(logger.error).toHaveBeenCalledWith('test-error', 2);
  });

  it<LocalTestContext>('should work when the retry succeeded', async ({ randomConfiguration }) => {
    const randomPlatform = Platform.CURSEFORGE;
    const randomModId = chance.word();
    const secondMod = generateRemoteModDetails();
    vi.mocked(fetchModDetails).mockReset();
    vi.mocked(fetchModDetails).mockRejectedValueOnce(new NoRemoteFileFound(randomModId, randomPlatform));
    vi.mocked(fetchModDetails).mockResolvedValueOnce(secondMod.generated);
    assumeDownloadIsSuccessful();
    vi.mocked(noRemoteFileFound).mockResolvedValueOnce({
      id: 'another-mod-id',
      platform: Platform.MODRINTH
    });

    await add(randomPlatform, randomModId, { config: 'config.json' }, logger);

    expect(noRemoteFileFound).toHaveBeenCalledOnce();
    expect(logger.error).not.toHaveBeenCalled();

    expectCommandStartTelemetryNth(1, {
      command: 'add',
      success: false,
      duration: expect.any(Number),
      arguments: {
        options: { config: 'config.json' },
        platform: randomPlatform,
        id: randomModId
      },
      config: randomConfiguration.generated,
      error: `No compatible files were found for the given mod: ${randomPlatform}: ${randomModId}`
    });

    expectCommandStartTelemetryNth(2, {
      command: 'add',
      success: true,
      duration: expect.any(Number),
      arguments: {
        options: { config: 'config.json' },
        platform: Platform.MODRINTH,
        id: 'another-mod-id'
      }
    });
  });

  it('should report unexpected errors', async () => {

    const randomErrorMessage = chance.sentence();
    const randomPlatform = getRandomPlatform();
    const randomMod = chance.word();
    const error = new Error(randomErrorMessage);

    vi.mocked(fetchModDetails).mockReset();
    vi.mocked(fetchModDetails).mockRejectedValueOnce(error);

    await expect(add(randomPlatform, randomMod, { config: 'config.json' }, logger)).rejects.toThrow(new Error('process.exit'));

    expect(logger.error).toHaveBeenCalledWith(randomErrorMessage, 2);

  });

  describe('when an incorrect platform is chosen in interactive mode', async () => {
    describe('and the user chooses to cancel', async () => {
      it('it should exit after the prompt', async () => {
        const wrongPlatformText = assumeWrongPlatform();
        const randomModId = chance.word();

        vi.mocked(inquirer.prompt).mockResolvedValueOnce({ platform: 'cancel' });

        await expect(add(wrongPlatformText, randomModId, { config: 'config.json' }, logger)).resolves.toBeUndefined();

        // @ts-ignore anyone with a fix for this?
        const inquirerOptions = vi.mocked(inquirer.prompt).mock.calls[0][0][0];

        expect(inquirerOptions.choices.sort()).toEqual(['cancel', ...Object.values(Platform)].sort());
        expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledTimes(1);
        // These mean that the add hasn't been recursively called
        expect(vi.mocked(ensureConfiguration)).toHaveBeenCalledTimes(1);
        expect(vi.mocked(fetchModDetails)).toHaveBeenCalledTimes(1);
        expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
      });
    });

    describe('and the user chooses an alternative option', async () => {
      it<LocalTestContext>('it should call add again with the new platform', async (context) => {
        const randomModId = chance.word();

        // 1st invocation fails
        const wrongPlatformText = assumeWrongPlatform();

        // we select a correct platform through the prompt
        const randomPlatform = getRandomPlatform();
        vi.mocked(inquirer.prompt).mockResolvedValueOnce({ platform: randomPlatform });

        // upon 2nd invocation of the fetch, return a correct response
        vi.mocked(fetchModDetails).mockResolvedValueOnce(context.randomModDetails.generated);

        // we assume that the download is successful
        assumeDownloadIsSuccessful();

        await add(wrongPlatformText, randomModId, { config: 'config.json' }, logger);

        expect(vi.mocked(downloadFile)).toHaveBeenCalledWith(
          context.randomModDetails.generated.downloadUrl,
          expect.any(String)
        );

        // make sure we save with the correct platform
        const actualConfiguration = vi.mocked(writeConfigFile).mock.calls[0][0];
        expect(actualConfiguration.mods[0].type).toEqual(randomPlatform);

        // validate our assumptions about how many times things have been called
        expect(vi.mocked(downloadFile)).toHaveBeenCalledOnce();
        expect(vi.mocked(fetchModDetails)).toHaveBeenCalledTimes(2);
        expect(vi.mocked(ensureConfiguration)).toHaveBeenCalledTimes(2);
        expect(vi.mocked(writeConfigFile)).toHaveBeenCalledTimes(1);

      });
    });
  });

  describe('when an incorrect platform is chosen in quiet mode', async () => {
    it('it should exit after an error message has been shown', async () => {
      const wrongPlatformText = assumeWrongPlatform('very-wrong-platform');
      const randomModId = chance.word();

      await expect(add(wrongPlatformText, randomModId, {
        config: 'config.json',
        quiet: true
      }, logger)).rejects.toThrow(new Error('process.exit'));

      expect(logger.error).toHaveBeenCalledWith('Unknown platform "very-wrong-platform". Please use one of the following: curseforge, modrinth');

      // These mean that the add hasn't been recursively called
      expect(vi.mocked(ensureConfiguration)).toHaveBeenCalledTimes(1);
      expect(vi.mocked(fetchModDetails)).toHaveBeenCalledTimes(1);
      expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    });
  });

  describe('when the mod can\'t be found', async () => {
    it('it should handle with the correct interaction', async () => {

      const secondRandomMod = generateRemoteModDetails();

      vi.mocked(modNotFound).mockResolvedValueOnce({
        id: chance.word(),
        platform: getRandomPlatform()
      });

      const { randomPlatform, randomMod } = assumeModNotFound();

      vi.mocked(fetchModDetails).mockResolvedValueOnce(secondRandomMod.generated);
      assumeDownloadIsSuccessful();

      await add(randomPlatform, randomMod, { config: 'config.json' }, logger);

      expect(vi.mocked(modNotFound)).toHaveBeenCalledWith(randomMod, randomPlatform, logger, { config: 'config.json' });

      // Validate our assumptions about the work being done
      expect(vi.mocked(ensureConfiguration)).toHaveBeenCalledTimes(2);
      expect(vi.mocked(fetchModDetails)).toHaveBeenCalledTimes(2);
      expect(vi.mocked(downloadFile)).toHaveBeenCalledOnce();
      expect(vi.mocked(writeConfigFile)).toHaveBeenCalledOnce();
      expect(vi.mocked(writeLockFile)).toHaveBeenCalledOnce();

    });
  });

  describe('When the download fails', () => {
    it('reports the correct error', async () => {
      const url = chance.url({ protocol: 'https' });
      const randomPlatform = getRandomPlatform();
      const randomModId = chance.word();
      vi.mocked(downloadFile).mockRejectedValueOnce(new DownloadFailedException(url));

      await expect(add(randomPlatform, randomModId, { config: 'config.json' }, logger)).rejects.toThrow('process.exit');

      expect(logger.error).toHaveBeenCalledOnce();
      const message = vi.mocked(logger.error).mock.calls[0][0];

      expect(message).toContain(url);
      expect(message).toContain('Error downloading file: ');
      expect(message).toContain('please try again');

    });
  });
});
