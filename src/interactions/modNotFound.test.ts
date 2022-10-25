import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Platform } from '../lib/modlist.types.js';
import { modNotFound } from './modNotFound.js';

import inquirer from 'inquirer';
import { chance } from 'jest-chance';
import { Logger } from '../lib/Logger.js';

vi.mock('../mmm.js');
vi.mock('inquirer');
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

    await expect(modNotFound(testModId, testPlatform, logger, {
      config: 'config.json',
      quiet: true
    })).rejects.toThrow(new Error('process.exit'));

    const loggerErrorCall = vi.mocked(logger.error).mock.calls[0][0];

    expect(loggerErrorCall).toMatchInlineSnapshot('"Mod \\"test-mod-id\\" for curseforge does not exist"');
    expect(vi.mocked(inquirer.prompt)).not.toHaveBeenCalled();

  });

  it('aborts when the user does not want to modify their search', async () => {
    const testPlatform = chance.pickone(Object.values(Platform));
    const testModId = chance.word();

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({ confirm: false });

    await expect(modNotFound(testModId, testPlatform, logger, {
      config: 'config.json',
      quiet: false
    })).rejects.toThrow(new Error('process.exit'));

    const loggerErrorCall = vi.mocked(logger.error).mock.calls[0][0];

    expect(loggerErrorCall).toMatchInlineSnapshot('"Aborting"');
    expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledTimes(1);

  });

  it('asks the user for a new mod id and platform when the user wants to modify their search', async () => {
    const testPlatform = chance.pickone(Object.values(Platform));
    const testModId = chance.word();

    vi.mocked(inquirer.prompt).mockResolvedValueOnce({ confirm: true });
    vi.mocked(inquirer.prompt).mockResolvedValueOnce({ newPlatform: Platform.CURSEFORGE, newModName: 'new-mod-id' });

    const actual = await modNotFound(testModId, testPlatform, logger, { config: 'config.json', quiet: false });

    const inquirerPromptCallArgs = vi.mocked(inquirer.prompt).mock.calls[1][0];

    expect(actual).toEqual({ id: 'new-mod-id', platform: Platform.CURSEFORGE });

    expect(inquirerPromptCallArgs).toMatchInlineSnapshot(`
      [
        {
          "choices": [
            "curseforge",
            "modrinth",
          ],
          "default": "curseforge",
          "message": "Which platform would you like to use?",
          "name": "newPlatform",
          "type": "list",
        },
        {
          "message": "What is the project id of the mod you want to add?",
          "name": "newModName",
          "type": "input",
        },
      ]
    `);

  });
});
