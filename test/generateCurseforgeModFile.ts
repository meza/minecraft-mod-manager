import { chance } from 'jest-chance';
import { CurseforgeModFile, HashFunctions } from '../src/repositories/curseforge/fetch.js';
import { GeneratorResult } from './test.types.js';

export const generateCurseforgeModFile = (
  overrides?: Partial<CurseforgeModFile>
): GeneratorResult<CurseforgeModFile> => {
  const displayName = chance.word();
  const fileDate = chance.date().toISOString();
  const releaseType = chance.integer({ min: 1, max: 3 });
  const fileName = chance.word();
  const fileFingerprint = chance.integer({ min: 100000, max: 999999 });
  const downloadUrl = chance.url();
  const fileStatus = chance.integer({ min: 1, max: 3 });
  const isAvailable = chance.bool();
  const hashes = [
    {
      algo: HashFunctions.sha1,
      value: chance.hash()
    },
    {
      algo: HashFunctions.md5,
      value: chance.hash({ casing: 'upper', length: 16 })
    }
  ];
  const sortableGameVersions = [
    {
      gameVersionName: chance.word(),
      gameVersion: chance.word()
    }
  ];

  const generated: CurseforgeModFile = {
    fileFingerprint: fileFingerprint,
    displayName: displayName,
    fileDate: fileDate,
    releaseType: releaseType,
    fileName: fileName,
    downloadUrl: downloadUrl,
    fileStatus: fileStatus,
    isAvailable: isAvailable,
    hashes: hashes,
    sortableGameVersions: sortableGameVersions,
    ...overrides
  };

  const expected: CurseforgeModFile = {
    fileFingerprint: fileFingerprint,
    displayName: displayName,
    fileDate: fileDate,
    releaseType: releaseType,
    fileName: fileName,
    downloadUrl: downloadUrl,
    fileStatus: fileStatus,
    isAvailable: isAvailable,
    hashes: hashes,
    sortableGameVersions: sortableGameVersions,
    ...overrides
  };

  return {
    generated: generated,
    expected: expected
  };
};
