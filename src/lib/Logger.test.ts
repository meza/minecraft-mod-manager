import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Logger } from './Logger.js';
import { Command } from 'commander';
import { chance } from 'jest-chance';

vi.mock('commander');

interface TestContext {
  program: Command,
  randomMessage: string,
}

describe('The logger', () => {

  beforeEach<TestContext>((context) => {
    context.program = new Command();
    context.randomMessage = chance.sentence();
  });

  it<TestContext>('should log to the console in normal mode', ({ program, randomMessage }) => {
    const logger = new Logger(program);
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {
    });

    logger.log(randomMessage);

    expect(logSpy).toHaveBeenCalledWith(randomMessage);
  });

  it<TestContext>('should not debug to the console in normal mode', ({ program, randomMessage }) => {
    const logger = new Logger(program);
    const debugSpy = vi.spyOn(console, 'debug').mockImplementation(() => {
    });

    logger.debug(randomMessage);

    expect(debugSpy).not.toHaveBeenCalled();
  });

  it<TestContext>('should log errors', ({ program, randomMessage }) => {
    const logger = new Logger(program);

    logger.error(randomMessage);

    expect(vi.mocked(program.error)).toHaveBeenCalledWith(randomMessage, { exitCode: 1 });
  });

  it<TestContext>('should pass exit codes on to the handler', ({ program, randomMessage }) => {
    const logger = new Logger(program);
    const randomCode = chance.natural({ min: 1, max: 4 });
    logger.error(randomMessage, randomCode);

    expect(vi.mocked(program.error)).toHaveBeenCalledWith(randomMessage, { exitCode: randomCode });
  });

  describe('when the quiet mode', () => {
    it<TestContext>('should not log to the console', ({ program, randomMessage }) => {
      const logger = new Logger(program);
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {
      });

      logger.flagQuiet();
      logger.log(randomMessage);

      expect(logSpy).not.toHaveBeenCalled();
    });

    it<TestContext>('should log to the console when forced', ({ program, randomMessage }) => {
      const logger = new Logger(program);
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {
      });

      logger.flagQuiet();
      logger.log(randomMessage, true);

      expect(logSpy).toHaveBeenCalledWith(randomMessage);
    });
  });

  describe('when in debug mode', () => {
    it<TestContext>('should debug to the console', ({ program, randomMessage }) => {
      const logger = new Logger(program);
      const debugSpy = vi.spyOn(console, 'debug').mockImplementation(() => {
      });

      logger.flagDebug();
      logger.debug(randomMessage);

      expect(debugSpy).toHaveBeenCalledWith(randomMessage);
    });

    it<TestContext>('should log to the console even in quiet mode', ({ program, randomMessage }) => {
      const logger = new Logger(program);
      const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {
      });

      logger.flagDebug();
      logger.flagQuiet();
      logger.log(randomMessage);

      expect(logSpy).toHaveBeenCalledWith(randomMessage);
    });
  });
});
