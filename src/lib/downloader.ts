import fs from 'node:fs/promises';
import path from 'path';
import { fileExists } from './config.js';

export const downloadFile = async (url: string, destination: string) => {
  try {

    if (await fileExists(destination)) {
      return;
    }

    const dirname = path.dirname(destination);
    await fs.mkdir(dirname, { recursive: true });

    const file = await fs.open(destination, 'w');

    const response = await fetch(url);
    const buff = await response.arrayBuffer();

    await file.write(buff.toString());
  } catch (error) {
    throw new Error('Could not download mod');
  }
};
