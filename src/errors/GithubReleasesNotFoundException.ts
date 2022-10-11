export class GithubReleasesNotFoundException extends Error {
  constructor() {
    super('Github releases not found');
  }
}
