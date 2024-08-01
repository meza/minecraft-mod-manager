import fs from 'node:fs/promises';
import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { expectCommandStartTelemetry } from '../../test/telemetryHelper.js';
import { Logger } from '../lib/Logger.js';
import {
  ensureConfiguration,
  fileExists,
  getModsFolder,
  readLockFile,
  writeConfigFile,
  writeLockFile
} from '../lib/config.js';
import { DefaultOptions } from '../mmm.js';
import { changeGameVersion } from './change.js';
import { install } from './install.js';
import { testGameVersion } from './testGameVersion.js';

vi.mock('../lib/Logger.js');
vi.mock('./install.js');
vi.mock('../lib/config.js');
vi.mock('./testGameVersion.js');
vi.mock('node:fs/promises');
vi.mock('../mmm.js');

interface LocalTestContext {
  version: string;
  logger: Logger;
  options: DefaultOptions;
}

describe('The change action', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();

    context.version = chance.word();
    context.logger = new Logger({} as never);
    context.options = {
      config: 'config.json',
      debug: false,
      quiet: false
    };

    vi.mocked(install).mockResolvedValueOnce();
    vi.mocked(testGameVersion).mockResolvedValueOnce({
      version: context.version
    } as never);
  });

  it<LocalTestContext>('changes the version in the config', async ({ version, options, logger }) => {
    const config = generateModsJson({ gameVersion: 'old' }).generated;
    const quietFlag = chance.bool();
    options.quiet = quietFlag;

    vi.mocked(ensureConfiguration).mockResolvedValueOnce(config);
    vi.mocked(getModsFolder).mockReturnValue(config.modsFolder);
    vi.mocked(readLockFile).mockResolvedValue([]);

    await changeGameVersion(version, options, logger);

    expect(writeConfigFile).toHaveBeenCalledOnce();

    const savedConfig = vi.mocked(writeConfigFile).mock.calls[0][0];

    expect(savedConfig.gameVersion).not.toEqual('old');
    expect(savedConfig.gameVersion).toEqual(version);
    expect(ensureConfiguration).toHaveBeenCalledWith(options.config, logger, quietFlag);
  });

  it<LocalTestContext>('empties the lockfile', async ({ version, options, logger }) => {
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(generateModsJson().generated);
    vi.mocked(readLockFile).mockResolvedValue([generateModInstall().generated]);

    await changeGameVersion(version, options, logger);

    expect(writeLockFile).toHaveBeenCalledOnce();

    expect(writeLockFile).toHaveBeenCalledWith([], options, logger);
  });

  it<LocalTestContext>('calls the install module', async ({ version, options, logger }) => {
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(generateModsJson().generated);
    vi.mocked(readLockFile).mockResolvedValue([]);

    await changeGameVersion(version, options, logger);

    expect(install).toHaveBeenCalledOnce();
  });

  it<LocalTestContext>('removes the local installations', async ({ version, options, logger }) => {
    vi.mocked(fileExists).mockResolvedValue(true);

    const install1 = generateModInstall({ fileName: 'mymod1' }).generated;
    const install2 = generateModInstall({ fileName: 'mymod2' }).generated;
    const install3 = generateModInstall({ fileName: 'mymod3' }).generated;

    vi.mocked(readLockFile).mockResolvedValueOnce([install1, install2, install3]);
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(
      generateModsJson({
        mods: [
          generateModConfig({ id: install1.id, type: install1.type }).generated,
          generateModConfig({ id: install2.id, type: install2.type }).generated,
          generateModConfig({ id: install3.id, type: install3.type }).generated
        ],
        modsFolder: '/mods'
      }).generated
    );
    vi.mocked(getModsFolder).mockReturnValue('/mods');

    await changeGameVersion(version, options, logger);

    expect(fs.rm).toHaveBeenCalledTimes(3);

    expect(fs.rm).toHaveBeenCalledWith(expect.stringContaining(install1.fileName));
    expect(fs.rm).toHaveBeenCalledWith(expect.stringContaining(install2.fileName));
    expect(fs.rm).toHaveBeenCalledWith(expect.stringContaining(install3.fileName));
  });

  it<LocalTestContext>('ignores mods that are not installed', async ({ version, options, logger }) => {
    vi.mocked(fileExists).mockResolvedValue(true);

    const install1 = generateModInstall({ fileName: 'mymod1' }).generated;
    const install2 = generateModInstall({ fileName: 'mymod2' }).generated;
    const install3 = generateModInstall({ fileName: 'mymod3' }).generated;

    vi.mocked(readLockFile).mockResolvedValueOnce([install1, install3]);

    vi.mocked(ensureConfiguration).mockResolvedValue(
      generateModsJson({
        modsFolder: '/mods',
        mods: [
          generateModConfig({ id: install1.id, type: install1.type }).generated,
          generateModConfig({ id: install2.id, type: install2.type }).generated,
          generateModConfig({ id: install3.id, type: install3.type }).generated
        ]
      }).generated
    );
    vi.mocked(getModsFolder).mockReturnValue('/mods');

    await changeGameVersion(version, options, logger);

    expect(fs.rm).not.toHaveBeenCalledWith(expect.stringContaining(install2.fileName));

    expect(fs.rm).toHaveBeenCalledTimes(2);

    expect(fs.rm).toHaveBeenCalledWith(expect.stringContaining(install1.fileName));
    expect(fs.rm).toHaveBeenCalledWith(expect.stringContaining(install3.fileName));
  });

  it<LocalTestContext>('calls the correct telemetry', async ({ version, options, logger }) => {
    vi.mocked(ensureConfiguration).mockResolvedValueOnce(generateModsJson().generated);
    vi.mocked(readLockFile).mockResolvedValue([]);

    await changeGameVersion(version, options, logger);

    expectCommandStartTelemetry({
      command: 'change',
      success: true,
      duration: expect.any(Number),
      arguments: {
        gameVersion: version,
        options: options
      }
    });
  });
});
