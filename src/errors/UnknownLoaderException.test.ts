import { describe, it, expect } from 'vitest';
import { UnknownLoaderException } from './UnknownLoaderException.js';
import { generateRandomPlatform } from '../../test/generateRandomPlatform.js';

describe('The Unknown Loader Exception', () => {
  it('records the platform', () => {
    const randomPlatform = generateRandomPlatform();

    const error = new UnknownLoaderException(randomPlatform);

    expect(error.loader).toBe(randomPlatform);

  });
});
