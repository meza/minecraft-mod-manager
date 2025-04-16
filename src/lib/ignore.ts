import path from 'path';
import * as fs from 'fs/promises';
import { glob } from 'glob';
import { fileExists } from './config.js';

const ignored = async (rootLookupDir: string) => {
  const ignoreFileLocation = path.resolve(rootLookupDir, '.mmmignore');
  const ignoredFiles = new Set<string>();
  let ignoreFileContents: string[] = [];

  if (await fileExists(ignoreFileLocation)) {
    ignoreFileContents = (await fs.readFile(ignoreFileLocation))
      .toString()
      .split('\n')
      .filter((line) => line.length > 0);
  }

  const patterns = ['**/*.disabled', ...ignoreFileContents];

  patterns.forEach((pattern) => {
    const result = glob.sync(pattern, {
      cwd: rootLookupDir,
      absolute: true
    });

    result.forEach((ignored) => {
      ignoredFiles.add(ignored);
    });
  });
  return ignoredFiles;
};

export const notIgnored = async (rootLookupDir: string, files: string[]): Promise<string[]> => {
  const jarFiles = files.filter((file) => {
    return file.slice(-4) === '.jar';
  });
  const ignoredFiles = await ignored(rootLookupDir);

  if (ignoredFiles.size === 0) {
    return jarFiles;
  }

  return jarFiles.filter((file) => {
    return !ignoredFiles.has(file);
  });
};
