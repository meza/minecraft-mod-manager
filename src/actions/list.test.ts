import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { list } from './list.js';
import { ensureConfiguration, readLockFile } from '../lib/config.js';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { Logger } from '../lib/Logger.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { ErrorTexts } from '../errors/ErrorTexts.js';
import { chance } from 'jest-chance';
import { DefaultOptions } from '../mmm.js';

vi.mock('../lib/Logger.js');
vi.mock('../lib/config.js');

interface LocalTestContext {
  options: DefaultOptions;
  logger: Logger;
}

describe('The list action', async () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.options = {
      config: 'config.json',
      quiet: false,
      debug: false
    };

    vi.mocked(context.logger.error).mockImplementation(() => {
      throw new Error('process.exit');
    });

  });

  describe('when all the mods are installed', () => {
    it<LocalTestContext>('it should list all the mods sorted', async ({ options, logger }) => {

      const randomConfig = generateModsJson().generated;

      const mod3 = generateModConfig({ name: 'mod3.jar' }).generated;
      const mod1 = generateModConfig({ name: 'mod1.jar' }).generated;
      const mod2 = generateModConfig({ name: 'mod2.jar' }).generated;

      randomConfig.mods = [mod3, mod1, mod2];

      vi.mocked(ensureConfiguration).mockResolvedValue(randomConfig);

      const installedMods = [
        generateModInstall({ id: mod3.id, type: mod3.type }).generated,
        generateModInstall({ id: mod1.id, type: mod1.type }).generated,
        generateModInstall({ id: mod2.id, type: mod2.type }).generated
      ];

      vi.mocked(readLockFile).mockResolvedValueOnce(installedMods);

      await list(options, logger);

      expect(logger.log).toHaveBeenNthCalledWith(1, 'Configured mods', true);
      expect(logger.log).toHaveBeenNthCalledWith(2, '\u2705 mod1.jar is installed', true);
      expect(logger.log).toHaveBeenNthCalledWith(3, '\u2705 mod2.jar is installed', true);
      expect(logger.log).toHaveBeenNthCalledWith(4, '\u2705 mod3.jar is installed', true);

    });
  });

  describe('when some of the mods are not installed', () => {
    it<LocalTestContext>('it should list all the mods appropriately', async ({ options, logger }) => {
      const randomConfig = generateModsJson().generated;

      const mod1 = generateModConfig({ name: 'mod1.jar' }).generated;
      const mod2 = generateModConfig({ name: 'mod2.jar' }).generated;
      const mod3 = generateModConfig({ name: 'mod3.jar' }).generated;

      randomConfig.mods = [mod1, mod2, mod3];

      const installedMods = [
        generateModInstall({ id: mod1.id, type: mod1.type }).generated,
        generateModInstall({ id: mod3.id, type: mod3.type }).generated
      ];

      vi.mocked(readLockFile).mockResolvedValueOnce(installedMods);

      vi.mocked(ensureConfiguration).mockResolvedValue(randomConfig);

      await list(options, logger);

      expect(logger.log).toHaveBeenNthCalledWith(1, 'Configured mods', true);
      expect(logger.log).toHaveBeenNthCalledWith(2, '\u2705 mod1.jar is installed', true);
      expect(logger.log).toHaveBeenNthCalledWith(3, '\u274c mod2.jar is not installed', true);
      expect(logger.log).toHaveBeenNthCalledWith(4, '\u2705 mod3.jar is installed', true);

    });

    it<LocalTestContext>('shows the correct error message when the config file is missing', async ({ options, logger }) => {
      vi.mocked(ensureConfiguration).mockRejectedValueOnce(new ConfigFileNotFoundException(options.config));
      await expect(list(options, logger)).rejects.toThrow('process.exit');

      expect(vi.mocked(logger.error)).toHaveBeenCalledWith(ErrorTexts.configNotFound);

    });
  });

  it<LocalTestContext>('handles unexpected errors', async ({ options, logger }) => {
    const randomErrorMessage = chance.sentence();
    vi.mocked(ensureConfiguration).mockRejectedValueOnce(new Error(randomErrorMessage));
    await expect(list(options, logger)).rejects.toThrow('process.exit');
    expect(logger.error).toHaveBeenCalledWith(randomErrorMessage, 2);
  });
});
