import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Platform } from '../lib/modlist.types.js';
import { modNotFound } from './modNotFound.js';

import { confirm, input, select } from '@inquirer/prompts';
import { chance } from 'jest-chance';
import { generateRandomPlatform } from '../../test/generateRandomPlatform.js';
import { Logger } from '../lib/Logger.js';

vi.mock('../mmm.js');
vi.mock('@inquirer/prompts');
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

    await expect(
      modNotFound(testModId, testPlatform, logger, {
        config: 'config.json',
        quiet: true
      })
    ).rejects.toThrow(new Error('process.exit'));

    const loggerErrorCall = vi.mocked(logger.error).mock.calls[0][0];

    expect(loggerErrorCall).toMatchInlineSnapshot('"Mod "test-mod-id" for curseforge does not exist"');
    expect(vi.mocked(input)).not.toHaveBeenCalled();
    expect(vi.mocked(confirm)).not.toHaveBeenCalled();
    expect(vi.mocked(select)).not.toHaveBeenCalled();
  });

  it('aborts when the user does not want to modify their search', async () => {
    const testPlatform = generateRandomPlatform();
    const testModId = chance.word();

    vi.mocked(confirm).mockResolvedValueOnce(false);

    await expect(
      modNotFound(testModId, testPlatform, logger, {
        config: 'config.json',
        quiet: false
      })
    ).rejects.toThrow(new Error('process.exit'));

    const loggerErrorCall = vi.mocked(logger.error).mock.calls[0][0];

    expect(loggerErrorCall).toMatchInlineSnapshot('"Aborting"');
    expect(vi.mocked(confirm)).toHaveBeenCalledTimes(1);
    expect(vi.mocked(input)).not.toHaveBeenCalled();
    expect(vi.mocked(select)).not.toHaveBeenCalled();
  });

  it('asks the user for a new mod id and platform when the user wants to modify their search', async () => {
    const testPlatform = Platform.MODRINTH;
    const testModId = chance.word();

    vi.mocked(confirm).mockResolvedValueOnce(true);
    vi.mocked(select).mockResolvedValueOnce(Platform.CURSEFORGE);
    vi.mocked(input).mockResolvedValueOnce('new-mod-id');

    const actual = await modNotFound(testModId, testPlatform, logger, { config: 'config.json', quiet: false });

    expect(actual).toEqual({ id: 'new-mod-id', platform: Platform.CURSEFORGE });

    const selectArgs = vi.mocked(select).mock.calls[0][0];
    const inputArgs = vi.mocked(input).mock.calls[0][0];

    expect(selectArgs).toMatchInlineSnapshot(`
      {
        "choices": [
          "curseforge",
          "modrinth",
        ],
        "default": "modrinth",
        "message": "Which platform would you like to use?",
      }
    `);
    expect(inputArgs).toMatchInlineSnapshot(`
      {
        "message": "What is the project id of the mod you want to add?",
      }
    `);
  });
});
