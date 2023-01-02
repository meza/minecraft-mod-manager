export class RedundantVersionException extends Error {
  constructor(version: string) {
    super(`You're already using (${version}).`);
  }
}
