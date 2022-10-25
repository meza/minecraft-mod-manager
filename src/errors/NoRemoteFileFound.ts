import { Platform } from '../lib/modlist.types.js';

export class NoRemoteFileFound extends Error {
  public readonly modName: string;
  public readonly platform: Platform;

  constructor(modName: string, platform: Platform) {
    super(`No compatible files were found for the given mod: ${platform}: ${modName}`);
    this.modName = modName;
    this.platform = platform;
  }
}
