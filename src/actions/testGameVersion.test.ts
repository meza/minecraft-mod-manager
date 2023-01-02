import { describe, vi, it, beforeEach, expect } from 'vitest';
import { chance } from 'jest-chance';
import { UpgradeVerificationResult, verifyUpgradeIsPossible } from '../lib/verifyUpgrade.js';
import { testGameVersion } from './testGameVersion.js';
import { Logger } from '../lib/Logger.js';
import { DefaultOptions } from '../mmm.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import { RedundantVersionException } from '../errors/RedundantVersionException.js';

vi.mock('../lib/verifyUpgrade.js');
vi.mock('../lib/Logger.js');

interface LocalTestContext {
  version: string;
  logger: Logger;
  options: DefaultOptions;
}

describe('The test action', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();

    context.version = chance.word();
    context.logger = new Logger({} as never);
    context.options = {
      config: 'config.json',
      debug: false,
      quiet: false
    };

  });
  describe('when the mods can be upgraded safely', () => {
    it<LocalTestContext>('should report success for the given version', async ({ options, logger, version }) => {

      const result = {
        canUpgrade: true,
        version: version
      } as unknown as UpgradeVerificationResult;
      vi.mocked(verifyUpgradeIsPossible).mockResolvedValueOnce(result);

      const actual = await testGameVersion(version, options, logger);

      const logMessage = vi.mocked(logger.log).mock.calls[0][0];

      expect(logMessage).toContain('All mods have support for ');
      expect(logMessage).toContain(version);
      expect(logMessage).toContain('You can safely upgrade.');

      expect(actual).toBe(result);

    });
  });

  describe('when some of the mods cannot be upgraded safely', () => {
    it<LocalTestContext>('reports the error as expected', async ({ options, logger, version }) => {
      const mod1 = generateModConfig().generated;
      const mod2 = generateModConfig().generated;
      vi.mocked(verifyUpgradeIsPossible).mockResolvedValueOnce({
        canUpgrade: false,
        version: version,
        modsInError: [mod1, mod2]
      });

      await testGameVersion(version, options, logger);

      expect(logger.log).toHaveBeenNthCalledWith(1, expect.stringContaining('Some mods are missing support for '));
      expect(logger.log).toHaveBeenNthCalledWith(1, expect.stringContaining(version));
      expect(logger.log).toHaveBeenNthCalledWith(2, expect.stringContaining(mod1.name));
      expect(logger.log).toHaveBeenNthCalledWith(3, expect.stringContaining(mod2.name));

      expect(logger.error).toHaveBeenCalledWith(expect.stringContaining('You cannot upgrade to'), 1);
      expect(logger.error).toHaveBeenCalledWith(expect.stringContaining(version), 1);
    });
  });

  describe('when an incorrect version is supplied', () => {
    it<LocalTestContext>('exits with the correct error', async ({ options, logger }) => {
      vi.mocked(verifyUpgradeIsPossible).mockRejectedValueOnce(new IncorrectMinecraftVersionException('bad-version'));

      await testGameVersion('bad-version', options, logger);

      expect(logger.error).toHaveBeenCalledWith('The specified Minecraft version (bad-version) is not valid.', 1);

    });
  });

  describe('when an redundant version is supplied', () => {
    it<LocalTestContext>('exits with the correct error', async ({ options, logger }) => {
      vi.mocked(verifyUpgradeIsPossible).mockRejectedValueOnce(new RedundantVersionException('redundant-version'));

      await testGameVersion('redundant-version', options, logger);

      expect(logger.error).toHaveBeenCalledWith('You\'re already using (redundant-version).', 2);

    });
  });
});
