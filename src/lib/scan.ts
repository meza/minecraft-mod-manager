import { DefaultOptions } from '../mmm.js';
import { Logger } from './Logger.js';
import { readConfigFile } from './config.js';
import path from 'path';
import fs from 'fs/promises';
import { hashForMod } from '../lib/murmurhash3.js';

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
import addon from '../addon.cjs';

const expectedFp = 2096795665;

export const scan = async (options: DefaultOptions, logger: Logger) => {
  const configuration = await readConfigFile(options.config);

  const modsFolder = path.resolve(configuration.modsFolder);

  const files = await fs.readdir(modsFolder);

  const all = files.map(async (file) => {
    if (file === 'voicechat-fabric-1.19.2-2.3.26.jar') {
      console.log('working on it... Looking for: ', expectedFp);
      const f = path.resolve(modsFolder, file);
      const fp = addon.hash(f);
      console.log(fp);

      console.log('myhash', await hashForMod(f));

    }

  });

  await Promise.all(all);

  logger.log('done');

};
