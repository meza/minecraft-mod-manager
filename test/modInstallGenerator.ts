import { ModInstall, Platform } from '../src/lib/modlist.types.js';
import { GeneratorResult } from './test.types.js';
import { chance } from 'jest-chance';

export const generateModInstall = (overrides?: Partial<ModInstall>): GeneratorResult<ModInstall> => {

  const fileName = chance.word();
  const releasedOn = chance.date({ string: true });
  const hash = chance.hash();
  const downloadUrl = chance.url();
  const name = chance.word();
  const type = chance.pickone(Object.values(Platform));
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
