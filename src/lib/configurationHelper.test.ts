import { describe, it, expect, beforeEach } from 'vitest';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { Mod, ModInstall, Platform } from './modlist.types.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { fileIsManaged, getInstallation, hasInstallation } from './configurationHelper.js';

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
});
