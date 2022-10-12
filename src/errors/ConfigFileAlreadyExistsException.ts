export class ConfigFileAlreadyExistsException extends Error {
  constructor(configFile: string) {
    super(`Config file already exists: ${configFile}`);
  }
}
