import path from 'path';
import { confirm, input } from '@inquirer/prompts';
import { chance } from 'jest-chance';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ConfigFileAlreadyExistsException } from '../errors/ConfigFileAlreadyExistsException.js';
import { fileExists } from '../lib/config.js';
import { configFile } from './configFileOverwrite.js';

vi.mock('@inquirer/prompts');
vi.mock('../lib/config.js', () => ({
  fileExists: vi.fn().mockResolvedValue(false),
  writeConfigFile: vi.fn()
}));

describe('The Config Overwrite Interaction', () => {
  afterEach(() => {
    vi.resetAllMocks();
  });

  describe('when the supplied config file already exists', () => {
    describe('and we are in non-interactive mode', () => {
      it('it should throw an error', async () => {
        vi.mocked(fileExists).mockResolvedValue(true);
        const inputOptions = { quiet: true, config: chance.word() };
        await expect(configFile(inputOptions, chance.word())).rejects.toEqual(
          new ConfigFileAlreadyExistsException(inputOptions.config)
        );
      });
    });

    describe('and we are in interactive mode', () => {
      describe('when the user wants to overwrite it', () => {
        it('it should use the submitted options', async () => {
          const root = '/' + chance.word();
          const filename = chance.word();
          const location = path.resolve(root, filename);

          vi.mocked(fileExists).mockResolvedValue(true);
          vi.mocked(confirm).mockResolvedValueOnce(true); // for the config question

          const inputOptions = {
            config: filename,
            quiet: false // explicitly
          };

          const actual = await configFile(inputOptions, root);

          expect(actual).toEqual(location);
        });
      });

      describe('when the user does not want to overwrite it', () => {
        it('it should use the submitted options', async () => {
          const root = '/' + chance.word();
          const filename = chance.word();
          const newFilename = chance.word();
          const newLocation = path.resolve(root, newFilename);

          vi.mocked(fileExists).mockResolvedValue(true);
          vi.mocked(confirm).mockResolvedValueOnce(false); // for the config question
          vi.mocked(input).mockResolvedValueOnce(newFilename); // for the config question

          const inputOptions = {
            config: filename,
            quiet: false // explicitly
          };

          const actual = await configFile(inputOptions, root);

          expect(actual).toEqual(newLocation);
        });

        it('it identifies existing files when it validates the user input correctly', async () => {
          vi.mocked(fileExists).mockResolvedValue(true);
          vi.mocked(confirm).mockResolvedValueOnce(false); // for the config question
          vi.mocked(input).mockResolvedValueOnce(chance.word()); // for the config question

          const inputOptions = { config: chance.word() };

          await configFile(inputOptions, chance.word());

          // The only way to ensure the correct validator function has been wired up is to run it
          // We grab the actual submitted callback from inquirer
          const validatorFunction = vi.mocked(input).mock.calls[0][0].validate;

          const filename = chance.word();

          vi.mocked(fileExists).mockResolvedValueOnce(true);

          expect(validatorFunction).toBeDefined();
          if (validatorFunction) {
            const actual = await validatorFunction(filename);

            expect(actual).toEqual('The config file already exists. Please choose a different name');
          }
        });

        it('it identifies non existing files when it validates the user input correctly', async () => {
          vi.mocked(fileExists).mockResolvedValue(true);
          vi.mocked(confirm).mockResolvedValueOnce(false); // for the config question
          vi.mocked(input).mockResolvedValueOnce(chance.word()); // for the config question

          const inputOptions = { config: chance.word() };

          await configFile(inputOptions, chance.word());

          // The only way to ensure the correct validator function has been wired up is to run it
          // We grab the actual submitted callback from inquirer
          const validatorFunction = vi.mocked(input).mock.calls[0][0].validate;

          const filename = chance.word();

          vi.mocked(fileExists).mockResolvedValueOnce(false);

          expect(validatorFunction).toBeDefined();
          if (validatorFunction) {
            const actual = await validatorFunction(filename);

            expect(actual).toEqual(true);
          }
        });
      });
    });
  });

  it('should show the correct messages', async () => {
    const originalFile = 'test.json';
    vi.mocked(fileExists).mockResolvedValue(true);
    vi.mocked(confirm).mockResolvedValueOnce(false); // for the config question
    vi.mocked(input).mockResolvedValueOnce(chance.word()); // for the config question

    const inputOptions = { config: originalFile };

    await configFile(inputOptions, chance.word());

    expect(vi.mocked(confirm).mock.calls[0][0].message).toMatchInlineSnapshot(
      '"The config file test.json already exists. Would you like to overwrite it?"'
    );
    expect(vi.mocked(input).mock.calls[0][0].message).toMatchInlineSnapshot('"Please enter a new config file name"');
  });

  it('should show the correct default for the new file', async () => {
    const filename = 'test2.json';
    const expectedFileName = 'test2-new.json';

    vi.mocked(fileExists).mockResolvedValue(true);
    vi.mocked(confirm).mockResolvedValueOnce(false); // for the config question
    vi.mocked(input).mockResolvedValueOnce(chance.word()); // for the config question

    const inputOptions = { config: filename };

    await configFile(inputOptions, chance.word());

    expect(vi.mocked(input).mock.calls[0][0].default).toEqual(expectedFileName);
  });
});
