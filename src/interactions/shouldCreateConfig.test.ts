import { confirm } from '@inquirer/prompts';
import { chance } from 'jest-chance';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { shouldCreateConfig } from './shouldCreateConfig.js';

vi.mock('@inquirer/prompts');
describe('The should create config interaction', () => {
  afterEach(() => {
    vi.resetAllMocks();
  });

  it('invokes the expected interaction', async () => {
    vi.mocked(confirm).mockResolvedValueOnce(true);

    const configFile = chance.word() + '.json';

    await shouldCreateConfig(configFile);

    expect(vi.mocked(confirm)).toHaveBeenCalledWith({
      default: true,
      message: `The config file: (${configFile}) does not exist. Should we create it?`
    });

    expect(vi.mocked(confirm)).toHaveBeenCalledOnce();
  });

  it.each([true, false])("returns the user's selection when it is %s", async (selection) => {
    vi.mocked(confirm).mockResolvedValueOnce(selection);
    const actual = await shouldCreateConfig(chance.word());

    expect(actual).toEqual(selection);
  });
});
