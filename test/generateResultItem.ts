import { chance } from 'jest-chance';
import { ResultItem } from '../src/repositories/index.js';
import { generatePlatformLookupResult } from './generatePlatformLookupResult.js';
import { GeneratorResult } from './test.types.js';

const generateHits = (numberOfHits = 1) => {
  return chance.n(() => {
    return generatePlatformLookupResult().generated;
  }, numberOfHits);
};

export const generateResultItem = (overrides?: Partial<ResultItem>): GeneratorResult<ResultItem> => {
  const randomHash = overrides?.sha1Hash || chance.hash();
  const hits = overrides?.hits || generateHits(chance.ingeger({ min: 0, max: 5 }));

  return {
    generated: {
      sha1Hash: randomHash,
      hits: hits
    },
    expected: {
      sha1Hash: randomHash,
      hits: hits
    }
  };
};
