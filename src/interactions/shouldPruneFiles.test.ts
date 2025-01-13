import { confirm } from '@inquirer/prompts';
import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PruneOptions } from '../actions/prune.js';
import { Logger } from '../lib/Logger.js';
import { shouldPruneFiles } from './shouldPruneFiles.js';

vi.mock('@inquirer/prompts');
vi.mock('../lib/Logger.js');

interface LocalTestContext {
  logger: Logger;
}

describe('The should prune files interaction', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();

    context.logger = new Logger({} as never);
  });

  it<LocalTestContext>('returns true if the force is already set', async ({ logger }) => {
    const actual = await shouldPruneFiles(
      {
        force: true
      } as PruneOptions,
      logger
    );

    expect(actual).toBeTruthy();
  });

  it<LocalTestContext>('should log out the help message when not in force mode but in quiet mode', async ({
    logger
  }) => {
    const actual = await shouldPruneFiles(
      {
        force: false,
        quiet: true
      } as PruneOptions,
      logger
    );

    expect(actual).toBeFalsy();
    expect(vi.mocked(logger.log).mock.calls[0][0]).toMatchInlineSnapshot(
      '"There are files to prune but you are using --quiet."'
    );
    expect(vi.mocked(logger.log).mock.calls[1][0]).toMatchInlineSnapshot(
      '"Use mmm prune --quiet --force to prune all the files without any interaction"'
    );
  });

  it<LocalTestContext>('should invoke inqurer properly when needed', async ({ logger }) => {
    const randomResponse = chance.bool();
    vi.mocked(confirm).mockResolvedValueOnce(randomResponse);

    const actual = await shouldPruneFiles(
      {
        force: false,
        quiet: false
      } as PruneOptions,
      logger
    );

    expect(vi.mocked(confirm).mock.calls[0]).toMatchInlineSnapshot(`
      [
        {
          "default": true,
          "message": "Do you want to delete these files?",
        },
      ]
    `);
    expect(actual).toEqual(randomResponse);
  });
});
