import { DefaultOptions } from '../mmm.js';
import { Logger } from '../lib/Logger.js';
import { ensureConfiguration, readLockFile } from '../lib/config.js';
import path from 'node:path';
import fs from 'fs/promises';
import { fileIsManaged } from '../lib/configurationHelper.js';
import { shouldPruneFiles } from '../interactions/shouldPruneFiles.js';
import { getModFiles } from '../lib/fileHelper.js';

export interface PruneOptions extends DefaultOptions {
  force: boolean;
}

export const prune = async (options: PruneOptions, logger: Logger) => {
  const configuration = await ensureConfiguration(options.config, logger);
  const installations = await readLockFile(options, logger);
  const modsFolder = path.resolve(configuration.modsFolder);

  const files = await getModFiles(options.config, configuration.modsFolder);

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
};
