import { afterEach, describe, expect, it, vi } from 'vitest';
import { add } from './actions/add.js';
import { program } from './mmm.js';
import { chance } from 'jest-chance';
import { Platform } from './lib/modlist.types.js';
import { list } from './actions/list.js';
import { install } from './actions/install.js';
import { update } from './actions/update.js';
import { initializeConfig } from './interactions/initializeConfig.js';

vi.mock('./actions/add.js');
vi.mock('./actions/list.js');
vi.mock('./actions/install.js');
vi.mock('./actions/update.js');
vi.mock('./interactions/initializeConfig.js');

describe('The main CLI configuration', () => {

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('is set up correctly', async () => {
    expect(program).toMatchSnapshot();
  });

  it('has add hooked up to the correct function', async () => {
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
    vi.mocked(list).mockResolvedValueOnce();
    await program.parse([
      '',
      '',
      chance.pickone(['list', 'ls'])
    ]);
    expect(vi.mocked(list)).toHaveBeenCalledOnce();
  });

  it('has install hooked up to the correct function', async () => {
    vi.mocked(install).mockResolvedValueOnce();
    await program.parse([
      '',
      '',
      chance.pickone(['install', 'i'])
    ]);
    expect(vi.mocked(install)).toHaveBeenCalledOnce();
  });

  it('has update hooked up to the correct function', async () => {
    vi.mocked(update).mockResolvedValueOnce();
    await program.parse([
      '',
      '',
      chance.pickone(['update', 'u'])
    ]);
    expect(vi.mocked(update)).toHaveBeenCalledOnce();
  });

  it('has initialize hooked up to the correct function', async () => {
    vi.mocked(initializeConfig).mockResolvedValueOnce(expect.anything());
    await program.parse([
      '',
      '',
      chance.pickone(['init'])
    ]);
    expect(vi.mocked(initializeConfig)).toHaveBeenCalledOnce();
  });
});
