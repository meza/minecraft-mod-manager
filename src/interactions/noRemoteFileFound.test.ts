import { confirm, input } from '@inquirer/prompts';
import { chance } from 'jest-chance';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { generateRandomPlatform } from '../../test/generateRandomPlatform.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { Logger } from '../lib/Logger.js';
import { Loader, Platform } from '../lib/modlist.types.js';
import { noRemoteFileFound } from './noRemoteFileFound.js';

vi.mock('@inquirer/prompts');
vi.mock('../mmm.js');
vi.mock('../lib/Logger.js');

describe('The mod not found interaction', () => {
  let logger: Logger;

  beforeEach(() => {
    logger = new Logger({} as never);
    vi.mocked(logger.error).mockImplementation(() => {
      throw new Error('process.exit');
    });
  });
  afterEach(() => {
    vi.resetAllMocks();
  });

  it('shows the correct error message when in quiet mode', async () => {
    const testPlatform = Platform.CURSEFORGE;
    const testModId = 'test-mod-id';
    const randomConfig = generateModsJson({
      gameVersion: '1.16.5',
      loader: Loader.FORGE
    }).generated;

    await expect(
      noRemoteFileFound(testModId, testPlatform, randomConfig, logger, {
        config: 'config.json',
        quiet: true
      })
    ).rejects.toThrow(new Error('process.exit'));

    const loggerErrorCall = vi.mocked(logger.error).mock.calls[0][0];

    expect(loggerErrorCall).toMatchInlineSnapshot(
      '"Could not find a file for test-mod-id and the Minecraft version 1.16.5 for forge loader"'
    );
    expect(vi.mocked(confirm)).not.toHaveBeenCalled();
    expect(vi.mocked(input)).not.toHaveBeenCalled();
  });

  it('aborts when the user does not want to modify their search', async () => {
    const testPlatform = generateRandomPlatform();
    const testModId = chance.word();
    const randomConfig = generateModsJson().generated;

    vi.mocked(confirm).mockResolvedValueOnce(false);

    await expect(
      noRemoteFileFound(testModId, testPlatform, randomConfig, logger, {
        config: 'config.json',
        quiet: false
      })
    ).rejects.toThrow(new Error('process.exit'));

    const loggerErrorCall = vi.mocked(logger.error).mock.calls[0][0];

    expect(loggerErrorCall).toMatchInlineSnapshot('"Aborting"');
    expect(vi.mocked(confirm)).toHaveBeenCalledTimes(1);
    expect(vi.mocked(input)).not.toHaveBeenCalled();
  });

  describe.each([
    { userInput: Platform.CURSEFORGE, expected: Platform.MODRINTH },
    { userInput: Platform.MODRINTH, expected: Platform.CURSEFORGE }
  ])('when the user wants to modify their search for $input', ({ userInput, expected }) => {
    it(`it asks for ${expected} when they come from ${input}`, async () => {
      const testPlatform = userInput;
      const testModId = chance.word();
      const randomConfig = generateModsJson().generated;

      vi.mocked(confirm).mockResolvedValueOnce(true);
      vi.mocked(input).mockResolvedValueOnce('new-mod-id');

      const actual = await noRemoteFileFound(testModId, testPlatform, randomConfig, logger, {
        config: 'config.json',
        quiet: false
      });

      const inquirerPromptCallArgs = vi.mocked(input).mock.calls[0][0];

      expect(actual).toEqual({ id: 'new-mod-id', platform: expected });

      expect(inquirerPromptCallArgs).toMatchSnapshot();
    });
  });
});
