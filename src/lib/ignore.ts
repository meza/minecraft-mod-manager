import path from 'path';
import { fileExists } from './config.js';
import * as fs from 'fs/promises';
import { glob } from 'glob';

const ignored = async (rootLookupDir: string) => {
  const ignoreFileLocation = path.resolve(rootLookupDir, '.mmmignore');
  const ignoredFiles = new Set<string>();
  if (!(await fileExists(ignoreFileLocation))) {
    return ignoredFiles;
  }

  const patterns = (await fs.readFile(ignoreFileLocation)).toString().split('\n').filter((line) => line.length > 0);

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
  const ignoredFiles = await ignored(rootLookupDir);
  if (ignoredFiles.size === 0) {
    return files;
  }

  return files.filter((file) => {
    return !ignoredFiles.has(file);
  });
};
