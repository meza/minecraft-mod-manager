import { IncorrectMinecraftVersionException } from '../errors/IncorrectMinecraftVersionException.js';
import { RedundantVersionException } from '../errors/RedundantVersionException.js';
import { DefaultOptions } from '../mmm.js';
import { fetchModDetails } from '../repositories/index.js';
import { readConfigFile } from './config.js';
import { Logger } from './Logger.js';
import { verifyMinecraftVersion } from './minecraftVersionVerifier.js';
import { Mod } from './modlist.types.js';
import { getLatestMinecraftVersion } from '../interactions/getLatestMinecraftVersion.js';

export type VerifyUpgradeOptions = DefaultOptions & {
  force?: boolean;
};

export interface UpgradeVerificationResult {
  canUpgrade: boolean;
  version: string;
  modsInError: Mod[];
}

export const verifyUpgradeIsPossible = async (gameVersion: string, options: VerifyUpgradeOptions, logger: Logger): Promise<UpgradeVerificationResult> => {
  let version = gameVersion;

  if (gameVersion.toLowerCase() === 'latest') {
    version = await getLatestMinecraftVersion(options, logger);
  }

  const isValidVersion = await verifyMinecraftVersion(version);
  if (!isValidVersion) {
    throw new IncorrectMinecraftVersionException(version);
  }

  const configuration = await readConfigFile(options.config);

  if (configuration.gameVersion === version) {
    throw new RedundantVersionException(version);
  }

  const mods = configuration.mods;
  const errors: Mod[] = [];

  if (options.force) {
    return {
      canUpgrade: true,
      version: version,
      modsInError: []
    };
  }

  const processMod = async (mod: Mod) => {

    logger.debug(`Checking ${mod.name} for ${mod.type} for ${version}`);
    try {
      await fetchModDetails(
        mod.type,
        mod.id,
        mod.allowedReleaseTypes || configuration.defaultAllowedReleaseTypes,
        version,
        configuration.loader,
        !!mod.allowVersionFallback
      );
    } catch {
      errors.push(mod);
    }
    return;

  };
  const promises = mods.map(processMod);

  await Promise.allSettled(promises);

  return {
    canUpgrade: errors.length === 0,
    version: version,
    modsInError: errors
  };
};
