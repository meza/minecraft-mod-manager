export class InvalidReleaseTypeException extends Error {
  constructor(releaseType: number) {
    super(`Invalid release type ${releaseType}`);
  }
};
