import { DefaultOptions, telemetry } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, getModsFolder, readLockFile } from '../lib/config.js';
import path from 'node:path';
import fs from 'fs/promises';
import { fileIsManaged } from '../lib/configurationHelper.js';
import { shouldPruneFiles } from '../interactions/shouldPruneFiles.js';
import { getModFiles } from '../lib/fileHelper.js';

export interface PruneOptions extends DefaultOptions {
  force: boolean;
}

export const prune = async (options: PruneOptions, logger: Logger) => {
  performance.mark('prune-start');
  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options, logger);
  const modsFolder = getModsFolder(options.config, configuration);

  const files = await getModFiles(options.config, configuration);

  if (files.length === 0) {
    logger.log('You have no files in your mods folder.');
    return;
  }

  const unmanaged = files.filter((file) => {
    return !fileIsManaged(file, installations);
  });

  if (unmanaged.length === 0) {
    logger.log('You have no unmanaged mods in your mods folder.');
    return;
  }

  if (!await shouldPruneFiles(options, logger)) {
    return;
  }

  for (const file of unmanaged) {
    const filePath = path.resolve(modsFolder, file);
    await fs.rm(filePath, { force: true });
    logger.log(`Deleted: ${filePath}`);
  }

  performance.mark('prune-succeed');

  await telemetry.captureCommand({
    command: 'prune',
    success: true,
    arguments: {
      options: options
    },
    duration: performance.measure('prune-duration', 'prune-start', 'prune-succeed').duration
  });
};
