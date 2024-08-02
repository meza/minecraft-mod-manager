import { chance } from 'jest-chance';
import { ModInstall } from '../src/lib/modlist.types.js';
import { generateRandomPlatform } from './generateRandomPlatform.js';
import { GeneratorResult } from './test.types.js';

export const generateModInstall = (overrides?: Partial<ModInstall>): GeneratorResult<ModInstall> => {
  const fileName = chance.word();
  const releasedOn = chance.date({ string: true });
  const hash = chance.hash();
  const downloadUrl = chance.url();
  const name = chance.word();
  const type = generateRandomPlatform();
  const id = chance.word();

  const generated: ModInstall = {
    type: type,
    id: id,
    fileName: fileName,
    releasedOn: releasedOn,
    hash: hash,
    downloadUrl: downloadUrl,
    name: name,
    ...overrides
  };

  const expected: ModInstall = {
    type: type,
    id: id,
    fileName: fileName,
    releasedOn: releasedOn,
    hash: hash,
    downloadUrl: downloadUrl,
    name: name,
    ...overrides
  };

  return {
    generated: generated,
    expected: expected
  };
};
