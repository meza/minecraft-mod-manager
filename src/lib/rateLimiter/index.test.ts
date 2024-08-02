import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MaximumRetriesReached } from './MaximumRetriesReached.js';
import { RateLimit, rateLimitingFetch } from './index.js';
import { Queue } from './queue.js';

import { FetchJob } from './FetchJob.js';
import * as queueExports from './queue.js';

interface LocalTestContext {
  rateLimit: RateLimit;
  init: RequestInit;
  input: RequestInfo | URL;
  randomResponse: (success?: boolean) => Response;
}

/**
 * Decided to not mock the underlying FetchJob class as it would introduce way more complexity to the tests.
 * Instead, we're mocking the transport layer beneath the FetchJob class.
 *
 * It's not the nicest thing to do, but it will have to do for now.
 * If you think you know of a more readable and nicer way of doing it, please submit a PR on github.
 */
describe('The Rate Limiting Library', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    vi.stubGlobal('fetch', vi.fn());

    context.init = {
      method: 'GET'
    };
    context.rateLimit = {
      timeBetweenCalls: 0,
      maxAttempts: 3
    };
    context.input = chance.url({ protocol: 'https' });
    context.randomResponse = (success = true) =>
      ({
        ok: success,
        headers: {
          has: vi.fn().mockReturnValue(false),
          get: vi.fn()
        }
      }) as unknown as Response;
  });

  it<LocalTestContext>('can resolve successfully', async ({ randomResponse, init, input, rateLimit }) => {
    const response = randomResponse();
    vi.mocked(fetch).mockResolvedValueOnce(response);

    const actual = await rateLimitingFetch(input, init, rateLimit);
    expect(actual).toBe(response);
  });

  it<LocalTestContext>('can reject successfully', async ({ randomResponse, init, input }) => {
    const response = randomResponse(false);
    vi.mocked(fetch).mockResolvedValue(response);
    await expect(
      rateLimitingFetch(input, init, {
        timeBetweenCalls: 0,
        maxAttempts: 3
      })
    ).rejects.toThrow(MaximumRetriesReached);

    expect(fetch).toHaveBeenCalledTimes(3); //maxAttempts amount of times
  });

  it<LocalTestContext>('can throw successfully', async ({ init, input }) => {
    const error = new Error('happens rarely');
    vi.mocked(fetch).mockRejectedValue(error);
    await expect(
      rateLimitingFetch(input, init, {
        maxAttempts: 3,
        timeBetweenCalls: 0
      })
    ).rejects.toThrow(error);

    expect(fetch).toHaveBeenCalledTimes(1); //regardless of max retries, we reject after the first
  });

  it<LocalTestContext>('can handle multiple hosts', async ({ randomResponse, init }) => {
    const response1 = randomResponse();
    const response2 = randomResponse();
    vi.useRealTimers();
    vi.mocked(fetch).mockResolvedValueOnce(response1);
    vi.mocked(fetch).mockResolvedValueOnce(response2);

    const retry: RateLimit = {
      maxAttempts: 1,
      timeBetweenCalls: 10000 //set it to high to make sure different inputs don't queue
    };

    const actual1 = await rateLimitingFetch(chance.url(), init, retry);
    const actual2 = await rateLimitingFetch(chance.url(), init, retry);

    expect(actual1).toBe(response1);
    expect(actual2).toBe(response2);
  });

  it<LocalTestContext>('can queue multiple requests to the same host', async ({ randomResponse, init }) => {
    vi.useFakeTimers({
      now: 0,
      shouldAdvanceTime: true
    });

    const response = randomResponse(true);
    const url = chance.url();

    vi.mocked(fetch).mockResolvedValue(response);

    const retry: RateLimit = {
      maxAttempts: 1,
      timeBetweenCalls: 500 //set it to high to make sure different inputs don't queue
    };

    const promises = [rateLimitingFetch(url, init, retry), rateLimitingFetch(url)];

    await Promise.all(promises);

    /**
     * 100 for the initial process delay
     * 500 for the timeBetweenCalls
     * ---
     * 600
     */
    expect(Date.now()).toEqual(600);
  });

  it<LocalTestContext>('can handle a suddenly empty queue', ({ input }) => {
    /**
     * This is mainly to cover a very slim edge case that should never happen.
     */
    vi.useFakeTimers();
    vi.spyOn(queueExports, 'Queue').mockImplementation(
      () =>
        ({
          dequeue: () => undefined,
          enqueue: () => {}
        }) as unknown as Queue<FetchJob>
    );

    rateLimitingFetch(input);

    vi.advanceTimersByTime(500);

    expect(fetch).not.toHaveBeenCalled();
  });
});
