import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import { RedundantVersionException } from '../errors/RedundantVersionException.js';
import { DefaultOptions } from '../mmm.js';
import { fetchModDetails } from '../repositories/index.js';
import { readConfigFile } from './config.js';
import { Logger } from './Logger.js';
import { getLatestMinecraftVersion, verifyMinecraftVersion } from './minecraftVersionVerifier.js';
import { ModsJson } from './modlist.types.js';
import { verifyUpgradeIsPossible } from './verifyUpgrade.js';

interface LocalTestContext {
  randomConfiguration: ModsJson;
  logger: Logger;
  options: DefaultOptions;
  randomVersion: string;
}

vi.mock('../repositories/index.js');
vi.mock('../lib/Logger.js');
vi.mock('../lib/config.js');
vi.mock('../lib/minecraftVersionVerifier.js');

describe('The Upgrade Test Module', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();

    const mods = [generateModConfig().generated, generateModConfig().generated, generateModConfig().generated];

    context.randomConfiguration = generateModsJson({ mods: mods }).generated;
    context.logger = new Logger({} as never);
    context.options = {
      config: 'config.json',
      debug: false,
      quiet: false
    };
    context.randomVersion = chance.pickone(['1.19.2', '1.19.3', '1.17']);

    vi.mocked(readConfigFile).mockResolvedValueOnce(context.randomConfiguration);

  });

  describe('when using the "latest" keyword as the version', () => {
    it<LocalTestContext>('uses the latest available minecraft version', async ({ randomVersion, logger, options }) => {
      vi.mocked(getLatestMinecraftVersion).mockResolvedValueOnce(randomVersion);
      vi.mocked(verifyMinecraftVersion).mockResolvedValue(false); //Quickest way out of the function is to error it

      await expect(verifyUpgradeIsPossible('latest', options, logger)).rejects.toThrow();

      expect(getLatestMinecraftVersion).toHaveBeenCalledOnce();
      expect(verifyMinecraftVersion).toHaveBeenCalledWith(randomVersion);

    });
  });

  describe('when all mods support the given version', () => {
    it<LocalTestContext>('should report everything fine', async ({ randomVersion, logger, options }) => {
      vi.mocked(verifyMinecraftVersion).mockResolvedValue(true);

      vi.mocked(fetchModDetails).mockResolvedValue({} as any);

      const result = await verifyUpgradeIsPossible(randomVersion, options, logger);
      expect(result).toEqual({
        canUpgrade: true,
        version: randomVersion,
        modsInError: []
      });

    });
  });

  describe('when some mods don\'t support the given version', () => {
    it<LocalTestContext>('should report the affected mods', async ({
      randomConfiguration,
      randomVersion,
      logger,
      options
    }) => {
      const mod1 = generateModConfig().generated;
      const mod2 = generateModConfig().generated;

      randomConfiguration.mods = [mod1, mod2];
      const failingModIndex = chance.pickone([0, 1]);

      vi.mocked(readConfigFile).mockReset();
      vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);

      vi.mocked(verifyMinecraftVersion).mockResolvedValue(true);

      if (failingModIndex === 0) { // randomizing which mod fails just for good measure
        vi.mocked(fetchModDetails).mockRejectedValueOnce(new Error());
        vi.mocked(fetchModDetails).mockResolvedValueOnce({} as any);
      } else {
        vi.mocked(fetchModDetails).mockResolvedValueOnce({} as any);
        vi.mocked(fetchModDetails).mockRejectedValueOnce(new Error());
      }

      const result = await verifyUpgradeIsPossible(randomVersion, options, logger);
      expect(result).toEqual({
        canUpgrade: false,
        version: randomVersion,
        modsInError: [randomConfiguration.mods.at(failingModIndex)]
      });
    });
  });

  describe('when an invalid game version is used', () => {
    it<LocalTestContext>('should report the invalid version', async ({ randomVersion, options, logger }) => {
      vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(false);

      await expect(verifyUpgradeIsPossible(randomVersion, options, logger)).rejects.toThrow(IncorrectMinecraftVersionException);

    });
  });

  describe('when the same game version is used as the one in the current config', () => {
    it<LocalTestContext>('should report that the version is already set', async ({
      randomConfiguration,
      options,
      logger
    }) => {
      vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(true);
      await expect(verifyUpgradeIsPossible(randomConfiguration.gameVersion, options, logger)).rejects.toThrow(RedundantVersionException);
    });
  });

  describe('when there are no mods configured in the config', () => {
    it<LocalTestContext>('should report that everything is fine', async ({
      randomConfiguration,
      randomVersion,
      logger,
      options
    }) => {
      randomConfiguration.mods = [];

      vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(true);
      const result = await verifyUpgradeIsPossible(randomVersion, options, logger);

      expect(result).toEqual({
        canUpgrade: true,
        version: randomVersion,
        modsInError: []
      });

      expect(fetchModDetails).not.toHaveBeenCalled();

    });
  });

  it<LocalTestContext>('should delegate the correct information to the details fetching', async ({
    randomConfiguration,
    randomVersion,
    logger,
    options
  }) => {

    const mod1 = generateModConfig();
    const mod2 = generateModConfig();
    const mod2Deets = mod2.generated;
    delete mod2Deets.allowedReleaseTypes;

    randomConfiguration.mods = [mod1.generated, mod2Deets];

    vi.mocked(readConfigFile).mockReset();
    vi.mocked(readConfigFile).mockResolvedValueOnce(randomConfiguration);

    vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(true);
    await verifyUpgradeIsPossible(randomVersion, options, logger);

    const expected1 = mod1.expected;
    const expected2 = mod2.expected;

    expect(fetchModDetails).toHaveBeenCalledWith(
      expected1.type,
      expected1.id,
      expected1.allowedReleaseTypes,
      randomVersion,
      randomConfiguration.loader,
      randomConfiguration.allowVersionFallback
    );

    expect(fetchModDetails).toHaveBeenCalledWith(
      expected2.type,
      expected2.id,
      randomConfiguration.defaultAllowedReleaseTypes,
      randomVersion,
      randomConfiguration.loader,
      randomConfiguration.allowVersionFallback
    );
  });
});
