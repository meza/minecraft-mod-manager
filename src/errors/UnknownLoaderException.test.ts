import { describe, expect, it } from 'vitest';
import { generateRandomPlatform } from '../../test/generateRandomPlatform.js';
import { UnknownLoaderException } from './UnknownLoaderException.js';

describe('The Unknown Loader Exception', () => {
  it('records the platform', () => {
    const randomPlatform = generateRandomPlatform();

    const error = new UnknownLoaderException(randomPlatform);

    expect(error.loader).toBe(randomPlatform);
  });
});
