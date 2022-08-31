import { DefaultOptions } from '../index.js';
import { readConfigFile } from './config.js';

export interface ListOptions extends DefaultOptions {}

export const list = async (options: ListOptions) => {
  await readConfigFile(options.config);

};
