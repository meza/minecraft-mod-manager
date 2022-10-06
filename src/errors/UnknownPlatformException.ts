export class UnknownPlatformException extends Error {
  public readonly platform: string;

  constructor(platform: string) {
    super(`Unknown platform: ${platform}`);
    this.platform = platform;
  }
}
