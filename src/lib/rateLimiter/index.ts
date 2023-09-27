import { Queue } from './queue.js';
import { FetchJob } from './FetchJob.js';
import { Retrying } from './Retrying.js';

export interface RateLimit {
  maxAttempts: number;
  timeBetweenCalls: number;
}

interface JobState {
  host: string;
  running: boolean;
}

interface QueueRecord {
  host: string;
  queue: Queue<FetchJob>;
}

const defaultRateLimiting: RateLimit = {
  timeBetweenCalls: 100,
  maxAttempts: 3
};

const queues: QueueRecord[] = [];
const state: JobState[] = [];

const isRunning = (forHost: string) => {
  const index = state.findIndex((s) => s.host === forHost);

  if (index === -1) {
    return false;
  }

  return state[index].running;
};

const mark = (forHost: string, newState: boolean) => {
  const index = state.findIndex((s) => s.host === forHost);

  if (index === -1) {
    state.push({
      host: forHost,
      running: newState
    });
    return;
  }

  state[index].running = newState;
};

const getQueue = (forHost: string): Queue<FetchJob> => {
  const index = queues.findIndex(q => q.host === forHost);

  if (index === -1) {
    const newQueue = new Queue<FetchJob>();
    queues.push({
      host: forHost,
      queue: newQueue
    });
    return newQueue;
  }

  return queues[index].queue;
};

const processQueue = (host: string, queue: Queue<FetchJob>) => {
  const item = queue.dequeue();

  if (!item) {
    mark(host, false);
    return;
  }

  mark(host, true);

  item.execute()
    .catch((e) => {
      if (e instanceof Retrying) {
        queue.enqueue(item);
      }
    })
    .finally(() => {
      if (!queue.isEmpty()) {
        setTimeout(() => {
          processQueue(host, queue);
        }, item.retryIn());
        return;
      }
      mark(host, false);
    });
};

export const rateLimitingFetch = (input: RequestInfo | URL, init?: RequestInit, rateLimit?: RateLimit): Promise<Response> => {
  const request = new Request(input);
  const host = new URL(request.url).hostname;
  const jobs = getQueue(host);

  const promise = new Promise<Response>((resolve, reject) => {
    const job = new FetchJob(input, init || {}, rateLimit || defaultRateLimiting);
    job.onResponse(resolve);
    job.onError(reject);
    jobs.enqueue(job);
  });

  if (!isRunning(host)) {
    mark(host, true);
    setTimeout(() => {
      processQueue(host, jobs);
    }, 100);
  }

  return promise;
};
