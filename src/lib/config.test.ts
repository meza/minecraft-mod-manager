import fs from 'node:fs/promises';
import path from 'node:path';
import { chance } from 'jest-chance';
import * as process from 'process';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { ConfigFileNotFoundException } from '../errors/ConfigFileNotFoundException.js';
import { fileToWrite } from '../interactions/fileToWrite.js';
import { initializeConfig } from '../interactions/initializeConfig.js';
import { shouldCreateConfig } from '../interactions/shouldCreateConfig.js';
import { DefaultOptions } from '../mmm.js';
import { Logger } from './Logger.js';
import {
  ModsJsonSchema,
  ensureConfiguration,
  fileExists,
  getModsFolder,
  initializeConfigFile,
  readConfigFile,
  readLockFile,
  writeConfigFile,
  writeLockFile
} from './config.js';
import { Loader, ModInstall, ModsJson, Platform, ReleaseType } from './modlist.types.js';

vi.mock('../interactions/shouldCreateConfig.js');
vi.mock('../interactions/initializeConfig.js');
vi.mock('node:fs/promises');
vi.mock('../interactions/fileToWrite.js');
vi.mock('../lib/Logger.js');

interface LocalTestContext {
  options: DefaultOptions;
}

describe('The config library', () => {
  let logger: Logger;
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    vi.unstubAllGlobals();
    logger = new Logger({} as never);
    context.options = {
      config: 'config.json',
      debug: false,
      quiet: false
    };
    vi.mocked(fileToWrite).mockImplementation(async (file: string) => {
      return file;
    });
    vi.mocked(logger.error).mockImplementation((message) => {
      throw new Error(message);
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

  it<LocalTestContext>('can write the config file', async ({ options }) => {
    const config = {
      something: 'value'
    } as unknown as ModsJson;

    await writeConfigFile(config, options, logger);

    expect(vi.mocked(fileToWrite)).toHaveBeenCalledWith(path.resolve(options.config), options, logger);

    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(
      path.resolve(options.config),
      '{\n' + '  "something": "value"\n' + '}'
    );
  });

  it<LocalTestContext>('can write the lock file', async ({ options }) => {
    const config = [
      {
        something: 'value'
      }
    ] as unknown as ModInstall[];
    options.config = chance.word();
    const expectedLockFilePath = path.resolve(`${options.config}-lock.json`);

    await writeLockFile(config, options, logger);
    expect(vi.mocked(fileToWrite)).toHaveBeenCalledWith(expectedLockFilePath, options, logger);

    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(
      expectedLockFilePath,
      '[\n' + '  {\n' + '    "something": "value"\n' + '  }\n' + ']'
    );
  });

  it<LocalTestContext>('can read the lock file when it exists', async ({ options }) => {
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

    expect(vi.mocked(fs.readFile)).toHaveBeenCalledWith(path.resolve(lockfileName), { encoding: 'utf8' });
  });

  it<LocalTestContext>('can returns an empty array and creates the file when the lock file does not exist', async ({
    options
  }) => {
    options.config = 'config.json';

    // Make file exist
    vi.mocked(fs.access).mockRejectedValueOnce(new Error());

    const actualOutput = await readLockFile(options, logger);

    expect(actualOutput).toEqual([]);

    expect(vi.mocked(fs.readFile)).not.toHaveBeenCalled();
    expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith(path.resolve('config-lock.json'), '[]');
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

  it('throws an error when there are misconfigurations', async () => {
    const configName = 'config.json';
    const invalidModsJson = {
      loader: 'invalid_loader',
      gameVersion: '1.16.5',
      defaultAllowedReleaseTypes: ['invalid_release_type'],
      modsFolder: 'mods',
      mods: [
        {
          id: 'mod1',
          platform: 'invalid_platform',
          version: '1.0.0',
          allowVersionFallback: true,
          allowedReleaseTypes: ['invalid_release_type']
        }
      ]
    };

    const fileContents = JSON.stringify(invalidModsJson, null, 2);
    vi.mocked(fs.access).mockResolvedValueOnce();
    // Return config file contents
    vi.mocked(fs.readFile).mockResolvedValueOnce(fileContents);
    await expect(ensureConfiguration(configName, logger)).rejects.toThrowErrorMatchingSnapshot();
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

  it('can resolve a relative mod folder', () => {
    const randomModsJson = generateModsJson().generated;
    const configPath = path.resolve('/some-path/config.json');
    const expected = path.resolve('/some-path/mods');
    randomModsJson.modsFolder = 'mods';
    const actual = getModsFolder(configPath, randomModsJson);

    expect(actual).toEqual(expected);
  });

  it('can resolve an absolute mod folder', () => {
    const randomModsJson = generateModsJson().generated;
    const configPath = '/some-path/config.json';
    const modsFolder = '/my-ultimate-mods';
    randomModsJson.modsFolder = modsFolder;
    const expected = '/my-ultimate-mods';
    const actual = getModsFolder(configPath, randomModsJson);

    expect(actual).toEqual(expected);
  });

  it('should validate a correct ModsJson object', () => {
    const validModsJson: ModsJson = {
      loader: Loader.FORGE,
      gameVersion: '1.16.5',
      defaultAllowedReleaseTypes: [ReleaseType.RELEASE, ReleaseType.BETA],
      modsFolder: 'mods',
      mods: [
        {
          id: 'mod1',
          type: Platform.CURSEFORGE,
          name: 'Mod 1',
          version: '1.0.0',
          allowVersionFallback: true,
          allowedReleaseTypes: [ReleaseType.RELEASE]
        }
      ]
    };

    const result = ModsJsonSchema.safeParse(validModsJson);
    expect(result.success).toBe(true);
  });

  it('should invalidate an incorrect ModsJson object', () => {
    const invalidModsJson = {
      loader: 'invalid_loader',
      gameVersion: '1.16.5',
      defaultAllowedReleaseTypes: ['invalid_release_type'],
      modsFolder: 'mods',
      mods: [
        {
          id: 'mod1',
          platform: 'invalid_platform',
          version: '1.0.0',
          allowVersionFallback: true,
          allowedReleaseTypes: ['invalid_release_type']
        }
      ]
    };

    const result = ModsJsonSchema.safeParse(invalidModsJson);
    expect(result.success).toBe(false);
  });
});
