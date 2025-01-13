import { confirm } from '@inquirer/prompts';
import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ScanOptions } from '../actions/scan.js';
import { Logger } from '../lib/Logger.js';
import { shouldAddScanResults } from './shouldAddScanResults.js';

vi.mock('@inquirer/prompts');
vi.mock('../lib/Logger.js');

interface LocalTestContext {
  logger: Logger;
}

describe('The should add scan results interaction', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();

    context.logger = new Logger({} as never);
  });

  it<LocalTestContext>('returns true if the add is already set', async ({ logger }) => {
    const actual = await shouldAddScanResults(
      {
        add: true
      } as ScanOptions,
      logger
    );

    expect(actual).toBeTruthy();
  });

  it<LocalTestContext>('should log out the help message when not in add mode but in quiet mode', async ({ logger }) => {
    const actual = await shouldAddScanResults(
      {
        add: false,
        quiet: true
      } as ScanOptions,
      logger
    );

    expect(actual).toBeFalsy();
    expect(vi.mocked(logger.log).mock.calls[0][0]).toMatchInlineSnapshot(`
      "
      Use the --add flag to add these mod to your modlist."
    `);
  });

  it<LocalTestContext>('should invoke inqurer properly when needed', async ({ logger }) => {
    const randomResponse = chance.bool();
    vi.mocked(confirm).mockResolvedValueOnce(randomResponse);

    const actual = await shouldAddScanResults(
      {
        add: false,
        quiet: false
      } as ScanOptions,
      logger
    );

    expect(vi.mocked(confirm).mock.calls[0]).toMatchInlineSnapshot(`
      [
        {
          "default": true,
          "message": "Do you want to add these mods and/or make changes to your config?",
        },
      ]
    `);
    expect(actual).toEqual(randomResponse);
  });
});
