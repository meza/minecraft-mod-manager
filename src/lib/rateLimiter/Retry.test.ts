import { describe, it, expect } from 'vitest';
import { Retrying } from './Retrying.js';

describe('The retry exception', () => {
  it('can return the last response', () => {
    const testResponse = { ok: true } as Response;

    const retry = new Retrying(testResponse);

    expect(retry.lastResponse()).toBe(testResponse);
  });
});
