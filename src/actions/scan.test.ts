import { beforeEach, describe, vi, it, expect } from 'vitest';
import { Logger } from '../lib/Logger.js';
import { scan, ScanOptions } from './scan.js';
import { chance } from 'jest-chance';
import { ModInstall, ModsJson, Platform } from '../lib/modlist.types.js';
import { scan as scanLib } from '../lib/scan.js';
import { ensureConfiguration, readLockFile, writeConfigFile, writeLockFile } from '../lib/config.js';
import { generateModsJson } from '../../test/modlistGenerator.js';
import { generateScanResult, ScanResultGeneratorOverrides } from '../../test/generateScanResult.js';
import { shouldAddScanResults } from '../interactions/shouldAddScanResults.js';
import { generateModConfig } from '../../test/modConfigGenerator.js';
import { generateModInstall } from '../../test/modInstallGenerator.js';

interface LocalTestContext {
  logger: Logger;
  options: ScanOptions;
  randomConfiguration: ModsJson;
  randomInstallations: ModInstall[];
}

vi.mock('../lib/Logger.js');
vi.mock('../lib/scan.js');
vi.mock('../lib/config');
vi.mock('../interactions/shouldAddScanResults.js');

const randomModDetails = (): ScanResultGeneratorOverrides => {
  return {
    name: chance.word(),
    platform: chance.pickone(Object.values(Platform)),
    modId: chance.word()
  };
};

describe('The Scan action', () => {
  beforeEach<LocalTestContext>((context) => {
    vi.resetAllMocks();
    context.logger = new Logger({} as never);
    context.options = {
      config: 'config.js',
      quiet: false,
      debug: false,
      add: false,
      prefer: chance.pickone(Object.values(Platform))
    };

    context.randomConfiguration = generateModsJson().generated;
    context.randomInstallations = [];

    vi.mocked(ensureConfiguration).mockResolvedValue(context.randomConfiguration);
    vi.mocked(readLockFile).mockResolvedValueOnce(context.randomInstallations);
  });

  it<LocalTestContext>('correctly reports when there are no unmanaged mods', async ({ options, logger }) => {
    vi.mocked(scanLib).mockResolvedValueOnce([]);

    options.quiet = false;
    await scan(options, logger);

    expect(vi.mocked(logger.log).mock.calls[0][0]).toMatchInlineSnapshot('"You have no unmanaged mods in your mods folder."');

    expect(vi.mocked(writeConfigFile)).not.toHaveBeenCalled();
    expect(vi.mocked(writeLockFile)).not.toHaveBeenCalled();
  });

  it<LocalTestContext>('properly logs found mods', async ({ options, logger }) => {
    const name = chance.word();
    const randomResult = generateScanResult({ name: name }).generated;

    vi.mocked(scanLib).mockResolvedValueOnce([randomResult]);
    vi.mocked(shouldAddScanResults).mockResolvedValueOnce(false);

    await scan(options, logger);

    const logMessage = vi.mocked(logger.log).mock.calls[0];

    expect(logMessage[0]).toContain(name);
    expect(logMessage[0]).toContain('Found unmanaged mod: ');
    expect(logMessage[1]).toBeTruthy(); //we log even when in quiet mode

  });

  it<LocalTestContext>('properly logs all found mods', async ({ options, logger }) => {
    const amountToGenerate = chance.integer({ min: 10, max: 30 });

    const results = [];
    for (let i = 0; i < amountToGenerate; i++) {
      results.push(generateScanResult().generated);
    }

    vi.mocked(scanLib).mockResolvedValueOnce(results);
    vi.mocked(shouldAddScanResults).mockResolvedValueOnce(false);

    await scan(options, logger);
    const logCalls = vi.mocked(logger.log).mock.calls;

    logCalls.forEach((call) => {
      expect(call[0]).toContain(('Found unmanaged mod: '));
    });

    expect(vi.mocked(logger.log)).toHaveBeenCalledTimes(amountToGenerate);

  });

  it<LocalTestContext>('should add the new results', async (context) => {
    /**
     * Set up a config that has one mod and a corresponding install
     */
    const preexistingMod = generateModConfig().generated;
    const preexistingInstall = generateModInstall({
      type: preexistingMod.type,
      id: preexistingMod.id,
      name: preexistingMod.name
    }).generated;

    context.randomConfiguration.mods.push(preexistingMod);
    context.randomInstallations.push(preexistingInstall);

    /**
     * Create 3 scan results
     */
    const details1 = randomModDetails();
    const randomResult1 = generateScanResult(details1).generated;
    const details2 = randomModDetails();
    const randomResult2 = generateScanResult(details2).generated;
    const details3 = randomModDetails();
    const randomResult3 = generateScanResult(details3).generated;

    vi.mocked(scanLib).mockResolvedValueOnce([randomResult1, randomResult2, randomResult3]);
    vi.mocked(shouldAddScanResults).mockResolvedValueOnce(true);

    await scan(context.options, context.logger);

    expect(vi.mocked(writeConfigFile)).toHaveBeenCalledOnce();
    expect(vi.mocked(writeLockFile)).toHaveBeenCalledOnce();

    const writtenConfig = vi.mocked(writeConfigFile).mock.calls[0][0];
    const writtenLock = vi.mocked(writeLockFile).mock.calls[0][0];

    expect(writtenConfig.mods).toContainEqual(preexistingMod);
    expect(writtenConfig.mods).toContainEqual({
      name: details1.name,
      id: details1.modId,
      type: details1.platform
    });
    expect(writtenConfig.mods).toContainEqual({
      name: details2.name,
      id: details2.modId,
      type: details2.platform
    });
    expect(writtenConfig.mods).toContainEqual({
      name: details3.name,
      id: details3.modId,
      type: details3.platform
    });

    expect(writtenLock).toContainEqual(preexistingInstall);
    expect(writtenLock).toContainEqual({
      downloadUrl: randomResult1.localDetails.mod.downloadUrl,
      fileName: randomResult1.localDetails.mod.fileName,
      hash: randomResult1.localDetails.mod.hash,
      id: details1.modId,
      name: details1.name,
      releasedOn: randomResult1.localDetails.mod.releaseDate,
      type: details1.platform
    });
    expect(writtenLock).toContainEqual({
      downloadUrl: randomResult2.localDetails.mod.downloadUrl,
      fileName: randomResult2.localDetails.mod.fileName,
      hash: randomResult2.localDetails.mod.hash,
      id: details2.modId,
      name: details2.name,
      releasedOn: randomResult2.localDetails.mod.releaseDate,
      type: details2.platform
    });
    expect(writtenLock).toContainEqual({
      downloadUrl: randomResult3.localDetails.mod.downloadUrl,
      fileName: randomResult3.localDetails.mod.fileName,
      hash: randomResult3.localDetails.mod.hash,
      id: details3.modId,
      name: details3.name,
      releasedOn: randomResult3.localDetails.mod.releaseDate,
      type: details3.platform
    });

  });
});
