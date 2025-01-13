import { input } from '@inquirer/prompts';
import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Logger } from '../lib/Logger.js';
import { getLatestMinecraftVersion as getLatestMinecraftVersionLib } from '../lib/minecraftVersionVerifier.js';
import { DefaultOptions } from '../mmm.js';
import { getLatestMinecraftVersion } from './getLatestMinecraftVersion.js';

vi.mock('../lib/minecraftVersionVerifier.js');
vi.mock('../lib/Logger.js');
vi.mock('@inquirer/prompts');

interface LocalTestContext {
  logger: Logger;
  options: DefaultOptions;
}

describe('The latest Minecraft Version Interaction', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.options = {
      quiet: false,
      debug: false,
      config: chance.word()
    };

    vi.mocked(context.logger.error).mockImplementation(() => {
      throw new Error('process.exit');
    });
  });

  describe('when everything is fine', () => {
    it<LocalTestContext>('returns whatever the library returns, no questions asked', async ({ options, logger }) => {
      const version = chance.word();
      vi.mocked(getLatestMinecraftVersionLib).mockResolvedValueOnce(version);

      const actual = await getLatestMinecraftVersion(options, logger);
      expect(actual).toEqual(version);
    });
  });

  describe('when in quiet mode', () => {
    it<LocalTestContext>('exits with the appropriate error', async ({ options, logger }) => {
      const error = chance.word();
      vi.mocked(getLatestMinecraftVersionLib).mockRejectedValueOnce(error);

      options.quiet = true;
      await expect(getLatestMinecraftVersion(options, logger)).rejects.toThrow('process.exit');

      expect(logger.error).toHaveBeenCalledOnce();

      const errorMessage = vi.mocked(logger.error).mock.calls[0][0];
      const exitCode = vi.mocked(logger.error).mock.calls[0][1];

      expect(exitCode).toEqual(1);
      expect(errorMessage).toMatchInlineSnapshot(
        '"The Minecraft APIs are down and the latest minecraft version could not be determined."'
      );
    });
  });

  describe('when in interactive mode', () => {
    it<LocalTestContext>('asks for the latest version appropriately', async ({ options, logger }) => {
      const error = chance.word();
      const version = chance.word();
      vi.mocked(getLatestMinecraftVersionLib).mockRejectedValueOnce(error);
      vi.mocked(input).mockResolvedValueOnce(version);

      const actual = await getLatestMinecraftVersion(options, logger);

      expect(vi.mocked(input).mock.calls[0][0]).toMatchInlineSnapshot(`
        {
          "message": "The Minecraft APIs are down. What is the latest Minecraft version? (for example: 1.19.3, 1.20)",
        }
      `);

      expect(actual).toEqual(version);
    });
  });
});
