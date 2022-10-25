export class IncorrectMinecraftVersionException extends Error {
  constructor(version: string) {
    super(`The specified Minecraft version (${version}) is not valid.`);
  }
}
