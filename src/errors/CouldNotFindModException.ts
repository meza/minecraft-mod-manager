import { Platform } from '../lib/modlist.types.js';

export class CouldNotFindModException extends Error {
  public readonly modId: string;
  public readonly platform: Platform;

  constructor(modId: string, platform: Platform) {
    super(`Could not find the given mod: ${platform}: ${modId}`);
    this.modId = modId;
    this.platform = platform;
  }
}
