/* eslint-disable camelcase */
import { ModrinthFile, ModrinthVersion } from '../src/repositories/modrinth/fetch.js';
import { GeneratorResult } from './test.types.js';
import { chance } from 'jest-chance';
import { Loader } from '../src/lib/modlist.types.js';
import { generateModrinthFile } from './generateModrinthFile.js';

export const generateModrinthVersion = (overrides?: Partial<ModrinthVersion>): GeneratorResult<ModrinthVersion> => {

  const name = chance.word();
  const projectId = chance.word();
  const versionType = chance.integer({ min: 1, max: 3 });
  const datePublished = chance.date().toISOString();
  const loaders = chance.pickset(Object.values(Loader), chance.integer({ min: 1, max: 2 }));
  const gameVersions = chance.pickset(['1.16.5', '1.17.1', '1.18', '1.19', '1.19.2'], chance.integer({
    min: 1,
    max: 2
  }));
  const filesToGenerate = chance.integer({ min: 1, max: 3 });
  const files: ModrinthFile[] = [];

  for (let i = 0; i < filesToGenerate; i++) {
    files.push(generateModrinthFile().generated);
  }

  const generated: ModrinthVersion = {
    name: name,
    project_id: projectId,
    loaders: loaders,
    game_versions: gameVersions,
    date_published: datePublished,
    version_type: versionType,
    files: files,
    ...overrides
  };

  const expected: ModrinthVersion = {
    name: name,
    project_id: projectId,
    loaders: loaders,
    game_versions: gameVersions,
    date_published: datePublished,
    version_type: versionType,
    files: files,
    ...overrides
  };

  return {
    generated: generated,
    expected: expected
  };
};
