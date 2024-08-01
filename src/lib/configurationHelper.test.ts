import { chance } from 'jest-chance';
import { beforeEach, describe, expect, it } from 'vitest';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { fileIsManaged, findLocalMods, getInstallation, hasInstallation } from './configurationHelper.js';
import { Mod, ModInstall, Platform } from './modlist.types.js';

interface LocalTestContext {
  installations: ModInstall[];
  mod: Mod;
}

describe('The configuration helper', () => {
  beforeEach<LocalTestContext>((context) => {
    const install1 = generateModInstall({ type: Platform.MODRINTH, id: '1' }).generated;
    const install2 = generateModInstall({ type: Platform.CURSEFORGE, id: '2' }).generated;
    const install3 = generateModInstall({ type: Platform.MODRINTH, id: '3' }).generated;

    context.installations = [install1, install2, install3];
    context.mod = generateModConfig({ type: install2.type, id: install2.id }).generated;
  });

  it<LocalTestContext>('can find an installation', ({ installations, mod }) => {
    const actual = getInstallation(mod, installations);

    expect(actual).toEqual(1);
  });

  it<LocalTestContext>('can answer the question if something has an instalation', ({ installations, mod }) => {
    expect(hasInstallation(mod, installations)).toBeTruthy();

    const unavailableMod = generateModConfig({ id: 'does-not-exist' }).generated;
    expect(hasInstallation(unavailableMod, installations)).toBeFalsy();
  });

  it<LocalTestContext>('can tell if a file is managed by the config', ({ installations }) => {
    expect(fileIsManaged(installations[1].fileName, installations)).toBeTruthy();

    expect(fileIsManaged('does-not-exist.jar', installations)).toBeFalsy();
  });

  describe('when looking up mods', () => {
    it('can find a mod by ID', () => {
      const modId = chance.word();
      const toFind = generateModConfig({ id: modId }).generated;
      const mods = [generateModConfig().generated, toFind, generateModConfig().generated];
      const config = generateModsJson({ mods: mods }).generated;
      const actual = findLocalMods([modId], config);

      expect(actual.size).toEqual(1);
      expect(actual).toContainEqual(toFind);
    });

    it('can find multiple mods by ID', () => {
      const toFind1 = generateModConfig().generated;
      const toFind2 = generateModConfig().generated;
      const mods = [toFind2, generateModConfig().generated, toFind1, generateModConfig().generated];
      const config = generateModsJson({ mods: mods }).generated;
      const actual = findLocalMods([toFind1.id.toUpperCase(), toFind2.id], config);

      expect(actual.size).toEqual(2);
      expect(actual).toContainEqual(toFind1);
      expect(actual).toContainEqual(toFind2);
    });

    it('can find mods based on a pattern', () => {
      const baseName = chance.word();
      const toFind1 = generateModConfig({
        name: baseName + chance.word()
      }).generated;
      const toFind2 = generateModConfig({
        name: baseName + chance.word()
      }).generated;
      const mods = [toFind2, generateModConfig().generated, toFind1, generateModConfig().generated];
      const config = generateModsJson({ mods: mods }).generated;
      const actual = findLocalMods([`${baseName}*`], config);

      expect(actual.size).toEqual(2);
      expect(actual).toContainEqual(toFind1);
      expect(actual).toContainEqual(toFind2);
    });

    it('returns an empty result set when nothing is found', () => {
      const config = generateModsJson().generated;
      const actual = findLocalMods(['something'], config);

      expect(actual.size).toEqual(0);
    });
  });
});
