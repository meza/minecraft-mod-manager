import { MaximumRetriesReached } from './MaximumRetriesReached.js';
import { Retrying } from './Retrying.js';
import { RateLimit } from './index.js';

export class FetchJob {
  private tries = 0;
  private isRateLimiting = false;
  private rateLimitRetryInSeconds = 60;
  private readonly input: RequestInfo | URL;
  private readonly init?: RequestInit | undefined;
  private readonly rateLimit: RateLimit;
  private responseCallback: (result: Response) => void;
  private errorCallback: (error: Error) => void;

  constructor(input: RequestInfo | URL, init: RequestInit, rateLimit: RateLimit) {
    this.input = input;
    this.init = init;
    this.rateLimit = rateLimit;
    this.responseCallback = () => {
      //
    };
    this.errorCallback = () => {
      //
    };
  }

  host() {
    return new URL(new Request(this.input).url).host;
  }

  retryIn() {
    if (this.isRateLimiting) {
      return (this.rateLimitRetryInSeconds + 1) * 1000;
    }
    return this.rateLimit.timeBetweenCalls;
  }

  onResponse(responseCallback: (result: Response) => void) {
    this.responseCallback = responseCallback;
  }

  onError(errorCallback: (reason: unknown) => void) {
    this.errorCallback = errorCallback;
  }

  execute(): Promise<Response> {
    this.tries++;
    return new Promise<Response>((resolve, reject) => {
      fetch(this.input, this.init)
        .then((response) => {
          // handle rate limit headers
          if (response.headers.has('X-Ratelimit-Remaining')) {
            const remaining = response.headers.get('X-Ratelimit-Remaining');
            const resetInSeconds = response.headers.get('X-Ratelimit-Reset') || 60;

            if (Number(remaining) < 10) {
              this.isRateLimiting = true;
              this.rateLimitRetryInSeconds = Number(resetInSeconds);
            } else {
              this.isRateLimiting = false;
            }
          }

          if (!response.ok) {
            if (this.tries === this.rateLimit.maxAttempts) {
              this.errorCallback(new MaximumRetriesReached(response));
              reject(new MaximumRetriesReached(response));
              return;
            }
            reject(new Retrying(response));
            return;
          }

          // response.ok fallthrough
          this.responseCallback(response);
          resolve(response);
        })
        .catch((reason) => {
          this.errorCallback(reason);
          reject(reason);
          return;
        });
    });
  }
}
