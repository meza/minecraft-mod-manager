import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ModInstall, ModsJson, Platform } from './modlist.types.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { generateRandomPlatform } from '../../test/generateRandomPlatform.js';
import { scan } from './scan.js';
import { fetchModDetails, lookup } from '../repositories/index.js';
import fs from 'fs/promises';
import { chance } from 'jest-chance';
import path from 'path';
import curseforge from '@meza/curseforge-fingerprint';
import { getHash } from './hash.js';
import { fileIsManaged } from './configurationHelper.js';
import { generateResultItem } from '../../test/generateResultItem.js';
import { generatePlatformLookupResult } from '../../test/generatePlatformLookupResult.js';
import { generateRemoteModDetails } from '../../test/generateRemoteDetails.js';

vi.mock('fs/promises');
vi.mock('./hash.js');
vi.mock('@meza/curseforge-fingerprint');
vi.mock('./configurationHelper.js');
vi.mock('../repositories/index.js');

interface LocalTestContext {
  randomConfiguration: ModsJson;
  randomInstallations: ModInstall[];
  randomPlatform: Platform;
}

describe('The scan library', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.randomConfiguration = generateModsJson().generated;
    context.randomInstallations = [];
    context.randomPlatform = generateRandomPlatform();
  });

  describe('when there are no files in the mods folder', () => {
    it<LocalTestContext>('returns an empty array', async (context) => {
      const randomModsFolder = chance.word();
      context.randomConfiguration.modsFolder = randomModsFolder;

      vi.mocked(fs.readdir).mockResolvedValueOnce([]);
      vi.mocked(lookup).mockResolvedValueOnce([]);

      const actual = await scan(context.randomPlatform, context.randomConfiguration, context.randomInstallations);

      expect(actual).toEqual([]);
      expect(vi.mocked(fs.readdir)).toHaveBeenCalledWith(path.resolve(randomModsFolder));
    });
  });

  describe('when all files are managed', () => {
    it<LocalTestContext>('returns an empty array', async (context) => {
      vi.mocked(fs.readdir).mockResolvedValueOnce(chance.n(chance.word, chance.integer({ min: 5, max: 50 })));
      vi.mocked(fileIsManaged).mockReturnValue(true);

      const actual = await scan(context.randomPlatform, context.randomConfiguration, context.randomInstallations);

      expect(actual).toEqual([]);
    });
  });

  describe('when there are unmanaged files in the mods folder', () => {
    it<LocalTestContext>('passes the correct inputs to the hashing functions', async (context) => {
      const randomModsFolder = 'mods-folder';
      const randomFileName = chance.word();
      const randomHash = chance.hash();
      const randomFingerprint = chance.integer({ min: 6, max: 6 });
      const expectedPath = path.resolve('mods-folder', randomFileName);

      context.randomConfiguration.modsFolder = randomModsFolder;
      vi.mocked(fs.readdir).mockResolvedValueOnce([randomFileName]);
      vi.mocked(fileIsManaged).mockReturnValueOnce(false); // non-managed path
      vi.mocked(curseforge.fingerprint).mockReturnValueOnce(randomFingerprint);
      vi.mocked(getHash).mockResolvedValueOnce(randomHash);

      vi.mocked(lookup).mockResolvedValueOnce([]); // we don't care about the return just yet

      await scan(context.randomPlatform, context.randomConfiguration, context.randomInstallations);

      //expectations
      // do we call the curseforge fingerprint with the correct values?
      expect(curseforge.fingerprint).toHaveBeenCalledOnce();
      expect(curseforge.fingerprint).toHaveBeenCalledWith(expectedPath);

      // do we call the modrinth hasher with the correct values?
      expect(getHash).toHaveBeenCalledOnce();
      expect(getHash).toHaveBeenCalledWith(expectedPath, 'sha1');

    });

    it<LocalTestContext>('passes the correct inputs to the lookup', async (context) => {

      const randomFileName = chance.word();
      const randomHash = chance.hash();
      const randomFingerprint = chance.integer({ min: 6, max: 6 });

      vi.mocked(fs.readdir).mockResolvedValueOnce([randomFileName]);
      vi.mocked(fileIsManaged).mockReturnValueOnce(false); // non-managed path
      vi.mocked(curseforge.fingerprint).mockReturnValueOnce(randomFingerprint);
      vi.mocked(getHash).mockResolvedValueOnce(randomHash);

      vi.mocked(lookup).mockResolvedValueOnce([]); // we don't care about the return just yet

      await scan(context.randomPlatform, context.randomConfiguration, context.randomInstallations);

      const lookupArgs = vi.mocked(lookup).mock.calls[0][0];

      expect(lookupArgs).toContainEqual({
        platform: Platform.CURSEFORGE,
        hash: [randomFingerprint.toString()]
      });
      expect(lookupArgs).toContainEqual({
        platform: Platform.MODRINTH,
        hash: [randomHash]
      });
    });

    describe('and the preferred platform has no results', () => {
      const preferredPlatform = Platform.MODRINTH;
      const notThePreferredPlatform = Platform.CURSEFORGE;

      it<LocalTestContext>('returns the first hits', async (context) => {
        const modId = chance.word();
        const expectedModDetails = generateRemoteModDetails({}).generated;
        const expectedHit = generatePlatformLookupResult({ modId: modId, mod: expectedModDetails, platform: notThePreferredPlatform }).generated;
        const notExpectedHit = generatePlatformLookupResult({ platform: notThePreferredPlatform }).generated;
        const lookupResult = generateResultItem({
          hits: [
            expectedHit,
            notExpectedHit
          ]
        }).generated;

        vi.mocked(fs.readdir).mockResolvedValueOnce([chance.word()]); // we don't care about the files
        vi.mocked(fetchModDetails).mockResolvedValueOnce(expectedModDetails);
        vi.mocked(lookup).mockResolvedValueOnce([lookupResult]);

        const actual = await scan(preferredPlatform, context.randomConfiguration, context.randomInstallations);

        expect(actual[0].resolvedDetails).toBe(expectedModDetails);
        expect(actual[0].localDetails).toBe(expectedHit);
        expect(actual).not.toContainEqual(notExpectedHit);

      });
    });

    describe('and the preferred platform has results', () => {
      const preferredPlatform = Platform.MODRINTH;
      const notThePreferredPlatform = Platform.CURSEFORGE;

      it<LocalTestContext>('returns the preferred platform results', async (context) => {
        const modId = chance.word();
        const expectedModDetails = generateRemoteModDetails().generated;
        const preferredHit = generatePlatformLookupResult({ modId: modId, mod: expectedModDetails, platform: preferredPlatform }).generated;
        const notPreferredHit = generatePlatformLookupResult({ platform: notThePreferredPlatform }).generated;
        const lookupResult = generateResultItem({
          hits: [
            notPreferredHit,
            preferredHit
          ]
        }).generated;

        vi.mocked(fs.readdir).mockResolvedValueOnce([chance.word()]); // we don't care about the files
        vi.mocked(fetchModDetails).mockResolvedValueOnce(expectedModDetails);
        vi.mocked(lookup).mockResolvedValueOnce([lookupResult]);

        const actual = await scan(preferredPlatform, context.randomConfiguration, context.randomInstallations);

        expect(actual[0].resolvedDetails).toBe(expectedModDetails);
        expect(actual[0].localDetails).toBe(preferredHit);
        expect(actual).not.toContainEqual(notPreferredHit);
      });
    });

    describe('and the mod details cannot be fetched', () => {
      it<LocalTestContext>('skips the mods in error', async (context) => {
        const lookupResult = generateResultItem({
          hits: [
            generatePlatformLookupResult().generated
          ]
        }).generated;

        vi.mocked(fs.readdir).mockResolvedValueOnce([chance.word()]); // we don't care about the files
        vi.mocked(fetchModDetails).mockRejectedValueOnce(new Error('test-error'));
        vi.mocked(lookup).mockResolvedValueOnce([lookupResult]);

        const actual = await scan(context.randomPlatform, context.randomConfiguration, context.randomInstallations);

        expect(actual).toEqual([]);

      });
    });
  });
});
