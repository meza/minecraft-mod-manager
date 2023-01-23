import { beforeEach, describe, it, vi, expect } from 'vitest';
import { Modrinth } from './index.js';
import { Loader, ReleaseType } from '../../lib/modlist.types.js';
import { chance } from 'jest-chance';
import { getMod } from './fetch.js';
import { generateRemoteModDetails } from '../../../test/generateRemoteDetails.js';
import { generatePlatformLookupResult } from '../../../test/generatePlatformLookupResult.js';
import { lookup as cfLookup } from './lookup.js';

vi.mock('./fetch.js');
vi.mock('./lookup.js');

describe('The Modrinth Repository class', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('has the correct api headers', async () => {
    expect(Modrinth.API_HEADERS).toMatchInlineSnapshot(`
      {
        "Accept": "application/json",
        "Authorization": "REPL_MODRINTH_API_KEY",
        "user-agent": "github_com/meza/minecraft-mod-manager/DEV",
      }
    `);
  });

  it('calls through to the fetching module', async () => {
    const projectId = chance.word();
    const allowedReleaseTypes = [chance.pickone(Object.values(ReleaseType))];
    const allowedGameVersion = chance.word();
    const loader = chance.pickone(Object.values(Loader));
    const allowFallback = chance.bool();
    const result = generateRemoteModDetails().generated;
    vi.mocked(getMod).mockResolvedValueOnce(result);

    const modrinth = new Modrinth();
    const actual = await modrinth.fetchMod(
      projectId,
      allowedReleaseTypes,
      allowedGameVersion,
      loader,
      allowFallback
    );

    expect(actual).toEqual(result);
    expect(vi.mocked(getMod)).toHaveBeenCalledOnce();
    expect(vi.mocked(getMod)).toHaveBeenCalledWith(
      projectId,
      allowedReleaseTypes,
      allowedGameVersion,
      loader,
      allowFallback
    );
  });

  it('calls through to the lookup module', async () => {
    const result = [generatePlatformLookupResult().generated];
    const lookupInput = chance.n(chance.word, chance.integer({ min: 1, max: 20 }));

    vi.mocked(cfLookup).mockResolvedValueOnce(result);
    const modrinth = new Modrinth();
    const actual = await modrinth.lookup(lookupInput);

    expect(vi.mocked(cfLookup)).toHaveBeenCalledWith(lookupInput);
    expect(actual).toBe(result);

  });
});
