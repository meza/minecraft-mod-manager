export class DownloadFailedException extends Error {
  constructor(url: string) {
    super(`Error downloading file: "${url}" please try again`);
  }
}
