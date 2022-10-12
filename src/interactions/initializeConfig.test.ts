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

});
