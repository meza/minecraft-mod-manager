export class ConfigFileInvalidError extends Error {
  constructor() {
    super('Config file is invalid');
  }
}
