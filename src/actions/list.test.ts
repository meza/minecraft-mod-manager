import { describe, it, vi, expect, afterEach } from 'vitest';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { list } from './list.js';
import { readConfigFile, readLockFile } from '../lib/config.js';
import { generateModInstall } from '../../test/modInstallGenerator.js';

vi.mock('../lib/config.js');

describe('The list action', async () => {

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe('when all the mods are installed', () => {
    it('it should list all the mods', async () => {
      const consoleSpy = vi.spyOn(console, 'log');
      consoleSpy.mockImplementation(() => { });

      const randomConfig = generateModsJson().generated;

      const mod1 = generateModConfig({ name: 'mod1.jar' }).generated;
      const mod2 = generateModConfig({ name: 'mod2.jar' }).generated;
      const mod3 = generateModConfig({ name: 'mod3.jar' }).generated;

      randomConfig.mods = [mod1, mod2, mod3];

      vi.mocked(readConfigFile).mockResolvedValue(randomConfig);

      const installedMods = [
        generateModInstall({ id: mod1.id, type: mod1.type }).generated,
        generateModInstall({ id: mod2.id, type: mod2.type }).generated,
        generateModInstall({ id: mod3.id, type: mod3.type }).generated
      ];

      vi.mocked(readLockFile).mockResolvedValueOnce(installedMods);

      await list({ config: 'config.json' });

      expect(consoleSpy).toHaveBeenNthCalledWith(1, 'Configured mods');
      expect(consoleSpy).toHaveBeenNthCalledWith(2, '\u2705', 'mod1.jar', 'is installed');
      expect(consoleSpy).toHaveBeenNthCalledWith(3, '\u2705', 'mod2.jar', 'is installed');
      expect(consoleSpy).toHaveBeenNthCalledWith(4, '\u2705', 'mod3.jar', 'is installed');

    });
  });

  describe('when some of the mods are not installed', () => {
    it('it should list all the mods appropriately', async () => {
      const consoleSpy = vi.spyOn(console, 'log');
      consoleSpy.mockImplementation(() => { });

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

      vi.mocked(readConfigFile).mockResolvedValue(randomConfig);

      await list({ config: 'config.json' });

      expect(consoleSpy).toHaveBeenNthCalledWith(1, 'Configured mods');
      expect(consoleSpy).toHaveBeenNthCalledWith(2, '\u2705', 'mod1.jar', 'is installed');
      expect(consoleSpy).toHaveBeenNthCalledWith(3, '\u274c', 'mod2.jar', 'is not installed');
      expect(consoleSpy).toHaveBeenNthCalledWith(4, '\u2705', 'mod3.jar', 'is installed');

    });
  });
});
