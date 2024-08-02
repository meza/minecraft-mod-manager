import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FetchJob } from './FetchJob.js';
import { MaximumRetriesReached } from './MaximumRetriesReached.js';
import { Retrying } from './Retrying.js';
import { RateLimit } from './index.js';

interface LocalTestContext {
  testRateLimit: RateLimit;
  randomDomain: string;
}

describe('The FetchJob class', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    vi.stubGlobal('fetch', vi.fn());

    context.testRateLimit = {
      maxAttempts: 3,
      timeBetweenCalls: 1000
    };

    context.randomDomain = chance.url({ protocol: 'https' });
  });

  it<LocalTestContext>('can return the host', ({ randomDomain, testRateLimit }) => {
    const job = new FetchJob(randomDomain, {}, testRateLimit);

    expect(job.host()).toEqual(new URL(randomDomain).host);
  });

  it<LocalTestContext>('returns the default retry rate', ({ randomDomain, testRateLimit }) => {
    const job = new FetchJob(randomDomain, {}, testRateLimit);

    expect(job.retryIn()).toEqual(testRateLimit.timeBetweenCalls);
  });

  it<LocalTestContext>('returns on a successful fetch', async ({ randomDomain, testRateLimit }) => {
    const randomResponse = {
      ok: true,
      headers: {
        has: vi.fn().mockReturnValue(false)
      }
    } as unknown as Response;

    vi.mocked(fetch).mockResolvedValueOnce(randomResponse);

    const job = new FetchJob(randomDomain, {}, testRateLimit);

    const actual = await job.execute();

    expect(actual).toBe(randomResponse);
  });

  it<LocalTestContext>('rejects properly', async ({ randomDomain, testRateLimit }) => {
    const randomReason = chance.word();

    vi.mocked(fetch).mockRejectedValueOnce(randomReason);

    const job = new FetchJob(randomDomain, {}, testRateLimit);

    await expect(job.execute()).rejects.toThrow(randomReason);
  });

  it<LocalTestContext>('calls the response handler on success', async ({ randomDomain, testRateLimit }) => {
    const randomResponse = {
      ok: true,
      headers: {
        has: vi.fn().mockReturnValue(false)
      }
    } as unknown as Response;

    vi.mocked(fetch).mockResolvedValueOnce(randomResponse);
    const handler = vi.fn();

    const job = new FetchJob(randomDomain, {}, testRateLimit);
    job.onResponse(handler);

    await job.execute();

    expect(handler).toHaveBeenCalledWith(randomResponse);
  });

  it<LocalTestContext>('calls the error handler on failure', async ({ randomDomain }) => {
    const randomResponse = {
      ok: false,
      headers: {
        has: vi.fn().mockReturnValue(false)
      }
    } as unknown as Response;

    vi.mocked(fetch).mockResolvedValueOnce(randomResponse);
    const handler = vi.fn();

    const job = new FetchJob(
      randomDomain,
      {},
      {
        timeBetweenCalls: 0,
        maxAttempts: 1
      }
    );

    job.onError(handler);

    try {
      await job.execute();
      expect('this should never happen').toEqual('');
    } catch (e) {
      expect(e).toBeInstanceOf(MaximumRetriesReached);
      expect((e as MaximumRetriesReached).response()).toBe(randomResponse);
    }

    expect(handler).toHaveBeenCalledWith(new MaximumRetriesReached(randomResponse));
  });

  it<LocalTestContext>('can operate without an error handler', async ({ randomDomain }) => {
    const randomResponse = {
      ok: false,
      headers: {
        has: vi.fn().mockReturnValue(false)
      }
    } as unknown as Response;

    vi.mocked(fetch).mockResolvedValueOnce(randomResponse);

    const job = new FetchJob(
      randomDomain,
      {},
      {
        timeBetweenCalls: 0,
        maxAttempts: 1
      }
    );

    try {
      await job.execute();
      expect('this should never happen').toEqual('');
    } catch (e) {
      expect(e).toBeInstanceOf(MaximumRetriesReached);
      expect((e as MaximumRetriesReached).response()).toBe(randomResponse);
    }
  });

  it<LocalTestContext>('can retry', async ({ randomDomain }) => {
    const randomResponse = {
      ok: false,
      headers: {
        has: vi.fn().mockReturnValue(false)
      }
    } as unknown as Response;

    vi.mocked(fetch).mockResolvedValue(randomResponse);
    const handler = vi.fn();

    const job = new FetchJob(
      randomDomain,
      {},
      {
        timeBetweenCalls: 0,
        maxAttempts: 3 // how many attempts to try
      }
    );
    job.onError(handler);

    await expect(job.execute()).rejects.toThrow(Retrying); //attempt 1
    await expect(job.execute()).rejects.toThrow(Retrying); //attempt 2
    await expect(job.execute()).rejects.toThrow(MaximumRetriesReached); //attempt 3

    expect(handler).toHaveBeenCalledWith(new MaximumRetriesReached(randomResponse));
  });

  it<LocalTestContext>('sets the retry time to the rate limit time', async ({ randomDomain }) => {
    const randomResponse = {
      ok: false,
      headers: {
        has: vi.fn(),
        get: vi.fn()
      }
    } as unknown as Response;

    vi.mocked(fetch).mockResolvedValue(randomResponse);

    const job = new FetchJob(
      randomDomain,
      {},
      {
        timeBetweenCalls: 0,
        maxAttempts: 3 // how many attempts to try
      }
    );

    vi.mocked(randomResponse.headers.has).mockReturnValue(true);
    vi.mocked(randomResponse.headers.get).mockReturnValueOnce('1');
    vi.mocked(randomResponse.headers.get).mockReturnValueOnce('10');

    await expect(job.execute()).rejects.toThrow(Retrying);

    expect(job.retryIn()).toEqual(11000);

    expect(vi.mocked(randomResponse.headers.has)).toHaveBeenCalledWith('X-Ratelimit-Remaining');
    expect(vi.mocked(randomResponse.headers.get)).toHaveBeenNthCalledWith(1, 'X-Ratelimit-Remaining');
    expect(vi.mocked(randomResponse.headers.get)).toHaveBeenNthCalledWith(2, 'X-Ratelimit-Reset');
  });

  it<LocalTestContext>('sets the retry time to the default time if there is none', async ({ randomDomain }) => {
    const randomResponse = {
      ok: false,
      headers: {
        has: vi.fn(),
        get: vi.fn()
      }
    } as unknown as Response;

    vi.mocked(fetch).mockResolvedValue(randomResponse);

    const job = new FetchJob(
      randomDomain,
      {},
      {
        timeBetweenCalls: 0,
        maxAttempts: 3 // how many attempts to try
      }
    );

    vi.mocked(randomResponse.headers.has).mockReturnValue(true);
    vi.mocked(randomResponse.headers.get).mockReturnValueOnce('1');
    vi.mocked(randomResponse.headers.get).mockReturnValueOnce(null);

    await expect(job.execute()).rejects.toThrow(Retrying);

    expect(job.retryIn()).toEqual(61000);
  });

  it<LocalTestContext>('ignores the retry if it is high enough', async ({ randomDomain }) => {
    const randomResponse = {
      ok: false,
      headers: {
        has: vi.fn(),
        get: vi.fn()
      }
    } as unknown as Response;

    vi.mocked(fetch).mockResolvedValue(randomResponse);

    const job = new FetchJob(
      randomDomain,
      {},
      {
        timeBetweenCalls: 0,
        maxAttempts: 3 // how many attempts to try
      }
    );

    vi.mocked(randomResponse.headers.has).mockReturnValue(true);
    vi.mocked(randomResponse.headers.get).mockReturnValueOnce('11');
    vi.mocked(randomResponse.headers.get).mockReturnValueOnce('10');

    await expect(job.execute()).rejects.toThrow(Retrying);

    expect(job.retryIn()).toEqual(0);

    expect(vi.mocked(randomResponse.headers.has)).toHaveBeenCalledWith('X-Ratelimit-Remaining');
    expect(vi.mocked(randomResponse.headers.get)).toHaveBeenNthCalledWith(1, 'X-Ratelimit-Remaining');
    expect(vi.mocked(randomResponse.headers.get)).toHaveBeenNthCalledWith(2, 'X-Ratelimit-Reset');
  });
});
