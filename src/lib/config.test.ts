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

vi.mock('../interactions/shouldCreateConfig.js');
vi.mock('../interactions/initializeConfig.js');
vi.mock('node:fs/promises');

describe('The config library', () => {
  let logger: Logger;
  beforeEach(() => {
    logger = new Logger({} as never);
    vi.resetAllMocks();
  });

  it('can determine that a file exists', async () => {
    vi.mocked(fs.access).mockResolvedValueOnce();
    expect(await fileExists(chance.word())).toBeTruthy();
  });

  it('can determine that a file does not exist', async () => {
    vi.mocked(fs.access).mockRejectedValueOnce(new Error());
    expect(await fileExists(chance.word())).toBeFalsy();
  });

  it('can write the config file', async () => {
    const config = {
      something: 'value'
    } as unknown as ModsJson;
    const configPath = path.resolve(chance.word());

    await writeConfigFile(config, configPath);

    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(
      configPath,
      '{\n'
      + '  "something": "value"\n'
      + '}'
    );
  });

  it('can write the lock file', async () => {
    const config = [{
      something: 'value'
    }] as unknown as ModInstall[];
    const configFileName = chance.word();
    const configPath = path.resolve(`${configFileName}.json`);
    const expectedLockFilePath = path.resolve(`${configFileName}-lock.json`);

    await writeLockFile(config, configPath);

    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(
      expectedLockFilePath,
      '[\n'
      + '  {\n'
      + '    "something": "value"\n'
      + '  }\n'
      + ']'
    );
  });

  it('can read the lock file when it exists', async () => {
    const configName = 'config.json';
    const lockfileName = 'config-lock.json';

    const randomModInstall = generateModInstall();

    const lockfileContents = JSON.stringify([randomModInstall.generated], null, 2);

    // Make file exist
    vi.mocked(fs.access).mockResolvedValueOnce();
    // Return file contents
    vi.mocked(fs.readFile).mockResolvedValueOnce(lockfileContents);

    const actualOutput = await readLockFile(configName);

    expect(actualOutput).toEqual([randomModInstall.expected]);

    expect(vi.mocked(fs.readFile)).toHaveBeenCalledWith(
      path.resolve(lockfileName),
      { encoding: 'utf8' }
    );

  });

  it('can returns an empty array and creates the file when the lock file does not exist', async () => {
    const configName = 'config.json';

    // Make file exist
    vi.mocked(fs.access).mockRejectedValueOnce(new Error());

    const actualOutput = await readLockFile(configName);

    expect(actualOutput).toEqual([]);

    expect(vi.mocked(fs.readFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(
      path.resolve('config-lock.json'),
      '[]'
    );

  });

  it('can read from the config file when it exists', async () => {
    const configName = 'config.json';

    // File exists
    vi.mocked(fs.access).mockResolvedValueOnce();

    const randomModsJson = generateModsJson();
    const fileContents = JSON.stringify(randomModsJson.generated, null, 2);

    // Return config file contents
    vi.mocked(fs.readFile).mockResolvedValueOnce(fileContents);

    const actualOutput = await readConfigFile(configName);

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
