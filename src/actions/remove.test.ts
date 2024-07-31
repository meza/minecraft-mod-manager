import path from 'node:path';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { expectCommandStartTelemetry } from '../../test/telemetryHelper.js';
import { Mod, ModInstall, ModsJson } from '../lib/modlist.types.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { removeAction, RemoveOptions } from './remove.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, getModsFolder, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { findLocalMods, getInstallation, hasInstallation } from '../lib/configurationHelper.js';
import { chance } from 'jest-chance';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import fs from 'fs/promises';

interface LocalTestContext {
  configuration: ModsJson;
  installations: ModInstall[];
  options: RemoveOptions;
  logger: Logger;
}

vi.mock('../mmm.js');
vi.mock('../lib/Logger.js');
vi.mock('../lib/config.js');
vi.mock('../lib/configurationHelper.js');
vi.mock('fs/promises');

describe('The remove action', () => {

  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();

    context.configuration = generateModsJson().generated;
    context.installations = [];
    context.options = {
      debug: false,
      config: 'config.js',
      dryRun: false,
      quiet: false
    };
    context.logger = new Logger({} as never);

    vi.mocked(ensureConfiguration).mockResolvedValueOnce(context.configuration);
    vi.mocked(readLockFile).mockResolvedValueOnce(context.installations);

  });

  it<LocalTestContext>('passes the correct inputs to the locator', async ({ options, logger, configuration }) => {
    vi.mocked(findLocalMods).mockReturnValueOnce(new Set<Mod>());

    const input = chance.n(chance.word, chance.integer({ min: 1, max: 10 }));
    await removeAction(input, options, logger);

    expect(findLocalMods).toHaveBeenCalledOnce();
    expect(findLocalMods).toHaveBeenCalledWith(input, configuration);

  });

  describe('when in dry-run mode', () => {
    it<LocalTestContext>('logs the correct thing when no mods are found', async ({ options, logger }) => {
      vi.mocked(findLocalMods).mockReturnValueOnce(new Set<Mod>());

      options.dryRun = true;
      await removeAction([], options, logger);

      expect(logger.log).toHaveBeenCalledTimes(1);
      const message = vi.mocked(logger.log).mock.calls[0][0];
      expect(message).toMatchInlineSnapshot('"Running in dry-run mode. Nothing will actually be removed."');
    });

    it<LocalTestContext>('logs the correct things when mods are found', async ({ options, logger }) => {
      options.dryRun = true;
      const mod1 = generateModConfig({ name: 'mod1' }).generated;
      const mod2 = generateModConfig({ name: 'mod2' }).generated;
      const mod3 = generateModConfig({ name: 'mod3' }).generated;
      vi.mocked(findLocalMods).mockReturnValueOnce(new Set([mod1, mod2, mod3]));
      await removeAction([], options, logger);

      expect(logger.log).toHaveBeenCalledTimes(4);
      const messages = vi.mocked(logger.log).mock.calls;
      expect(messages[0][0]).toMatchInlineSnapshot('"Running in dry-run mode. Nothing will actually be removed."');
      expect(messages[1][0]).toMatchInlineSnapshot('"Would have removed mod1"');
      expect(messages[2][0]).toMatchInlineSnapshot('"Would have removed mod2"');
      expect(messages[3][0]).toMatchInlineSnapshot('"Would have removed mod3"');
    });
  });

  describe('when there are no installations for the mods', () => {
    it<LocalTestContext>('removes the mod only from the configuration without touching anything else', async ({
      options,
      logger
    }) => {
      vi.mocked(ensureConfiguration).mockReset();
      vi.mocked(readLockFile).mockReset();

      const mod1 = generateModConfig({ name: 'removed mod1' }).generated;
      const mod2 = generateModConfig().generated;
      const mod3 = generateModConfig({ name: 'removed mod3' }).generated;
      const config = generateModsJson({ mods: [mod1, mod2, mod3] }).generated;

      const expectedConfig = structuredClone(config);
      expectedConfig.mods = [mod2];

      vi.mocked(ensureConfiguration).mockResolvedValueOnce(config);
      vi.mocked(readLockFile).mockResolvedValueOnce([]); //no installations

      vi.mocked(findLocalMods).mockReturnValueOnce(new Set<Mod>([mod1, mod3])); //what are we removing?

      await removeAction([], options, logger);

      expect(writeConfigFile).toHaveBeenCalledWith(expectedConfig, options, logger);
      expect(writeLockFile).not.toHaveBeenCalled();
      expect(logger.log).toHaveBeenCalledTimes(2);

      const logCalls = vi.mocked(logger.log).mock.calls;
      expect(logCalls[0][0]).toMatchInlineSnapshot('"Removed removed mod1"');
      expect(logCalls[1][0]).toMatchInlineSnapshot('"Removed removed mod3"');
    });
  });

  describe('when there are local files for the mods', () => {
    it<LocalTestContext>('removes everything', async ({
      options,
      logger
    }) => {
      vi.mocked(ensureConfiguration).mockReset();
      vi.mocked(readLockFile).mockReset();
      vi.mocked(getInstallation).mockRestore();
      vi.mocked(hasInstallation).mockReset();
      vi.mocked(hasInstallation).mockRestore();

      const mod1 = generateModConfig({ name: 'removed mod1' }).generated;
      const mod2 = generateModConfig({ name: 'removed mod2' }).generated;
      const mod3 = generateModConfig().generated;

      const mod1Filename = 'file1';
      const mod2Filename = 'file2';

      const mod1Install = generateModInstall({
        type: mod1.type,
        id: mod1.id,
        fileName: mod1Filename
      }).generated;
      const mod2Install = generateModInstall({
        type: mod2.type,
        id: mod2.id,
        fileName: mod2Filename
      }).generated;
      const mod3Install = generateModInstall({
        type: mod3.type,
        id: mod3.id
      }).generated;

      const config = generateModsJson({ mods: [mod1, mod2, mod3] }).generated;

      vi.mocked(getModsFolder).mockReturnValue('/mods');
      vi.mocked(ensureConfiguration).mockResolvedValueOnce(config);
      vi.mocked(readLockFile).mockResolvedValueOnce([
        mod1Install,
        mod2Install,
        mod3Install
      ]);

      vi.mocked(hasInstallation).mockReturnValue(true);
      vi.mocked(getInstallation).mockReturnValue(0);

      vi.mocked(findLocalMods).mockReturnValueOnce(new Set<Mod>([mod1, mod2])); //what are we removing?

      await removeAction([], options, logger);

      expect(fs.rm).toHaveBeenCalledTimes(2);
      expect(writeLockFile).toHaveBeenCalledTimes(2); //for both deleted mods

      expect(fs.rm).toHaveBeenNthCalledWith(1, path.resolve('/mods/file1'), { force: true });
      expect(writeLockFile).toHaveBeenNthCalledWith(1, [mod2Install, mod3Install], options, logger);

      expect(fs.rm).toHaveBeenNthCalledWith(2, path.resolve('/mods/file2'), { force: true });
      expect(writeLockFile).toHaveBeenNthCalledWith(2, [mod3Install], options, logger);
    });
  });

  it<LocalTestContext>('calls the correct telemetry', async ({ options, logger }) => {
    vi.mocked(findLocalMods).mockReturnValueOnce(new Set<Mod>());

    const input = chance.n(chance.word, chance.integer({ min: 1, max: 10 }));
    await removeAction(input, options, logger);

    expectCommandStartTelemetry({
      command: 'remove',
      success: true,
      duration: expect.any(Number),
      arguments: {
        mods: input,
        options: options
      }
    });

  });

});
