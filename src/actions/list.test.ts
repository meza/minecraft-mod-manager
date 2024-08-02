import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { expectCommandStartTelemetry } from '../../test/telemetryHelper.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, readLockFile } from '../lib/config.js';
import { DefaultOptions } from '../mmm.js';
import { list } from './list.js';

vi.mock('../lib/Logger.js');
vi.mock('../lib/config.js');
vi.mock('../mmm.js');

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

      const mod3 = generateModConfig({ name: 'mod3.jar', id: 'mod3id' }).generated;
      const mod1 = generateModConfig({ name: 'mod1.jar', id: 'mod1id' }).generated;
      const mod2 = generateModConfig({ name: 'mod2.jar', id: 'mod2id' }).generated;

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
      expect(logger.log).toHaveBeenNthCalledWith(2, '\u2705 mod1.jar (mod1id) is installed', true);
      expect(logger.log).toHaveBeenNthCalledWith(3, '\u2705 mod2.jar (mod2id) is installed', true);
      expect(logger.log).toHaveBeenNthCalledWith(4, '\u2705 mod3.jar (mod3id) is installed', true);
    });
  });

  describe('when some of the mods are not installed', () => {
    it<LocalTestContext>('it should list all the mods appropriately', async ({ options, logger }) => {
      const randomConfig = generateModsJson().generated;

      const mod1 = generateModConfig({ name: 'mod1.jar', id: 'mod1id' }).generated;
      const mod2 = generateModConfig({ name: 'mod2.jar', id: 'mod2id' }).generated;
      const mod3 = generateModConfig({ name: 'mod3.jar', id: 'mod3id' }).generated;

      randomConfig.mods = [mod1, mod2, mod3];

      const installedMods = [
        generateModInstall({ id: mod1.id, type: mod1.type }).generated,
        generateModInstall({ id: mod3.id, type: mod3.type }).generated
      ];

      vi.mocked(readLockFile).mockResolvedValueOnce(installedMods);

      vi.mocked(ensureConfiguration).mockResolvedValue(randomConfig);

      await list(options, logger);

      expect(logger.log).toHaveBeenNthCalledWith(1, 'Configured mods', true);
      expect(logger.log).toHaveBeenNthCalledWith(2, '\u2705 mod1.jar (mod1id) is installed', true);
      expect(logger.log).toHaveBeenNthCalledWith(3, '\u274c mod2.jar (mod2id) is not installed', true);
      expect(logger.log).toHaveBeenNthCalledWith(4, '\u2705 mod3.jar (mod3id) is installed', true);
    });
  });

  it<LocalTestContext>('calls the correct telemetry', async ({ options, logger }) => {
    const randomConfig = generateModsJson().generated;
    vi.mocked(ensureConfiguration).mockResolvedValue(randomConfig);

    await list(options, logger);

    expectCommandStartTelemetry({
      command: 'list',
      success: true,
      duration: expect.any(Number),
      arguments: options
    });
  });
});
