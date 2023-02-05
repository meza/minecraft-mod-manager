import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Logger } from '../lib/Logger.js';
import { Mod } from '../lib/modlist.types.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { CouldNotFindModException } from './CouldNotFindModException.js';
import { handleFetchErrors } from './handleFetchErrors.js';
import { NoRemoteFileFound } from './NoRemoteFileFound.js';
import { DownloadFailedException } from './DownloadFailedException.js';
import { chance } from 'jest-chance';

interface LocalTestContext {
  logger: Logger;
  randomMod: Mod;
}

vi.mock('../lib/Logger.js');

describe('The mod fetch error handler', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.randomMod = generateModConfig().generated;

    vi.mocked(context.logger.error).mockImplementation(() => {
      throw new Error('process.exit');
    });
  });

  it<LocalTestContext>('handles when the mod cannot be found', ({ logger, randomMod }) => {
    const error = new CouldNotFindModException(randomMod.id, randomMod.type);
    handleFetchErrors(error, randomMod, logger);

    const logCall = vi.mocked(logger.log).mock.calls[0];
    const logMessage = logCall[0];
    expect(logMessage).toContain(randomMod.name);
    expect(logMessage).toContain(randomMod.id);
    expect(logMessage).toContain(randomMod.type);
    expect(logMessage).toContain('cannot be found on');
    expect(logMessage).toContain('anymore. Was the mod revoked?');

    expect(logCall[1]).toBeTruthy();
  });

  it<LocalTestContext>('handles when no remote files are found', ({ logger, randomMod }) => {
    const error = new NoRemoteFileFound(randomMod.name, randomMod.type);
    handleFetchErrors(error, randomMod, logger);

    const logCall = vi.mocked(logger.log).mock.calls[0];
    const logMessage = logCall[0];
    expect(logMessage).toContain(randomMod.name);
    expect(logMessage).toContain(randomMod.id);
    expect(logMessage).toContain(randomMod.type);
    expect(logMessage).toContain('doesn\'t serve the required file for');
    expect(logMessage).toContain('anymore. Please update it.');

    expect(logCall[1]).toBeTruthy();
  });

  it<LocalTestContext>('handles when the download fails', ({ logger, randomMod }) => {
    const url = chance.url({ protocol: 'http' });
    const error = new DownloadFailedException(url);
    expect(() => {
      handleFetchErrors(error, randomMod, logger);
    }).toThrow('process.exit');

    const logCall = vi.mocked(logger.error).mock.calls[0];
    const logMessage = logCall[0];
    expect(logMessage).toEqual(error.message);
    expect(logCall[1]).toEqual(1);
  });

  it<LocalTestContext>('passes on all other errors', ({ randomMod, logger }) => {
    const errorMsg = chance.word();
    const error = new Error(errorMsg);
    expect(() => {
      handleFetchErrors(error, randomMod, logger);
    }).toThrow(error);
  });

});
