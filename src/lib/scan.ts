import { DefaultOptions } from '../mmm.js';
import { Logger } from './Logger.js';
import { readConfigFile } from './config.js';
import path from 'path';
import fs from 'fs/promises';
import md5file from 'md5-file';

import curseforge from '@meza/curseforge-fingerprint';
import { lookup } from '../repositories/curseforge/lookup.js';

export const scan = async (options: DefaultOptions, logger: Logger) => {
  const configuration = await readConfigFile(options.config);

  const modsFolder = path.resolve(configuration.modsFolder);

  const files = await fs.readdir(modsFolder);

  const all = files.map(async (file) => {
    const f = path.resolve(modsFolder, file);
    const fp = curseforge.fingerprint(f);
    try {
      const modId = await lookup(fp);
      console.log({ file: file, curseforge: fp, modrinth: await md5file(f), modId: modId });
    } catch {
      console.log({ file: file, curseforge: fp, modrinth: await md5file(f), modId: 'unknown' });
    }
  });

  await Promise.all(all);

  logger.log('done');

};
