import { beforeEach, describe, it, vi, expect } from 'vitest';
import { Curseforge, CurseforgeLoader } from './index.js';
import { Loader, ReleaseType } from '../../lib/modlist.types.js';
import { chance } from 'jest-chance';
import { UnknownLoaderException } from '../../errors/UnknownLoaderException.js';
import { getMod } from './fetch.js';
import { generateRemoteModDetails } from '../../../test/generateRemoteDetails.js';
import { generatePlatformLookupResult } from '../../../test/generatePlatformLookupResult.js';
import { lookup as cfLookup } from './lookup.js';

vi.mock('./fetch.js');
vi.mock('./lookup.js');
describe('The Curseforge Repository class', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });
  describe('when converting loaders', () => {
    it('can identify Forge', () => {
      const actual = Curseforge.curseforgeLoaderFromLoader(Loader.FORGE);
      expect(actual).toEqual(CurseforgeLoader.FORGE);
    });
    it('can identify Fabric', () => {
      const actual = Curseforge.curseforgeLoaderFromLoader(Loader.FABRIC);
      expect(actual).toEqual(CurseforgeLoader.FABRIC);
    });
    it('can identify Quilt', () => {
      const actual = Curseforge.curseforgeLoaderFromLoader(Loader.QUILT);
      expect(actual).toEqual(CurseforgeLoader.QUILT);
    });
    it('can identify NeoForge', () => {
      const actual = Curseforge.curseforgeLoaderFromLoader(Loader.NEOFORGE);
      expect(actual).toEqual(CurseforgeLoader.NEOFORGE);
    });
    it('can identify Liteloader', () => {
      const actual = Curseforge.curseforgeLoaderFromLoader(Loader.LITELOADER);
      expect(actual).toEqual(CurseforgeLoader.LITELOADER);
    });
    it('fails on an unknown platform', () => {
      const loader = chance.word() as Loader;
      try {
        Curseforge.curseforgeLoaderFromLoader(loader);
        expect('this should never happen').toEqual('');
      } catch (e) {
        expect(e instanceof UnknownLoaderException).toBeTruthy();
        expect((e as UnknownLoaderException).loader).toEqual(loader);
      }
    });
  });

  it('calls through to the fetching module', async () => {
    const projectId = chance.word();
    const allowedReleaseTypes = [chance.pickone(Object.values(ReleaseType))];
    const allowedGameVersion = chance.word();
    const loader = chance.pickone(Object.values(Loader));
    const allowFallback = chance.bool();
    const result = generateRemoteModDetails().generated;
    vi.mocked(getMod).mockResolvedValueOnce(result);

    const curseforge = new Curseforge();
    const actual = await curseforge.fetchMod(
      projectId,
      allowedReleaseTypes,
      allowedGameVersion,
      loader,
      allowFallback
    );

    expect(actual).toEqual(result);
    expect(vi.mocked(getMod)).toHaveBeenCalledOnce();
    expect(vi.mocked(getMod)).toHaveBeenCalledWith(
      projectId,
      allowedReleaseTypes,
      allowedGameVersion,
      loader,
      allowFallback
    );
  });

  it('calls through to the lookup module', async () => {
    const result = [generatePlatformLookupResult().generated];
    const lookupInput = chance.n(chance.word, chance.integer({ min: 1, max: 20 }));

    vi.mocked(cfLookup).mockResolvedValueOnce(result);
    const curseforge = new Curseforge();
    const actual = await curseforge.lookup(lookupInput);

    expect(vi.mocked(cfLookup)).toHaveBeenCalledWith(lookupInput);
    expect(actual).toBe(result);

  });
});
