export class MaximumRetriesReached extends Error {
  private readonly lastResponse: Response;
  constructor(lastResponse: Response) {
    super('Maximum retries reached. Last response status was: ' + lastResponse.statusText);
    this.lastResponse = lastResponse;
  }

  response() {
    return this.lastResponse;
  }
}
