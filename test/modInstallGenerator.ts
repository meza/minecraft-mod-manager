import { ModInstall } from '../src/lib/modlist.types.js';
import { GeneratorResult } from './test.types.js';
import { chance } from 'jest-chance';

export const generateModInstall = (overrides?: Partial<ModInstall>): GeneratorResult<ModInstall> => {

  const fileName = chance.word();
  const releasedOn = chance.date({ string: true });
  const hash = chance.hash();

  const generated: ModInstall = {

    fileName: fileName,
    releasedOn: releasedOn,
    hash: hash,
    ...overrides
  };

  const expected: ModInstall = {

    fileName: fileName,
    releasedOn: releasedOn,
    hash: hash,
    ...overrides
  };

  return {
    generated: generated,
    expected: expected
  };
};
