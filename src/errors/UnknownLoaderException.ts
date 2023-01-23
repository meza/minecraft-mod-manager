export class UnknownLoaderException extends Error {
  public readonly loader: string;

  constructor(platform: string) {
    super(`Unknown loader: ${platform}`);
    this.loader = platform;
  }
}
