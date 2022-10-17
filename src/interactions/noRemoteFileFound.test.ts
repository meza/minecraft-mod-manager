import { afterEach, describe, expect, it, vi } from 'vitest';
import { Platform } from '../lib/modlist.types.js';
import { logger } from '../mmm.js';
import inquirer from 'inquirer';
import { chance } from 'jest-chance';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { noRemoteFileFound } from './noRemoteFileFound.js';

vi.mock('../mmm.js');
vi.mock('inquirer');

describe('The mod not found interaction', () => {
  afterEach(() => {
    vi.resetAllMocks();
  });

  it('shows the correct error message when in quiet mode', async () => {
    const testPlatform = Platform.CURSEFORGE;
    const testModId = 'test-mod-id';
    const randomConfig = generateModsJson().generated;

    await noRemoteFileFound(testModId, testPlatform, randomConfig, logger, { config: 'config.json', quiet: true });

    const loggerErrorCall = vi.mocked(logger.error).mock.calls[0][0];

    expect(loggerErrorCall).toMatchInlineSnapshot('"Could not find a file for test-mod-id and the Minecraft version cuodja for forge loader"');
    expect(vi.mocked(inquirer.prompt)).not.toHaveBeenCalled();

  });

  it('aborts when the user does not want to modify their search', async () => {
    const testPlatform = chance.pickone(Object.values(Platform));
    const testModId = chance.word();
    const randomConfig = generateModsJson().generated;

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({ confirm: false });

    await noRemoteFileFound(testModId, testPlatform, randomConfig, logger, { config: 'config.json', quiet: false });

    const loggerErrorCall = vi.mocked(logger.error).mock.calls[0][0];

    expect(loggerErrorCall).toMatchInlineSnapshot('"Aborting"');
    expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledTimes(1);

  });

  describe.each([
    { input: Platform.CURSEFORGE, expected: Platform.MODRINTH },
    { input: Platform.MODRINTH, expected: Platform.CURSEFORGE }
  ])('when the user wants to modify their search', ({ input, expected }) => {
    it(`it asks for ${expected} when they come from ${input}`, async () => {
      const testPlatform = input;
      const testModId = chance.word();
      const randomConfig = generateModsJson().generated;

      vi.mocked(inquirer.prompt).mockResolvedValueOnce({ confirm: true });
      vi.mocked(inquirer.prompt).mockResolvedValueOnce({ newModName: 'new-mod-id' });

      const actual = await noRemoteFileFound(testModId, testPlatform, randomConfig, logger, {
        config: 'config.json',
        quiet: false
      });

      const inquirerPromptCallArgs = vi.mocked(inquirer.prompt).mock.calls[1][0];

      expect(actual).toEqual({ id: 'new-mod-id', platform: expected });

      expect(inquirerPromptCallArgs).toMatchSnapshot();

    });
  });
});
