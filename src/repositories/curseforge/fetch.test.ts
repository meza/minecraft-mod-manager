import { beforeEach, describe, expect, it, vi } from 'vitest';
import { chance } from 'jest-chance';
import { curseforgeFileToRemoteModDetails, CurseforgeModFile, getMod, HashFunctions } from './fetch.js';
import { Loader, Platform, ReleaseType, RemoteModDetails } from '../../lib/modlist.types.js';
import { RepositoryTestContext } from '../index.test.js';
import { CouldNotFindModException } from '../../errors/CouldNotFindModException.js';
import { generateCurseforgeModFile } from '../../../test/generateCurseforgeModFile.js';
import { NoRemoteFileFound } from '../../errors/NoRemoteFileFound.js';
import { rateLimitingFetch } from '../../lib/rateLimiter/index.js';

enum Release {
  ALPHA = 3,
  BETA = 2,
  RELEASE = 1
}

vi.mock('../../lib/rateLimiter/index.js');

const releasedStatus = 10;

const assumeFailedModFetch = () => {
  vi.mocked(rateLimitingFetch).mockResolvedValue({
    ok: false
  } as Response);
};

const assumeSuccessfulModFetch = (modName: string, latestFiles: CurseforgeModFile[]) => {
  vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
    ok: true,
    json: () => Promise.resolve({
      data: {
        name: modName
      }
    })
  } as Response);

  vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
    ok: true,
    json: () => Promise.resolve({
      data: latestFiles
    })
  } as Response);
};

describe('The Curseforge repository', () => {

  beforeEach<RepositoryTestContext>((context) => {
    vi.resetAllMocks();
    context.platform = Platform.CURSEFORGE;
    context.id = chance.word();
    context.allowedReleaseTypes = chance.pickset(Object.values(ReleaseType), chance.integer({
      min: 1,
      max: Object.keys(ReleaseType).length
    }));
    context.gameVersion = chance.pickone(['1.16.5', '1.17.1', '1.18.1', '1.18.2', '1.19']);
    context.loader = chance.pickone(Object.values(Loader));
    context.allowFallback = false;
  });

  it<RepositoryTestContext>('throws an error when the mod details could not be fetched', async (context) => {
    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        data: {
          name: chance.word()
        }
      })
    } as Response);

    vi.mocked(rateLimitingFetch).mockResolvedValueOnce({
      ok: false
    } as Response);

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new CouldNotFindModException(context.id, context.platform));
  });

  it<RepositoryTestContext>('throws an error when the files cannot be fetched', async (context) => {
    assumeFailedModFetch();

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new CouldNotFindModException(context.id, context.platform));
  });

  it<RepositoryTestContext>('throws an error when CF returns an invalid release type', async (context) => {
    const randomName = chance.word();
    const randomBadReleaseType = chance.integer({ min: 4, max: 100 });
    const randomFile = generateCurseforgeModFile({
      isAvailable: true,
      sortableGameVersions: [{
        gameVersion: context.gameVersion,
        gameVersionName: context.loader
      }],
      releaseType: randomBadReleaseType
    });

    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));
  });

  it<RepositoryTestContext>('throws an error when CF returns with no valid hash', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      isAvailable: true,
      fileStatus: releasedStatus,
      releaseType: Release.ALPHA,
      sortableGameVersions: [{
        gameVersion: context.gameVersion,
        gameVersionName: context.loader
      }],
      hashes: []
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        [ReleaseType.ALPHA],
        context.gameVersion,
        context.loader,
        false
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));
  });

  it<RepositoryTestContext>('throws an error when no files match the requested game version', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      isAvailable: true,
      fileStatus: releasedStatus,
      releaseType: Release.RELEASE,
      sortableGameVersions: [{
        gameVersionName: context.loader,
        gameVersion: 'improper version'
      }]
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        [ReleaseType.RELEASE],
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));

  });

  it<RepositoryTestContext>('throws an error when no files match the requested loader', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      sortableGameVersions: [{
        gameVersionName: 'no real loader',
        gameVersion: context.gameVersion
      }]
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));

  });

  it<RepositoryTestContext>('throws an error when no files are available', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      isAvailable: false,
      sortableGameVersions: [{
        gameVersionName: context.loader,
        gameVersion: context.gameVersion
      }]
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));
  });

  it<RepositoryTestContext>('throws an error when no files match the release type', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      isAvailable: true,
      releaseType: Release.BETA,
      sortableGameVersions: [{
        gameVersionName: context.loader,
        gameVersion: context.gameVersion
      }]
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    await expect(async () => {
      await getMod(
        context.id,
        [ReleaseType.RELEASE],
        context.gameVersion,
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));

  });

  describe.each([1, 2, 3, 5, 6, 7, 8, 9, 11, 12, 13, 14, 15])('when the file status is %i', (status) => {
    it<RepositoryTestContext>(`throws an error for ${status}`, async (context) => {
      const randomName = chance.word();
      const randomFile = generateCurseforgeModFile({
        isAvailable: true,
        fileStatus: status,
        sortableGameVersions: [{
          gameVersionName: context.loader,
          gameVersion: context.gameVersion
        }]
      });
      assumeSuccessfulModFetch(randomName, [randomFile.generated]);

      await expect(async () => {
        await getMod(
          context.id,
          [ReleaseType.RELEASE, ReleaseType.BETA, ReleaseType.ALPHA],
          context.gameVersion,
          context.loader,
          context.allowFallback
        );
      }).rejects.toThrow(new NoRemoteFileFound(randomName, context.platform));
    });
  });

  describe.each([
    { version: '1.19.1', message: 'a one lower' },
    { version: '1.19', message: 'the relevant major' }
  ])('when version fallback is allowed and the available version is $message version', ({ version }) => {
    beforeEach<RepositoryTestContext>((context) => {
      context.allowFallback = true;
    });

    it<RepositoryTestContext>(`it finds ${version} correctly instead of 1.19.2`, async (context) => {
      const randomName = chance.word();
      const randomFile = generateCurseforgeModFile({
        isAvailable: true,
        fileStatus: releasedStatus,
        releaseType: Release.RELEASE,
        sortableGameVersions: [{
          gameVersionName: context.loader,
          gameVersion: version
        }]
      });
      assumeSuccessfulModFetch(randomName, [randomFile.generated]);

      const actual = await getMod(
        context.id,
        [ReleaseType.RELEASE],
        '1.19.2',
        context.loader,
        true
      );

      expect(actual).toEqual({
        name: randomName,
        fileName: randomFile.generated.fileName,
        releaseDate: randomFile.generated.fileDate,
        hash: randomFile.generated.hashes.find((hash) => hash.algo === HashFunctions.sha1)?.value,
        downloadUrl: randomFile.generated.downloadUrl
      });

    });
  });

  it<RepositoryTestContext>('returns the file when a perfect game version match is found', async (context) => {
    const randomName = chance.word();
    const randomFile = generateCurseforgeModFile({
      isAvailable: true,
      fileStatus: releasedStatus,
      releaseType: Release.RELEASE,
      sortableGameVersions: [{
        gameVersionName: context.loader,
        gameVersion: context.gameVersion
      }]
    });
    assumeSuccessfulModFetch(randomName, [randomFile.generated]);

    const actual = await getMod(
      context.id,
      [ReleaseType.RELEASE],
      context.gameVersion,
      context.loader,
      context.allowFallback
    );

    expect(actual).toEqual({
      name: randomName,
      fileName: randomFile.generated.fileName,
      releaseDate: randomFile.generated.fileDate,
      hash: randomFile.generated.hashes.find((hash) => hash.algo === HashFunctions.sha1)?.value,
      downloadUrl: randomFile.generated.downloadUrl
    });

  });

  it<RepositoryTestContext>('returns the most recent file for a given version', async (context) => {
    const randomName = chance.word();
    const randomFile1 = generateCurseforgeModFile({
      isAvailable: true,
      fileStatus: releasedStatus,
      fileDate: '2019-08-24T14:15:22Z',
      releaseType: Release.RELEASE,
      sortableGameVersions: [{
        gameVersionName: context.loader,
        gameVersion: context.gameVersion
      }]
    });
    const randomFile2 = generateCurseforgeModFile({
      isAvailable: true,
      fileStatus: releasedStatus,
      fileDate: '2020-08-24T14:15:22Z',
      releaseType: Release.RELEASE,
      sortableGameVersions: [{
        gameVersionName: context.loader,
        gameVersion: context.gameVersion
      }]
    });
    const randomFile3 = generateCurseforgeModFile({
      isAvailable: true,
      fileStatus: releasedStatus,
      fileDate: '2018-08-24T14:15:22Z',
      releaseType: Release.RELEASE,
      sortableGameVersions: [{
        gameVersionName: context.loader,
        gameVersion: context.gameVersion
      }]
    });
    const randomFile4 = generateCurseforgeModFile({
      isAvailable: true,
      fileStatus: releasedStatus,
      fileDate: '2018-08-24T14:15:22Z',
      releaseType: Release.RELEASE,
      sortableGameVersions: [{
        gameVersionName: context.loader,
        gameVersion: context.gameVersion
      }]
    });
    assumeSuccessfulModFetch(randomName, [
      randomFile1.generated,
      randomFile2.generated,
      randomFile3.generated,
      randomFile4.generated
    ]);

    const actual = await getMod(
      context.id,
      [ReleaseType.RELEASE],
      context.gameVersion,
      context.loader,
      context.allowFallback
    );

    expect(actual).toEqual({
      name: randomName,
      fileName: randomFile2.generated.fileName,
      releaseDate: randomFile2.generated.fileDate,
      hash: randomFile2.generated.hashes.find((hash) => hash.algo === HashFunctions.sha1)?.value,
      downloadUrl: randomFile2.generated.downloadUrl
    });

  });

  it('can convert a CF file to Remote Mod Details', () => {
    const randomName = chance.word();
    const randomFileName = chance.word();
    const randomFileDate = chance.date();
    const randomHash = chance.word();
    const randomDownloadUrl = chance.word();

    const file = generateCurseforgeModFile({
      fileName: randomFileName,
      fileDate: randomFileDate,
      downloadUrl: randomDownloadUrl,
      hashes: [
        {
          algo: HashFunctions.sha1,
          value: randomHash
        }
      ]
    }).generated;

    const actual: RemoteModDetails = curseforgeFileToRemoteModDetails(file, randomName);

    expect(actual.name).toEqual(randomName);
    expect(actual.hash).toEqual(randomHash);
    expect(actual.fileName).toEqual(randomFileName);
    expect(actual.releaseDate).toEqual(randomFileDate);
    expect(actual.downloadUrl).toEqual(randomDownloadUrl);

  });

});
