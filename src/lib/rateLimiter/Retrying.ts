export class Retrying extends Error {
  private readonly response: Response;
  constructor(lastResponse: Response) {
    super('Retrying call: ' + lastResponse.statusText);
    this.response = lastResponse;
  }

  lastResponse() {
    return this.response;
  }
}
