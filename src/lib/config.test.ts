import { beforeEach, describe, expect, it, vi } from 'vitest';
import fs from 'node:fs/promises';
import { chance } from 'jest-chance';
import {
  ensureConfiguration,
  fileExists,
  initializeConfigFile,
  readConfigFile,
  readLockFile,
  writeConfigFile,
  writeLockFile
} from './config.js';
import { ModInstall, ModsJson } from './modlist.types.js';
import path from 'node:path';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { initializeConfig } from '../interactions/initializeConfig.js';
import * as process from 'process';
import { shouldCreateConfig } from '../interactions/shouldCreateConfig.js';
import { Logger } from './Logger.js';
import { fileToWrite } from '../interactions/fileToWrite.js';
import { DefaultOptions } from '../mmm.js';

vi.mock('../interactions/shouldCreateConfig.js');
vi.mock('../interactions/initializeConfig.js');
vi.mock('node:fs/promises');
vi.mock('../interactions/fileToWrite.js');

interface LocalTestContext {
  options: DefaultOptions;
  logger: Logger;
}

describe('The config library', () => {
  let logger: Logger;
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.options = {
      config: 'config.json',
      debug: false,
      quiet: false
    };
    vi.mocked(fileToWrite).mockImplementation(async (file: string) => {
      return file;
    });
  });

  it('can determine that a file exists', async () => {
    vi.mocked(fs.access).mockResolvedValueOnce();
    expect(await fileExists(chance.word())).toBeTruthy();
  });

  it('can determine that a file does not exist', async () => {
    vi.mocked(fs.access).mockRejectedValueOnce(new Error());
    expect(await fileExists(chance.word())).toBeFalsy();
  });

  it<LocalTestContext>('can write the config file', async ({ options, logger }) => {
    const config = {
      something: 'value'
    } as unknown as ModsJson;

    await writeConfigFile(config, options, logger);

    expect(vi.mocked(fileToWrite)).toHaveBeenCalledWith(path.resolve(options.config), options, logger);

    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(
      path.resolve(options.config),
      '{\n'
      + '  "something": "value"\n'
      + '}'
    );
  });

  it<LocalTestContext>('can write the lock file', async ({ options, logger }) => {
    const config = [{
      something: 'value'
    }] as unknown as ModInstall[];
    options.config = chance.word();
    const expectedLockFilePath = path.resolve(`${options.config}-lock.json`);

    await writeLockFile(config, options, logger);
    expect(vi.mocked(fileToWrite)).toHaveBeenCalledWith(expectedLockFilePath, options, logger);

    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(
      expectedLockFilePath,
      '[\n'
      + '  {\n'
      + '    "something": "value"\n'
      + '  }\n'
      + ']'
    );
  });

  it<LocalTestContext>('can read the lock file when it exists', async ({ options, logger }) => {
    options.config = 'config.json';
    const lockfileName = 'config-lock.json';

    const randomModInstall = generateModInstall();

    const lockfileContents = JSON.stringify([randomModInstall.generated], null, 2);

    // Make file exist
    vi.mocked(fs.access).mockResolvedValueOnce();
    // Return file contents
    vi.mocked(fs.readFile).mockResolvedValueOnce(lockfileContents);

    const actualOutput = await readLockFile(options, logger);

    expect(actualOutput).toEqual([randomModInstall.expected]);

    expect(vi.mocked(fs.readFile)).toHaveBeenCalledWith(
      path.resolve(lockfileName),
      { encoding: 'utf8' }
    );

  });

  it<LocalTestContext>('can returns an empty array and creates the file when the lock file does not exist', async ({ options, logger }) => {
    options.config = 'config.json';

    // Make file exist
    vi.mocked(fs.access).mockRejectedValueOnce(new Error());

    const actualOutput = await readLockFile(options, logger);

    expect(actualOutput).toEqual([]);

    expect(vi.mocked(fs.readFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(
      path.resolve('config-lock.json'),
      '[]'
    );

  });

  it<LocalTestContext>('can read from the config file when it exists', async ({ options }) => {
    options.config = 'config.json';

    // File exists
    vi.mocked(fs.access).mockResolvedValueOnce();

    const randomModsJson = generateModsJson();
    const fileContents = JSON.stringify(randomModsJson.generated, null, 2);

    // Return config file contents
    vi.mocked(fs.readFile).mockResolvedValueOnce(fileContents);

    const actualOutput = await readConfigFile(options.config);

    expect(actualOutput).toEqual(randomModsJson.expected);
  });

  it('throws an error when the config file does not exist', async () => {
    const configName = path.resolve('config.json');

    // File does not exist
    vi.mocked(fs.access).mockRejectedValueOnce(new Error());

    await expect(readConfigFile(configName)).rejects.toThrow(new ConfigFileNotFoundException(configName));
  });

  it('can initialize a new config file', async () => {
    const configName = 'config.json';

    const randomConfig = generateModsJson().generated;

    vi.mocked(initializeConfig).mockResolvedValueOnce(randomConfig);
    const actual = await initializeConfigFile(configName, logger);

    expect(actual).toEqual(randomConfig);
    expect(vi.mocked(initializeConfig)).toHaveBeenCalledWith({ config: configName }, process.cwd(), logger);

  });

  describe('when ensuring that a config exists', () => {
    describe('and in interactive mode', () => {
      it('creates a new one if there is no existing one', async () => {
        const configName = 'config.json';
        const randomConfig = generateModsJson().generated;

        //File does not exist
        vi.mocked(fs.access).mockRejectedValueOnce(new Error());
        vi.mocked(shouldCreateConfig).mockResolvedValueOnce(true);
        vi.mocked(initializeConfig).mockResolvedValueOnce(randomConfig);

        const actual = await ensureConfiguration(configName, logger);

        expect(actual).toEqual(randomConfig);
        expect(initializeConfig).toHaveBeenCalledOnce();
      });

      it('throws an error if the user chose to not create one', async () => {
        const configName = 'config.json';

        //File does not exist
        vi.mocked(fs.access).mockRejectedValueOnce(new Error());
        vi.mocked(shouldCreateConfig).mockResolvedValueOnce(false);

        await expect(ensureConfiguration(configName, logger)).rejects.toThrow(ConfigFileNotFoundException);

        expect(initializeConfig).not.toHaveBeenCalled();
      });
    });

    describe('and in non-interactive mode', () => {
      it('throws an error', async () => {
        const configName = 'config.json';
        const quiet = true;

        //File does not exist
        vi.mocked(fs.access).mockRejectedValueOnce(new Error());

        await expect(ensureConfiguration(configName, logger, quiet)).rejects.toThrow(ConfigFileNotFoundException);
      });
    });

    it('returns the existing one if there is one', async () => {
      const configName = 'config.json';

      // File exists
      vi.mocked(fs.access).mockResolvedValueOnce();

      const randomModsJson = generateModsJson();
      const fileContents = JSON.stringify(randomModsJson.generated, null, 2);

      // Return config file contents
      vi.mocked(fs.readFile).mockResolvedValueOnce(fileContents);

      const actualOutput = await ensureConfiguration(configName, logger);

      expect(actualOutput).toEqual(randomModsJson.expected);
    });
  });
});
