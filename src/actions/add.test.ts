import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { add } from './add.js';
import { initializeConfigFile, readConfigFile, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { fetchModDetails } from '../repositories/index.js';
import { downloadFile } from '../lib/downloader.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';
import { GeneratorResult } from '../../test/test.types.js';
import { chance } from 'jest-chance';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { UnknownPlatformException } from '../errors/UnknownPlatformException.js';
import inquirer from 'inquirer';
import { CouldNotFindModException } from '../errors/CouldNotFindModException.js';
import { NoRemoteFileFound } from '../errors/NoRemoteFileFound.js';
import { ModInstall, ModsJson, Platform, RemoteModDetails } from '../lib/modlist.types.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { shouldCreateConfig } from '../interactions/shouldCreateConfig.js';
import { Logger } from '../lib/Logger.js';

vi.mock('../lib/Logger.js');
vi.mock('../lib/config.js');
vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('inquirer');
vi.mock('../interactions/shouldCreateConfig.js');

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
  return chance.pickone(Object.values(Platform));
};

describe('The add module', async () => {
  let logger: Logger;
  beforeEach<LocalTestContext>((context) => {
    logger = new Logger({} as never);
    context.randomConfiguration = generateModsJson();

    // the main configuration to work with
    vi.mocked(readConfigFile).mockResolvedValue(context.randomConfiguration.generated);
    vi.mocked(readLockFile).mockResolvedValue([]);

    // the mod details returned from the repository
    context.randomModDetails = generateRemoteModDetails();
    vi.mocked(fetchModDetails).mockResolvedValueOnce(context.randomModDetails.generated);
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it<LocalTestContext>('should add a mod to the configuration', async (
    { randomConfiguration, randomModDetails }
  ) => {

    const randomPlatform = chance.pickone(['fabric', 'forge']);
    const randomModId = chance.word();

    assumeDownloadIsSuccessful();

    await add(randomPlatform, randomModId, { config: 'config.json' }, logger);

    expect(
      vi.mocked(readConfigFile),
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
    ).toHaveBeenCalledWith(expectedConfiguration, 'config.json');

    expect(
      vi.mocked(writeLockFile),
      'Writing the lock file after adding a mod has failed'
    ).toHaveBeenCalledWith(expectedLockFile, 'config.json');

  });

  describe('when the configuraiton file does not exist', () => {
    beforeEach(() => {
      vi.mocked(readConfigFile).mockReset();
      vi.mocked(readConfigFile).mockRejectedValueOnce(new ConfigFileNotFoundException('test-config.json'));
    });
    it('should report an error in quiet mode', async () => {
      const randomPlatform = getRandomPlatform();
      const randomModId = chance.word();

      await expect(async () => {
        await add(randomPlatform, randomModId, { config: 'test-config.json', quiet: true }, logger);
      }).rejects.toThrow(new ConfigFileNotFoundException('test-config.json'));
    });

    it<LocalTestContext>('should initialize the config in interactive mode when asked to', async ({
      randomConfiguration
    }) => {
      const randomPlatform = getRandomPlatform();
      const randomModId = chance.word();

      vi.mocked(shouldCreateConfig).mockResolvedValueOnce(true);
      vi.mocked(initializeConfigFile).mockResolvedValueOnce(randomConfiguration.generated);

      await add(randomPlatform, randomModId, { config: 'test-config.json', quiet: false }, logger);

      expect(vi.mocked(shouldCreateConfig)).toHaveBeenCalledOnce();

      expect(
        vi.mocked(writeConfigFile),
        'Writing the configuration file after auto-initializing the config has failed'
      ).toHaveBeenCalledWith(randomConfiguration.generated, 'test-config.json');

    });

    it('should throw an error if config creation was declined in interactive mode', async () => {
      const randomPlatform = getRandomPlatform();
      const randomModId = chance.word();

      vi.mocked(shouldCreateConfig).mockResolvedValueOnce(false);

      await expect(async () => {
        await add(randomPlatform, randomModId, { config: 'test-config.json', quiet: false }, logger);
      }).rejects.toThrow(new ConfigFileNotFoundException('test-config.json'));

      expect(vi.mocked(shouldCreateConfig)).toHaveBeenCalledOnce();

    });
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

    const randomPlatform = getRandomPlatform();
    const randomModId = chance.word();
    const randomVersion = randomConfiguration.expected.gameVersion;
    const randomLoader = randomConfiguration.expected.loader;

    vi.mocked(downloadFile).mockReset();
    vi.mocked(downloadFile).mockRejectedValueOnce(new NoRemoteFileFound(randomModId, randomPlatform));

    await add(randomPlatform, randomModId, { config: 'config.json' }, logger);

    expect(logger.error).toHaveBeenCalledWith(`Could not find a file for the version ${randomVersion} for ${randomLoader}`);

  });

  it('should report unexpected errors', async () => {

    const randomErrorMessage = chance.sentence();
    const randomPlatform = getRandomPlatform();
    const randomMod = chance.word();
    const error = new Error(randomErrorMessage);

    vi.mocked(fetchModDetails).mockReset();
    vi.mocked(fetchModDetails).mockRejectedValueOnce(error);

    await add(randomPlatform, randomMod, { config: 'config.json' }, logger);

    expect(logger.error).toHaveBeenCalledWith(randomErrorMessage, 2);

  });

  describe('when an incorrect platform is chosen in interactive mode', async () => {
    describe('and the user chooses to cancel', async () => {
      it('it should exit after the prompt', async () => {
        const wrongPlatformText = assumeWrongPlatform();
        const randomModId = chance.word();

        vi.mocked(inquirer.prompt).mockResolvedValueOnce({ platform: 'cancel' });

        await add(wrongPlatformText, randomModId, { config: 'config.json' }, logger);

        // @ts-ignore anyone with a fix for this?
        const inquirerOptions = vi.mocked(inquirer.prompt).mock.calls[0][0][0];

        expect(inquirerOptions.choices.sort()).toEqual(['cancel', ...Object.values(Platform)].sort());
        expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledTimes(1);

        // These mean that the add hasn't been recursively called
        expect(vi.mocked(readConfigFile)).toHaveBeenCalledTimes(1);
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
        expect(vi.mocked(readConfigFile)).toHaveBeenCalledTimes(2);
        expect(vi.mocked(writeConfigFile)).toHaveBeenCalledTimes(1);

      });
    });
  });

  describe('when an incorrect platform is chosen in quiet mode', async () => {
    it('it should exit after an error message has been shown', async () => {
      const wrongPlatformText = assumeWrongPlatform('very-wrong-platform');
      const randomModId = chance.word();

      await add(wrongPlatformText, randomModId, { config: 'config.json', quiet: true }, logger);

      expect(logger.error).toHaveBeenCalledWith('Unknown platform "very-wrong-platform". Please use one of the following: curseforge, modrinth');

      // These mean that the add hasn't been recursively called
      expect(vi.mocked(readConfigFile)).toHaveBeenCalledTimes(1);
      expect(vi.mocked(fetchModDetails)).toHaveBeenCalledTimes(1);
      expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();
    });
  });

  describe('when the mod can\'t be found', async () => {
    it('it should exit after an error message has been shown', async () => {

      const { randomPlatform, randomMod } = assumeModNotFound();

      await add(randomPlatform, randomMod, { config: 'config.json' }, logger);

      expect(logger.error).toHaveBeenCalledWith(`Mod "${randomMod}" for ${randomPlatform} does not exist`);

      // Validate our assumptions about the work being done
      expect(vi.mocked(readConfigFile)).toHaveBeenCalledTimes(1);
      expect(vi.mocked(fetchModDetails)).toHaveBeenCalledTimes(1);
      expect(vi.mocked(downloadFile)).not.toHaveBeenCalled();

    });
  });
});
