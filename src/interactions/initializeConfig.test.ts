import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { generateInitializeOptions } from '../../test/initializeOptionsGenerator.js';
import { initializeConfig, InitializeOptions } from './initializeConfig.js';
import inquirer from 'inquirer';
import { writeConfigFile } from '../lib/config.js';
import { chance } from 'jest-chance';
import { getLatestMinecraftVersion, verifyMinecraftVersion } from '../lib/minecraftVersionVerifier.js';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import * as path from 'path';
import { configFile } from './configFileOverwrite.js';
import { Loader, ReleaseType } from '../lib/modlist.types.js';
import { findQuestion } from '../../test/inquirerHelper.js';

vi.mock('../lib/minecraftVersionVerifier.js');
vi.mock('./configFileOverwrite.js');
vi.mock('../lib/config.js', () => ({
  fileExists: vi.fn().mockResolvedValue(false),
  writeConfigFile: vi.fn()
}));
vi.mock('inquirer');

describe('The Initialization Interaction', () => {
  beforeEach(() => {
    vi.mocked(configFile).mockImplementation(async (input, cwd) => path.resolve(cwd, input.config));
    vi.mocked(getLatestMinecraftVersion).mockResolvedValue('0.0.0');
    vi.mocked(verifyMinecraftVersion).mockResolvedValue(true);
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('should use the submitted options', async () => {

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(true);
    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    const inputOptions = generateInitializeOptions();

    const actual = await initializeConfig(inputOptions.generated, 'x');

    const transformedInput: Partial<InitializeOptions> = { ...inputOptions.expected };
    delete transformedInput.config;

    const expected = {
      ...transformedInput,
      mods: []
    };

    expect(actual).toEqual(expected);
    expect(vi.mocked(writeConfigFile)).toHaveBeenLastCalledWith(expected, path.resolve('x', inputOptions.expected.config));

  });

  describe('and the game version is supplied', () => {
    beforeEach(() => {
      vi.mocked(verifyMinecraftVersion).mockReset();
    });

    describe('when the correct version is supplied', () => {
      it('it should be successfully verified from the command line', async () => {
        vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(true);
        vi.mocked(inquirer.prompt).mockResolvedValue({});
        const inputOptions = generateInitializeOptions();

        await expect(initializeConfig(inputOptions.generated, chance.word())).resolves.not.toThrow();

      });
    });

    describe('when the incorrect version is supplied', async () => {
      it('it should throw an error', async () => {
        vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(false);
        vi.mocked(inquirer.prompt).mockResolvedValue({});
        const inputOptions = generateInitializeOptions();

        await expect(initializeConfig(inputOptions.generated, chance.word())).rejects.toThrow(
          new IncorrectMinecraftVersionException(inputOptions.expected.gameVersion!)
        );
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
    await initializeConfig(input, chance.word());

    expect(findQuestion(inquirer.prompt, 'loader')).toMatchInlineSnapshot(`
      {
        "choices": [
          "forge",
          "fabric",
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
    await initializeConfig(input, chance.word());

    expect(findQuestion(inquirer.prompt, 'loader').when).toBeFalsy();
  });

  it('asks for the version fallback when it isn\'t supplied', async () => {
    const input = generateInitializeOptions().generated;
    delete input.allowVersionFallback;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({
      allowVersionFallback: chance.bool()
    });
    await initializeConfig(input, chance.word());

    expect(findQuestion(inquirer.prompt, 'allowVersionFallback')).toMatchInlineSnapshot(`
      {
        "message": "Should we try to download mods for previous Minecraft versions if they do not exist for your Minecraft Version?",
        "name": "allowVersionFallback",
        "type": "confirm",
        "when": true,
      }
    `);
  });

  it('skips the version fallback question when it is supplied', async () => {
    const input = generateInitializeOptions().generated;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
    await initializeConfig(input, chance.word());

    expect(findQuestion(inquirer.prompt, 'allowVersionFallback').when).toBeFalsy();
  });

  it('asks for the release types when they aren\'t supplied', async () => {
    const input = generateInitializeOptions().generated;
    delete input.defaultAllowedReleaseTypes;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({
      defaultAllowedReleaseTypes: chance.pickset(Object.values(ReleaseType), { min: 1, max: 3 })
    });
    await initializeConfig(input, chance.word());

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
    await initializeConfig(input, chance.word());

    expect(findQuestion(inquirer.prompt, 'allowVersionFallback').when).toBeFalsy();
  });

});
