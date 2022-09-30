import { describe, it, expect, vi } from 'vitest';
import { afterEach, beforeEach } from 'vitest';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { readConfigFile } from '../lib/config.js';
import { GeneratorResult } from '../../test/test.types.js';
import { Mod, ModsJson } from '../lib/modlist.types.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { generateModInstall } from '../../test/modInstallGenerator.js';
import { install } from './install.js';
import { fetchModDetails } from '../repositories/index.js';

vi.mock('../lib/config.js');
vi.mock('../repositories/index.js');
vi.mock('../lib/downloader.js');
vi.mock('inquirer');

interface LocalTestContext {
  randomConfiguration: GeneratorResult<ModsJson>;
  randomUninstalledMod: GeneratorResult<Mod>;
  randomInstalledMod: GeneratorResult<Mod>;
  randomOutdatedMod: GeneratorResult<Mod>;
}

const expectModDetailsHaveBeenFetchedCorrectly = (context: LocalTestContext) => {
  expect(vi.mocked(fetchModDetails)).toHaveBeenNthCalledWith(
    1,
    context.randomUninstalledMod.expected.type,
    context.randomUninstalledMod.expected.id,
    context.randomConfiguration.expected.defaultAllowedReleaseTypes,
    context.randomConfiguration.expected.gameVersion,
    context.randomConfiguration.expected.loader,
    context.randomConfiguration.expected.allowVersionFallback
  );

  expect(vi.mocked(fetchModDetails)).toHaveBeenNthCalledWith(
    2,
    context.randomInstalledMod.expected.type,
    context.randomInstalledMod.expected.id,
    context.randomConfiguration.expected.defaultAllowedReleaseTypes,
    context.randomConfiguration.expected.gameVersion,
    context.randomConfiguration.expected.loader,
    context.randomConfiguration.expected.allowVersionFallback
  );

  expect(vi.mocked(fetchModDetails)).toHaveBeenNthCalledWith(
    3,
    context.randomOutdatedMod.expected.type,
    context.randomOutdatedMod.expected.id,
    context.randomConfiguration.expected.defaultAllowedReleaseTypes,
    context.randomConfiguration.expected.gameVersion,
    context.randomConfiguration.expected.loader,
    context.randomConfiguration.expected.allowVersionFallback
  );
};

describe('The install module', () => {

  beforeEach<LocalTestContext>((context) => {
    context.randomConfiguration = generateModsJson();

    context.randomUninstalledMod = generateModConfig();
    context.randomInstalledMod = generateModConfig();
    context.randomOutdatedMod = generateModConfig();

    context.randomInstalledMod.generated.installed = generateModInstall().generated;
    context.randomOutdatedMod.generated.installed = generateModInstall({ hash: 'outdated' }).generated;

    context.randomConfiguration.generated.mods = [
      context.randomUninstalledMod.generated,
      context.randomInstalledMod.generated,
      context.randomOutdatedMod.generated
    ];

    // the main configuration to work with
    vi.mocked(readConfigFile).mockResolvedValue(context.randomConfiguration.generated);

  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it<LocalTestContext>('gets the details for all mods in the list', async (context) => {
    await install({ config: 'config.json' });

    expectModDetailsHaveBeenFetchedCorrectly(context);

  });

  //iterate through modlist
  //if installation field
  //  if hash is different, download
  //  if file is missing, download


});
