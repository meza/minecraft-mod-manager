import * as crypto from 'crypto';
import fs from 'node:fs/promises';
import { fileExists } from './config.js';

export const getHash = async (file: string) => {
  if (!await fileExists(file)) {
    throw new Error(`File (${file}) does not exist, can't determine the hash`);
    // TODO handle file not found
  }

  const contents = await fs.readFile(file);
  const hash = crypto.createHash('sha1');
  hash.update(contents);
  return hash.digest('hex');
};
