import { Platform } from '../src/lib/modlist.types.js';
import { chance } from 'jest-chance';

export const generateRandomPlatform = (): Platform => {
  return chance.pickone(Object.values(Platform));
};
