import { chance } from 'jest-chance';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { add } from './actions/add.js';
import { install } from './actions/install.js';
import { list } from './actions/list.js';
import { testGameVersion } from './actions/testGameVersion.js';
import { update } from './actions/update.js';
import { initializeConfig } from './interactions/initializeConfig.js';
import { Logger } from './lib/Logger.js';
import { Platform } from './lib/modlist.types.js';

vi.mock('./lib/Logger.js');
vi.mock('./actions/add.js');
vi.mock('./actions/list.js');
vi.mock('./actions/install.js');
vi.mock('./actions/update.js');
vi.mock('./interactions/initializeConfig.js');
vi.mock('./actions/testGameVersion.js');

describe('The main CLI configuration', () => {
  let logger: Logger;
  beforeEach(() => {
    vi.resetModules();
    vi.spyOn(process, 'cwd').mockReturnValue('/path/to/minecraft/installation');
    logger = new Logger({} as never);
  });

  afterEach(() => {
    vi.resetAllMocks();
    vi.resetModules();
  });

  it('is set up correctly', async () => {
    const { program } = await import('./mmm.js');
    expect(program).toMatchSnapshot();
  });

  it('has add hooked up to the correct function', async () => {
    const { program } = await import('./mmm.js');
    vi.mocked(add).mockResolvedValueOnce();
    await program.parse([
      '',
      '',
      chance.pickone(['add', 'a']),
      chance.pickone(Object.values(Platform)),
      chance.word()]);
    expect(vi.mocked(add)).toHaveBeenCalledOnce();
  });

  it('has list hooked up to the correct function', async () => {
    const { program } = await import('./mmm.js');

    vi.mocked(list).mockResolvedValueOnce();
    await program.parse([
      '',
      '',
      chance.pickone(['list', 'ls'])
    ]);
    expect(vi.mocked(list)).toHaveBeenCalledOnce();
  });

  it('has install hooked up to the correct function', async () => {
    const { program } = await import('./mmm.js');

    vi.mocked(install).mockResolvedValueOnce();
    await program.parse([
      '',
      '',
      chance.pickone(['install', 'i'])
    ]);
    expect(vi.mocked(install)).toHaveBeenCalledOnce();
  });

  it('has update hooked up to the correct function', async () => {
    const { program } = await import('./mmm.js');

    vi.mocked(update).mockResolvedValueOnce();
    await program.parse([
      '',
      '',
      chance.pickone(['update', 'u'])
    ]);
    expect(vi.mocked(update)).toHaveBeenCalledOnce();
  });

  it('has initialize hooked up to the correct function', async () => {
    const { program } = await import('./mmm.js');

    vi.mocked(initializeConfig).mockResolvedValueOnce(expect.anything());
    await program.parse([
      '',
      '',
      chance.pickone(['init'])
    ]);
    expect(vi.mocked(initializeConfig)).toHaveBeenCalledOnce();
  });

  it('has the test hooked up to the correct function', async () => {
    const { program } = await import('./mmm.js');
    vi.mocked(testGameVersion).mockResolvedValueOnce(expect.anything());
    await program.parse([
      '',
      '',
      chance.pickone(['test', 't'])
    ]);
    expect(testGameVersion).toHaveBeenCalledOnce();
  });

  it('sets the logger to quiet when the quiet option is supplied', async () => {
    const { program } = await import('./mmm.js');
    await program.parse([
      '',
      '',
      chance.pickone(['-q', '--quiet']),
      chance.pickone(['init'])
    ]);
    expect(logger.flagQuiet).toHaveBeenCalledOnce();
  });

  it('sets the logger to debug when the debug option is supplied', async () => {
    const { program } = await import('./mmm.js');
    await program.parse([
      '',
      '',
      chance.pickone(['-d', '--debug']),
      chance.pickone(['init'])
    ]);
    expect(logger.flagDebug).toHaveBeenCalledOnce();
  });

  it('can stop the execution', async () => {
    vi.spyOn(process, 'exit').mockImplementation(() => {
      throw new Error('process.exit');
    });
    const { stop } = await import('./mmm.js');
    expect(stop).toThrow(new Error('process.exit'));
  });
});
