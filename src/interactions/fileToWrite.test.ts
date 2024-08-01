import fs from 'node:fs/promises';
import path from 'path';
import inquirer, { Question, QuestionCollection } from 'inquirer';
import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Logger } from '../lib/Logger.js';
import { fileExists } from '../lib/config.js';
import { DefaultOptions } from '../mmm.js';
import { fileToWrite } from './fileToWrite.js';

vi.mock('node:fs/promises');
vi.mock('../lib/config.js');
vi.mock('../lib/Logger.js');
vi.mock('inquirer');

interface LocalTestContext {
  options: DefaultOptions;
  logger: Logger;
}

describe('The file writable module', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.options = {
      config: chance.word(),
      quiet: false,
      debug: false
    };

    vi.mocked(context.logger.error).mockImplementation(() => {
      throw new Error('process.exit');
    });
  });

  describe('when everything is fine', () => {
    beforeEach(() => {
      vi.mocked(fileExists).mockResolvedValue(true);
      vi.mocked(fs.access).mockResolvedValue(undefined);
    });

    it<LocalTestContext>('returns the input path', async ({ options, logger }) => {
      const inputPath = chance.word();

      const actual = await fileToWrite(inputPath, options, logger);

      expect(actual).toEqual(inputPath);
      expect(vi.mocked(fs.access)).toHaveBeenCalledWith(path.resolve(inputPath), fs.constants.W_OK);
    });

    it<LocalTestContext>('logs the path for debugging', async ({ options, logger }) => {
      const inputPath = 'the-input-file.txt';
      options.debug = true;
      await fileToWrite(inputPath, options, logger);

      const logMessage = vi.mocked(logger.debug).mock.calls[0][0];

      expect(logMessage).toMatchInlineSnapshot('"Checking if the-input-file.txt is writable"');
    });

    describe('when the file is not writable', () => {
      beforeEach(() => {
        vi.mocked(fileExists).mockResolvedValue(true);
        vi.mocked(fs.access).mockRejectedValue(new Error(''));
      });

      it<LocalTestContext>('it exits properly in quiet mode', async ({ options, logger }) => {
        const inputPath = '/a/path/to/modlist.json';
        options.quiet = true;
        await expect(fileToWrite(inputPath, options, logger)).rejects.toThrow('process.exit');

        const errorCall = vi.mocked(logger.error).mock.calls[0];
        const message = errorCall[0];
        const errorCode = errorCall[1];

        expect(message).toMatchInlineSnapshot('"/a/path/to/modlist.json is not writable. Aborting."');
        expect(errorCode).toEqual(1);
      });

      it<LocalTestContext>('handles the inquirer interaction as expected', async ({ options, logger }) => {
        const inputPath = '/a/path/to/modlist2.json';
        const suppliedPath = '/a/path/after/the/interaction.json';

        vi.mocked(inquirer.prompt).mockResolvedValueOnce({
          filePath: suppliedPath
        });

        const actual = await fileToWrite(inputPath, options, logger);

        expect(actual).toEqual(suppliedPath);

        const inquirerParams = vi.mocked(inquirer.prompt).mock.calls[0][0];

        expect(inquirerParams).toMatchInlineSnapshot(`
          [
            {
              "default": "/a/path/to/modlist2.json",
              "message": "/a/path/to/modlist2.json is not writable, please choose another one",
              "name": "filePath",
              "type": "input",
              "validate": [Function],
              "validationText": "Checking if file is writable",
            },
          ]
        `);
      });

      it<LocalTestContext>('successfully validates the user input', async ({ options, logger }) => {
        const inputPath = '/a/path/to/modlist2.json';
        const suppliedPath = '/a/path/after/the/interaction.json';

        vi.mocked(inquirer.prompt).mockResolvedValueOnce({
          filePath: suppliedPath
        });

        await fileToWrite(inputPath, options, logger);

        const inquirerParams = (vi.mocked(inquirer.prompt).mock.calls[0][0] as QuestionCollection[])[0] as Question;
        const validator = inquirerParams.validate;

        vi.mocked(fileExists).mockReset();
        vi.mocked(fs.access).mockReset();

        vi.mocked(fileExists).mockResolvedValueOnce(false);
        vi.mocked(fs.access).mockResolvedValueOnce(undefined);
        expect(validator).toBeDefined();
        if (validator) {
          const actual = await validator(inputPath);
          expect(actual).toBeTruthy();
        }
      });

      it<LocalTestContext>('reports the user input valiator error', async ({ options, logger }) => {
        const inputPath = '/a/path/to/modlist2.json';
        const suppliedPath = '/a/path/after/the/interaction.json';

        vi.mocked(inquirer.prompt).mockResolvedValueOnce({
          filePath: suppliedPath
        });

        await fileToWrite(inputPath, options, logger);

        const inquirerParams = (vi.mocked(inquirer.prompt).mock.calls[0][0] as QuestionCollection[])[0] as Question;
        const validator = inquirerParams.validate;

        vi.mocked(fileExists).mockReset();
        vi.mocked(fs.access).mockReset();

        vi.mocked(fileExists).mockResolvedValueOnce(false);
        vi.mocked(fs.access).mockRejectedValueOnce({});

        expect(validator).toBeDefined();
        if (validator) {
          const actual = await validator(inputPath);
          expect(actual).toMatchInlineSnapshot('"/a/path/to/modlist2.json is not writable, please choose another one"');
        }
      });
    });
  });
});
