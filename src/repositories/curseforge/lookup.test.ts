import { describe, it, vi, beforeEach, expect } from 'vitest';
import { rateLimitingFetch } from '../../lib/rateLimiter/index.js';
import * as envvars from '../../env.js';
import { chance } from 'jest-chance';
import { lookup } from './lookup.js';
import { Logger } from '../../lib/Logger.js';
import { generateCurseforgeModFile } from '../../../test/generateCurseforgeModFile.js';
import { Platform } from '../../lib/modlist.types.js';
import { generateRemoteModDetails } from '../../../test/generateRemoteDetails.js';
import { curseforgeFileToRemoteModDetails } from './fetch.js';

vi.mock('../../lib/rateLimiter/index.js');
vi.mock('../../lib/Logger.js');
vi.mock('./fetch.js');

interface LocalTestContext {
  apiKey: string;
  logger: Logger;
}

describe('The Curseforge Lookup module', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.apiKey = chance.hash();

    vi.spyOn(envvars, 'curseForgeApiKey', 'get').mockReturnValue(context.apiKey);
  });

  it<LocalTestContext>('correctly calls the curseforge api', async ({ apiKey }) => {
    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: false // fastest way to exit out of the function under test
    } as unknown as Response);

    await lookup(['fingerprint1', 'fingerprint2', 'fingerprint3']);

    const fetchCall = vi.mocked(rateLimitingFetch).mock.calls[0];
    const url = fetchCall[0];
    const requestParams: RequestInit = fetchCall[1]!;

    expect(url).toMatchInlineSnapshot('"https://api.curseforge.com/v1/fingerprints"');
    expect(requestParams.method).toEqual('POST');
    expect(requestParams.headers).toHaveProperty('Accept', 'application/json');
    expect(requestParams.headers).toHaveProperty('Content-Type', 'application/json');
    expect(requestParams.headers).toHaveProperty('x-api-key', apiKey);
    expect(requestParams.body).toMatchInlineSnapshot('"{\\"fingerprints\\":[\\"fingerprint1\\",\\"fingerprint2\\",\\"fingerprint3\\"]}"');
  });

  it<LocalTestContext>('logs the failed attempt correctly', async ({ logger }) => {
    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: false // fastest way to exit out of the function under test
    } as unknown as Response);
    const actual = await lookup([]);

    const logMessage = vi.mocked(logger.log).mock.calls[0][0];
    expect(logMessage).toMatchInlineSnapshot('"Could not reach Curseforge, please try again"');

    expect(actual).toEqual([]);
  });

  it<LocalTestContext>('transforms the response correctly', async () => {
    const modId = chance.integer({ min: 6, max: 6 });
    const fingerprint = chance.integer({ min: 6, max: 6 });
    const modFile = generateCurseforgeModFile({ fileFingerprint: 123467 }).generated;
    const randomModFile = generateRemoteModDetails().generated;

    vi.mocked(curseforgeFileToRemoteModDetails).mockReturnValueOnce(randomModFile);
    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: {
          exactMatches: [
            {
              id: modId,
              file: modFile
            }
          ],
          exactFingerprints: [fingerprint]
        }
      })
    } as unknown as Response);
    const actual = await lookup([fingerprint]);

    expect(vi.mocked(curseforgeFileToRemoteModDetails)).toHaveBeenCalledWith(
      modFile,
      modFile.displayName
    );

    expect(actual.length).toEqual(1);
    expect(actual[0].platform).toEqual(Platform.CURSEFORGE);
    expect(actual[0].modId).toEqual(modId.toString());
    expect(actual[0].mod).toBe(randomModFile);
  });
});
