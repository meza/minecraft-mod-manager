import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { generateInitializeOptions } from '../../test/initializeOptionsGenerator.js';
import { initializeConfig, InitializeOptions } from './initializeConfig.js';
import inquirer from 'inquirer';
import { writeConfigFile } from '../lib/config.js';
import * as path from 'path';
import { chance } from 'jest-chance';
import { verifyMinecraftVersion } from '../lib/minecraftVersionVerifier.js';
import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';

vi.mock('../lib/minecraftVersionVerifier.js');
vi.mock('../lib/config.js', () => ({
  fileExists: vi.fn().mockResolvedValue(false),
  writeConfigFile: vi.fn()
}));
vi.mock('inquirer');

describe('The Initialization Interaction', () => {
  beforeEach(() => {
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
    describe('when the correct version is supplied', () => {
      it('it should be successfully verified from the command line', async () => {
        vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(true);
        vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
        const inputOptions = generateInitializeOptions();

        await expect(initializeConfig(inputOptions.generated, chance.word())).resolves.not.toThrow();

      });
    });

    describe('when the incorrect version is supplied', async () => {
      it('it should throw an error', async () => {
        vi.mocked(verifyMinecraftVersion).mockResolvedValueOnce(false);
        vi.mocked(inquirer.prompt).mockResolvedValueOnce({});
        const inputOptions = generateInitializeOptions();

        await expect(initializeConfig(inputOptions.generated, chance.word())).rejects.toThrow(
          new IncorrectMinecraftVersionException(inputOptions.expected.gameVersion!)
        );
      });
    });
  });

});
