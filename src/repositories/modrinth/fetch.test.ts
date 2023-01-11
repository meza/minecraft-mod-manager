import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { RepositoryTestContext } from '../index.test.js';
import { Loader, Platform, ReleaseType } from '../../lib/modlist.types.js';
import { chance } from 'jest-chance';
import { getMod, ModrinthVersion } from './fetch.js';
import { CouldNotFindModException } from '../../errors/CouldNotFindModException.js';
import { generateModrinthVersion } from '../../../test/generateModrinthVersion.js';
import { NoRemoteFileFound } from '../../errors/NoRemoteFileFound.js';
import { generateModrinthFile } from '../../../test/generateModrinthFile.js';

const assumeFailedModFetch = () => {
  vi.mocked(fetch).mockResolvedValueOnce({
    status: chance.pickone([401, 404, 500])
  } as Response);
};

const assumeSuccessfulModFetch = (name: string) => {
  vi.mocked(fetch).mockResolvedValueOnce({
    status: 200,
    json: () => Promise.resolve({
      title: name
    })
  } as Response); // name fetch
};

const assumeFailedDetailsFetch = (name: string) => {
  assumeSuccessfulModFetch(name);
  vi.mocked(fetch).mockResolvedValueOnce({
    status: chance.pickone([401, 404, 500])
  } as Response);
};

const assumeSuccessfulDetailsFetch = (name: string, data: ModrinthVersion[]) => {
  assumeSuccessfulModFetch(name);
  vi.mocked(fetch).mockResolvedValueOnce({
    status: 200,
    json: () => Promise.resolve(data)
  } as Response);
};

describe('The Modrinth repository', () => {

  beforeEach<RepositoryTestContext>((context) => {
    vi.stubGlobal('fetch', vi.fn());
    context.platform = Platform.MODRINTH;
    context.id = chance.word();
    context.allowedReleaseTypes = chance.pickset(Object.values(ReleaseType), chance.integer({
      min: 1,
      max: Object.keys(ReleaseType).length
    }));
    context.gameVersion = chance.pickone(['1.16.5', '1.17.1', '1.18.1', '1.18.2', '1.19']);
    context.loader = chance.pickone(Object.values(Loader));
    context.allowFallback = false;
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it<RepositoryTestContext>('throws an error when the mod details could not be fetched', async (context) => {
    assumeFailedModFetch();
    await expect(async () => {
      await getMod(context.id, context.allowedReleaseTypes, context.gameVersion, context.loader, context.allowFallback);
    }).rejects.toThrow(new CouldNotFindModException(context.id, Platform.MODRINTH));
  });

  it<RepositoryTestContext>('throws an error when the mod version could not be fetched', async (context) => {
    assumeFailedDetailsFetch(chance.word());
    await expect(async () => {
      await getMod(context.id, context.allowedReleaseTypes, context.gameVersion, context.loader, context.allowFallback);
    }).rejects.toThrow(new CouldNotFindModException(context.id, Platform.MODRINTH));
  });

  it<RepositoryTestContext>('throws an error when no files match the requested loader', async (context) => {
    const randomName = chance.word();
    const fakeLoader = chance.word();
    const randomVersion = generateModrinthVersion({ loaders: [fakeLoader] }).generated;

    assumeSuccessfulDetailsFetch(randomName, [randomVersion]);

    await expect(async () => {
      await getMod(context.id, context.allowedReleaseTypes, context.gameVersion, context.loader, context.allowFallback);
    }).rejects.toThrow(new NoRemoteFileFound(context.id, Platform.MODRINTH));
  });

  it<RepositoryTestContext>('throws an error when Modrinth returns an invalid release type', async (context) => {
    const randomName = chance.word();
    // eslint-disable-next-line camelcase
    const randomVersion = generateModrinthVersion({ version_type: 'something' as ReleaseType }).generated;

    assumeSuccessfulDetailsFetch(randomName, [randomVersion]);

    await expect(async () => {
      await getMod(context.id, context.allowedReleaseTypes, context.gameVersion, context.loader, context.allowFallback);
    }).rejects.toThrow(new NoRemoteFileFound(context.id, Platform.MODRINTH));
  });

  it<RepositoryTestContext>('throws an error when no files match the requested game version', async (context) => {
    const randomName = chance.word();

    // eslint-disable-next-line camelcase
    const randomVersion = generateModrinthVersion({ game_versions: ['totally-bad-version'] }).generated;

    assumeSuccessfulDetailsFetch(randomName, [randomVersion]);

    await expect(async () => {
      await getMod(
        context.id,
        [randomVersion.version_type],
        'correct-version',
        randomVersion.loaders[0] as Loader,
        false
      );
    }).rejects.toThrow(new NoRemoteFileFound(context.id, Platform.MODRINTH));
  });

  it<RepositoryTestContext>('throws an error when no files are available', async (context) => {
    const randomName = chance.word();

    const randomVersion = generateModrinthVersion({ files: [] }).generated;

    assumeSuccessfulDetailsFetch(randomName, [randomVersion]);

    await expect(async () => {
      await getMod(
        context.id,
        context.allowedReleaseTypes,
        'correct-version',
        context.loader,
        context.allowFallback
      );
    }).rejects.toThrow(new NoRemoteFileFound(context.id, Platform.MODRINTH));
  });

  it<RepositoryTestContext>('throws an error when no files match the release type', async (context) => {
    const randomName = chance.word();
    // eslint-disable-next-line camelcase
    const randomVersion = generateModrinthVersion({ version_type: ReleaseType.ALPHA as ReleaseType }).generated;

    assumeSuccessfulDetailsFetch(randomName, [randomVersion]);

    await expect(async () => {
      await getMod(context.id, [ReleaseType.BETA, ReleaseType.RELEASE], context.gameVersion, context.loader, context.allowFallback);
    }).rejects.toThrow(new NoRemoteFileFound(context.id, Platform.MODRINTH));
  });

  describe.each([
    { version: '1.19.1', message: 'a one lower' },
    { version: '1.19', message: 'the relevant major' }
  ])('when version fallback is allowed and the available version is $message version', ({ version }) => {

    afterEach(() => {
      vi.resetAllMocks();
    });

    it<RepositoryTestContext>(`it finds ${version} correctly instead of 1.19.2`, async (context) => {
      const randomName = chance.word();
      const randomFile = generateModrinthFile().generated;
      const randomVersion = generateModrinthVersion({
        loaders: [context.loader],
        // eslint-disable-next-line camelcase
        version_type: ReleaseType.RELEASE,
        // eslint-disable-next-line camelcase
        game_versions: [version],
        files: [randomFile]
      }).generated;
      const randomVersionRedHerring = generateModrinthVersion({
        loaders: [context.loader],
        // eslint-disable-next-line camelcase
        version_type: chance.pickone(context.allowedReleaseTypes),
        // eslint-disable-next-line camelcase
        game_versions: ['1.19.0']
      }).generated;

      assumeSuccessfulDetailsFetch(randomName, [randomVersionRedHerring, randomVersion]);

      const actual = await getMod(
        context.id,
        [ReleaseType.RELEASE],
        '1.19.2',
        context.loader,
        true
      );

      expect(actual).toEqual({
        name: randomName,
        fileName: randomFile.filename,
        releaseDate: randomVersion.date_published,
        hash: randomFile.hashes.sha1,
        downloadUrl: randomFile.url
      });
    });
  });

  it<RepositoryTestContext>('returns the file when a perfect game version match is found', async (context) => {
    const randomName = chance.word();
    const randomFile = generateModrinthFile().generated;
    const version = '1.19.2';
    const randomVersion = generateModrinthVersion({
      loaders: [context.loader],
      // eslint-disable-next-line camelcase
      version_type: ReleaseType.RELEASE,
      // eslint-disable-next-line camelcase
      game_versions: [version],
      files: [randomFile]
    }).generated;
    const randomVersionRedHerring = generateModrinthVersion({
      loaders: [context.loader],
      // eslint-disable-next-line camelcase
      version_type: chance.pickone(context.allowedReleaseTypes),
      // eslint-disable-next-line camelcase
      game_versions: ['1.19.0']
    }).generated;

    assumeSuccessfulDetailsFetch(randomName, [randomVersionRedHerring, randomVersion]);

    const actual = await getMod(
      context.id,
      [ReleaseType.RELEASE],
      '1.19.2',
      context.loader,
      true
    );

    expect(actual).toEqual({
      name: randomName,
      fileName: randomFile.filename,
      releaseDate: randomVersion.date_published,
      hash: randomFile.hashes.sha1,
      downloadUrl: randomFile.url
    });
  });

  it<RepositoryTestContext>('returns the most recent file for a given version', async (context) => {
    const randomName = chance.word();
    const randomFile = generateModrinthFile().generated;
    const version = '1.19.2';
    const randomVersion = generateModrinthVersion({
      loaders: [context.loader],
      // eslint-disable-next-line camelcase
      version_type: ReleaseType.RELEASE,
      // eslint-disable-next-line camelcase
      game_versions: [version],
      // eslint-disable-next-line camelcase
      date_published: '2021-01-03',
      files: [randomFile]
    }).generated;
    const randomVersionRedHerring = generateModrinthVersion({
      loaders: [context.loader],
      // eslint-disable-next-line camelcase
      version_type: ReleaseType.RELEASE,
      // eslint-disable-next-line camelcase
      game_versions: [version],
      // eslint-disable-next-line camelcase
      date_published: '2021-01-01'
    }).generated;
    const randomVersionAnotherRedHerring = generateModrinthVersion({
      loaders: [context.loader],
      // eslint-disable-next-line camelcase
      version_type: ReleaseType.RELEASE,
      // eslint-disable-next-line camelcase
      game_versions: [version],
      // eslint-disable-next-line camelcase
      date_published: '2021-01-02'
    }).generated;
    const randomVersionAnotherRedHerring2 = generateModrinthVersion({
      loaders: [context.loader],
      // eslint-disable-next-line camelcase
      version_type: ReleaseType.RELEASE,
      // eslint-disable-next-line camelcase
      game_versions: [version],
      // eslint-disable-next-line camelcase
      date_published: '2021-01-02'
    }).generated;

    assumeSuccessfulDetailsFetch(randomName, [randomVersionRedHerring, randomVersion, randomVersionAnotherRedHerring, randomVersionAnotherRedHerring2]);

    const actual = await getMod(
      context.id,
      [ReleaseType.RELEASE],
      '1.19.2',
      context.loader,
      true
    );

    expect(actual).toEqual({
      name: randomName,
      fileName: randomFile.filename,
      releaseDate: randomVersion.date_published,
      hash: randomFile.hashes.sha1,
      downloadUrl: randomFile.url
    });
  });

});
