import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { add } from './add.js';
import { ModDetails, ModlistConfig, Platform } from '../lib/modlist.types.js';
import { readConfigFile, writeConfigFile } from '../lib/config.js';
import { fetchModDetails } from '../repositories/index.js';
import { downloadFile } from '../lib/downloader.js';
import { generateModlist } from '../../test/modlistGenerator.js';
import { generateModDetails } from '../../test/modDetailsGenerator.js';
import { GeneratorResult } from '../../test/test.types.js';
import { chance } from 'jest-chance';
import { generateModConfig } from '../../test/modConfigGenerator.js';

vi.mock('../lib/config.js');
vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');

interface LocalTestContext {
  randomConfiguration: GeneratorResult<ModlistConfig>;
  randomModDetails: GeneratorResult<ModDetails>;
}

const assumeDownloadIsSuccessful = () => {
  vi.mocked(downloadFile).mockResolvedValueOnce();
};

describe('The add module', async () => {
  beforeEach<LocalTestContext>((context) => {
    context.randomConfiguration = generateModlist();

    // the main configuration to work with
    vi.mocked(readConfigFile).mockResolvedValueOnce(context.randomConfiguration.generated);

    // the mod details returned from the repository
    context.randomModDetails = generateModDetails();
    vi.mocked(fetchModDetails).mockResolvedValueOnce(context.randomModDetails.generated);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it<LocalTestContext>('should add a mod to the configuration', async (
    { randomConfiguration, randomModDetails }
  ) => {

    const randomPlatform = chance.pickone(['fabric', 'forge']);
    const randomModId = chance.word();

    assumeDownloadIsSuccessful();

    await add(randomPlatform, randomModId, { config: 'config.json' });

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
          name: randomModDetails.expected.name,
          installed: {
            fileName: randomModDetails.expected.fileName,
            releasedOn: randomModDetails.expected.releaseDate,
            hash: randomModDetails.expected.hash
          },
          allowedReleaseTypes: randomConfiguration.expected.defaultAllowedReleaseTypes
        }
      ]
    };

    expect(
      vi.mocked(writeConfigFile),
      'Writing the configuration file after adding a mod has failes'
    ).toHaveBeenCalledWith(expectedConfiguration, 'config.json');

  });

  it<LocalTestContext>('should skip the download if the mod already exists', async (context) => {
    const randomPlatform = chance.pickone(Object.values(Platform));
    const randomModId = chance.word();

    const randomModDetails = generateModConfig({
      type: randomPlatform,
      id: randomModId
    });

    context.randomConfiguration.generated.mods = [randomModDetails.generated];

    await add(randomPlatform, randomModId, { config: 'config.json' });

    expect(
      vi.mocked(fetchModDetails),
      'Fetched the mod details even though the mod already exists'
    ).toHaveBeenCalledTimes(0);

    expect(
      vi.mocked(downloadFile),
      'The download was called even though the mod already exists'
    ).toHaveBeenCalledTimes(0);
  });

  it<LocalTestContext>('should not show a debug message when it is not asked for', async (context) => {
    const consoleSpy = vi.spyOn(console, 'debug');

    const randomPlatform = Platform.CURSEFORGE;
    const randomModId = 'a-mod-id';
    const isDebug = false;

    const randomModDetails = generateModConfig({
      type: randomPlatform,
      id: randomModId
    });

    context.randomConfiguration.generated.mods = [randomModDetails.generated];

    await add(randomPlatform, randomModId, { config: 'config.json', debug: isDebug });

    expect(
      consoleSpy,
      'The debug message was not logged'
    ).not.toHaveBeenCalled();
  });

  it<LocalTestContext>('should show a debug message when it is not asked for', async (context) => {
    const consoleSpy = vi.spyOn(console, 'debug');

    const randomPlatform = Platform.MODRINTH;
    const randomModId = 'another-mod-id';
    const isDebug = true;

    const randomModDetails = generateModConfig({
      type: randomPlatform,
      id: randomModId
    });

    context.randomConfiguration.generated.mods = [randomModDetails.generated];

    await add(randomPlatform, randomModId, { config: 'config.json', debug: isDebug });

    expect(
      consoleSpy,
      'The debug message was not logged'
    ).toHaveBeenCalledWith('Mod another-mod-id for modrinth already exists in the configuration');
  });

});
