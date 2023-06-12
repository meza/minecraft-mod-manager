export class CurseforgeDownloadUrlError extends Error {
  constructor(modName: string) {
    super(`Curseforge doesn't provide a download url for ${modName}. Try adding the mod from Modrinth instead`);
  }
}
