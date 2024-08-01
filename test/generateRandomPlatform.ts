import { chance } from 'jest-chance';
import { Platform } from '../src/lib/modlist.types.js';

export const generateRandomPlatform = (): Platform => {
  return chance.pickone(Object.values(Platform));
};
