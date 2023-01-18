import { describe, it, expect } from 'vitest';
import { Platform } from '../lib/modlist.types.js';
import { chance } from 'jest-chance';
import { UnknownLoaderException } from './UnknownLoaderException.js';

describe('The Unknown Loader Exception', () => {
  it('records the platform', () => {
    const randomPlatform = chance.pickone(Object.values(Platform));

    const error = new UnknownLoaderException(randomPlatform);

    expect(error.loader).toBe(randomPlatform);

  });
});
