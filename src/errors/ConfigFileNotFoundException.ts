export class ConfigFileNotFoundException extends Error {
  constructor(configPath: string) {
    super(`Config file not found at ${configPath}`);
  }
}
