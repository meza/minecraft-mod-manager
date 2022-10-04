import { afterEach, describe, expect, it, vi } from 'vitest';
import inquirer from 'inquirer';
import { shouldCreateConfig } from './shouldCreateConfig.js';
import { chance } from 'jest-chance';

vi.mock('inquirer');
describe('The should create config interaction', () => {
  afterEach(() => {
    vi.resetAllMocks();
  });

  it('invokes the expected interaction', async () => {
    vi.mocked(inquirer.prompt).mockResolvedValueOnce({ create: true });

    const configFile = chance.word() + '.json';

    await shouldCreateConfig(configFile);

    expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledWith([
      {
        default: true,
        message: `The config file: (${configFile}) does not exist. Should we create it?`,
        name: 'create',
        type: 'confirm'
      }
    ]);

    expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledOnce();

  });

  it.each([true, false])('returns the user\'s selection when it is %s', async (selection) => {
    vi.mocked(inquirer.prompt).mockResolvedValueOnce({ create: selection });
    const actual = await shouldCreateConfig(chance.word());

    expect(actual).toEqual(selection);

  });

});
