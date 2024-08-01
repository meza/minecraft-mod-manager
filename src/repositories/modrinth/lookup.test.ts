import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { generateModrinthFile } from '../../../test/generateModrinthFile.js';
import { generateModrinthVersion } from '../../../test/generateModrinthVersion.js';
import { Logger } from '../../lib/Logger.js';
import { Platform } from '../../lib/modlist.types.js';
import { rateLimitingFetch } from '../../lib/rateLimiter/index.js';
import { Hash } from './fetch.js';
import { Modrinth } from './index.js';
import { lookup } from './lookup.js';

vi.mock('../../lib/rateLimiter/index.js');
vi.mock('../../lib/Logger.js');
vi.mock('./fetch.js');

interface LocalTestContext {
  logger: Logger;
}

describe('The Curseforge Lookup module', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
  });

  it<LocalTestContext>('correctly calls the modrinth api', async () => {
    vi.mocked(rateLimitingFetch).mockResolvedValue({
      ok: false // fastest way to exit out of the function under test
    } as unknown as Response);

    const actual = await lookup(['fingerprint1', 'fingerprint2', 'fingerprint3']);

    expect(rateLimitingFetch).toHaveBeenNthCalledWith(
      1,
      'https://api.modrinth.com/v2/version_file/fingerprint1?algorithm=sha1',
      { headers: Modrinth.API_HEADERS }
    );
    expect(rateLimitingFetch).toHaveBeenNthCalledWith(
      2,
      'https://api.modrinth.com/v2/version_file/fingerprint2?algorithm=sha1',
      { headers: Modrinth.API_HEADERS }
    );
    expect(rateLimitingFetch).toHaveBeenNthCalledWith(
      3,
      'https://api.modrinth.com/v2/version_file/fingerprint3?algorithm=sha1',
      { headers: Modrinth.API_HEADERS }
    );

    expect(actual).toEqual([]);
  });

  it<LocalTestContext>('transforms the response correctly', async () => {
    const modId = chance.hash({ length: 6 });
    const randomHash = chance.hash();

    const file = generateModrinthFile({
      hashes: { sha1: randomHash } as unknown as Hash
    }).generated;
    // eslint-disable-next-line camelcase
    const modVersion = generateModrinthVersion({ project_id: modId, files: [file] }).generated;

    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: true,
      json: async () => modVersion
    } as unknown as Response);
    const actual = await lookup([randomHash]);

    expect(actual.length).toEqual(1);
    expect(actual[0].platform).toEqual(Platform.MODRINTH);
    expect(actual[0].modId).toEqual(modId.toString());
    expect(actual[0].mod.hash).toEqual(randomHash);
    expect(actual[0].mod.name).toEqual(modVersion.name);
    expect(actual[0].mod.releaseDate).toEqual(modVersion.date_published);
    expect(actual[0].mod.fileName).toEqual(file.filename);
    expect(actual[0].mod.downloadUrl).toEqual(file.url);
  });
});
