import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateInitializeOptions } from '../../test/initializeOptionsGenerator.js';
import { initializeConfig, InitializeOptions } from './initializeConfig.js';
import inquirer from 'inquirer';
import { fileExists, writeConfigFile } from '../lib/config.js';
import { chance } from 'jest-chance';
import { verifyMinecraftVersion } from '../lib/minecraftVersionVerifier.js';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import * as path from 'path';
import { configFile } from './configFileOverwrite.js';
import { Loader, ReleaseType } from '../lib/modlist.types.js';
import { findQuestion } from '../../test/inquirerHelper.js';
import { Logger } from '../lib/Logger.js';
import { getLatestMinecraftVersion } from './getLatestMinecraftVersion.js';

vi.mock('../lib/minecraftVersionVerifier.js');
vi.mock('./configFileOverwrite.js');
vi.mock('../lib/config.js', () => ({
  fileExists: vi.fn().mockResolvedValue(false),
  writeConfigFile: vi.fn()
}));
vi.mock('inquirer');
vi.mock('../lib/Logger.js');
vi.mock('./getLatestMinecraftVersion.js');

describe('The Initialization Interaction', () => {
  let logger: Logger;
  beforeEach(() => {
    vi.resetAllMocks();
    logger = new Logger({} as never);
    vi.mocked(fileExists).mockResolvedValue(true);
    vi.mocked(configFile).mockImplementation(async (input, cwd) => path.resolve(cwd, input.config));
    vi.mocked(getLatestMinecraftVersion).mockResolvedValue('0.0.0');
    vi.mocked(verifyMinecraftVersion).mockResolvedValue(true);
  });

  it('should use the submitted options', async () => {

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(true);
    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    const inputOptions = generateInitializeOptions();

    const actual = await initializeConfig(inputOptions.generated, 'x', logger);

    const transformedInput: Partial<InitializeOptions> = { ...inputOptions.expected };
    delete transformedInput.config;

    const expected = {
      ...transformedInput,
      mods: []
    };

    expect(actual).toEqual(expected);
    expect(vi.mocked(writeConfigFile)).toHaveBeenLastCalledWith(expected, inputOptions.generated, logger);

  });

  describe('and the game version is supplied', () => {

    describe('when the correct version is supplied', () => {
      it('it should be successfully verified from the command line', async () => {
        vi.mocked(verifyMinecraftVersion).mockReset();
        vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(true);
        vi.mocked(inquirer.prompt).mockResolvedValue({});
        const inputOptions = generateInitializeOptions();

        await expect(initializeConfig(inputOptions.generated, chance.word(), logger)).resolves.not.toThrow();

      });

      it('it should be successfully verified from the interactive ui', async () => {
        vi.mocked(verifyMinecraftVersion).mockReset();
        vi.mocked(verifyMinecraftVersion).mockResolvedValue(true);
        const input = generateInitializeOptions().generated;
        delete input.gameVersion;
        vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
        await initializeConfig(input, chance.word(), logger);

        const question = findQuestion(inquirer.prompt, 'gameVersion');

        expect(question.validate).toBeDefined();

        const verifierFunction = question.validate;
        const actual = await verifierFunction!(chance.word());

        expect(actual).toBeTruthy();

      });
    });

    describe('when the incorrect version is supplied', async () => {
      it('it should throw an error', async () => {
        vi.mocked(verifyMinecraftVersion).mockReset();
        vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(false);
        vi.mocked(inquirer.prompt).mockResolvedValue({});
        const inputOptions = generateInitializeOptions();

        await expect(initializeConfig(inputOptions.generated, chance.word(), logger)).rejects.toThrow(
          new IncorrectMinecraftVersionException(inputOptions.expected.gameVersion!)
        );
      });

      it('it should show an error message on the interactive ui', async () => {
        vi.mocked(verifyMinecraftVersion).mockReset();
        vi.mocked(verifyMinecraftVersion).mockResolvedValue(false);
        const input = generateInitializeOptions().generated;
        delete input.gameVersion;

        vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
        await initializeConfig(input, chance.word(), logger);

        const question = findQuestion(inquirer.prompt, 'gameVersion');

        expect(question.validate).toBeDefined();

        const verifierFunction = question.validate;
        const actual = await verifierFunction!(chance.word());

        expect(actual).toMatchInlineSnapshot('"The game version is invalid. Please enter a valid game version"');

      });
    });
  });

  describe('and the mods folder is supplied', () => {
    beforeEach(() => {
      vi.mocked(fileExists).mockReset();
    });
    describe('when an existing folder is given', () => {
      it('it should be successfully verified from the command line', async () => {
        const root = '/' + chance.word();
        const folder = chance.word();
        const location = path.resolve(root, folder);
        vi.mocked(inquirer.prompt).mockResolvedValue({});
        const inputOptions = generateInitializeOptions({
          modsFolder: folder
        });

        vi.mocked(fileExists).mockResolvedValue(true);

        await expect(initializeConfig(inputOptions.generated, root, logger)).resolves.not.toThrow();

        expect(fileExists).toHaveBeenCalledWith(location);

      });
      it('it should be verified from the interactive ui', async () => {
        const root = '/' + chance.word();
        const folder = chance.word();
        const modsLocation = path.resolve(root, folder);
        const input = generateInitializeOptions().generated;
        delete input.modsFolder;
        vi.mocked(inquirer.prompt).mockResolvedValueOnce({});

        vi.mocked(fileExists).mockResolvedValueOnce(true);

        await initializeConfig(input, root, logger);

        const question = findQuestion(inquirer.prompt, 'modsFolder');

        expect(question.validate).toBeDefined();

        const verifierFunction = question.validate;
        const actual = await verifierFunction!(folder);

        expect(vi.mocked(fileExists)).toHaveBeenCalledWith(modsLocation);
        expect(actual).toBeTruthy();
      });
    });
    describe('when a non existing folder is given', () => {
      it('it should show an error message on the command line', async () => {
        const root = chance.word();
        const folder = chance.word();
        const location = path.resolve(root, folder);
        vi.mocked(fileExists).mockResolvedValueOnce(false);
        vi.mocked(inquirer.prompt).mockResolvedValue({});
        const inputOptions = generateInitializeOptions({
          modsFolder: folder
        });

        await expect(initializeConfig(inputOptions.generated, root, logger)).rejects.toThrow(
          new Error(`The folder: ${location} does not exist. Please enter a valid one and try again.`)
        );
      });

      it('it should show an error message on the interactive ui', async () => {
        const root = '/root';
        const folder = 'test-folder';
        const modsLocation = path.resolve(root, folder);
        const input = generateInitializeOptions().generated;
        delete input.modsFolder;
        vi.mocked(inquirer.prompt).mockResolvedValueOnce({});

        vi.mocked(fileExists).mockResolvedValueOnce(false);

        await initializeConfig(input, root, logger);

        const question = findQuestion(inquirer.prompt, 'modsFolder');

        expect(question.validate).toBeDefined();

        const verifierFunction = question.validate;
        const actual = await verifierFunction!(folder);

        expect(vi.mocked(fileExists)).toHaveBeenCalledWith(modsLocation);
        const expectedPath = path.resolve(root, folder);
        expect(actual).toEqual(`The folder: ${expectedPath} does not exist. Please enter a valid one and try again.`);
      });
    });
  });

  it('asks for the loader when the loader isn\'t supplied', async () => {
    const input = generateInitializeOptions().generated;
    delete input.loader;

    const selectedLoader = chance.pickone(Object.values(Loader));

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({
      loader: selectedLoader
    });
    await initializeConfig(input, chance.word(), logger);

    expect(findQuestion(inquirer.prompt, 'loader')).toMatchInlineSnapshot(`
      {
        "choices": [
          "bukkit",
          "bungeecord",
          "cauldron",
          "datapack",
          "fabric",
          "folia",
          "forge",
          "liteloader",
          "modloader",
          "neoforge",
          "paper",
          "purpur",
          "quilt",
          "rift",
          "spigot",
          "sponge",
          "velocity",
          "waterfall",
        ],
        "message": "Which loader would you like to use?",
        "name": "loader",
        "type": "list",
        "when": true,
      }
    `);
  });

  it('skips the loader question when the loader is supplied', async () => {
    const input = generateInitializeOptions().generated;
    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    await initializeConfig(input, chance.word(), logger);

    expect(findQuestion(inquirer.prompt, 'loader').when).toBeFalsy();
  });

  it('asks for the release types when they aren\'t supplied', async () => {
    const input = generateInitializeOptions().generated;
    delete input.defaultAllowedReleaseTypes;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({
      defaultAllowedReleaseTypes: chance.pickset(Object.values(ReleaseType), { min: 1, max: 3 })
    });
    await initializeConfig(input, chance.word(), logger);

    expect(findQuestion(inquirer.prompt, 'defaultAllowedReleaseTypes')).toMatchInlineSnapshot(`
      {
        "choices": [
          "alpha",
          "beta",
          "release",
        ],
        "default": [
          "release",
          "beta",
        ],
        "message": "Which types of releases would you like to consider to download?",
        "name": "defaultAllowedReleaseTypes",
        "type": "checkbox",
        "when": true,
      }
    `);
  });

  it('skips the release types question when they are supplied', async () => {
    const input = generateInitializeOptions().generated;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    await initializeConfig(input, chance.word(), logger);

    expect(findQuestion(inquirer.prompt, 'defaultAllowedReleaseTypes').when).toBeFalsy();
  });

  it('asks for the game version when it isn\'t supplied', async () => {
    const input = generateInitializeOptions().generated;
    delete input.gameVersion;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({
      gameVersion: chance.word()
    });
    await initializeConfig(input, chance.word(), logger);

    expect(findQuestion(inquirer.prompt, 'gameVersion')).toMatchInlineSnapshot(`
      {
        "default": "0.0.0",
        "message": "What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1)",
        "name": "gameVersion",
        "type": "input",
        "validate": [Function],
        "validateText": "Verifying the game version",
        "when": true,
      }
    `);
  });

  it('skips the game version question when it is supplied', async () => {
    const input = generateInitializeOptions().generated;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    await initializeConfig(input, chance.word(), logger);

    expect(findQuestion(inquirer.prompt, 'gameVersion').when).toBeFalsy();
  });

  it('asks for the mods folder when it isn\'t supplied', async () => {
    const input = generateInitializeOptions().generated;
    delete input.modsFolder;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({
      gameVersion: chance.word()
    });

    await initializeConfig(input, '/root', logger);

    expect(findQuestion(inquirer.prompt, 'modsFolder')).toMatchInlineSnapshot(`
      {
        "default": "./mods",
        "message": "where is your mods folder? (full or relative path from /root):",
        "name": "modsFolder",
        "type": "input",
        "validate": [Function],
        "when": true,
      }
    `);
  });

  it('skips the mods folder question when it is supplied', async () => {
    const input = generateInitializeOptions().generated;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    await initializeConfig(input, chance.word(), logger);

    expect(findQuestion(inquirer.prompt, 'modsFolder').when).toBeFalsy();
  });

});
