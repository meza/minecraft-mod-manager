import { logger, program } from './mmm.js';
import { hasUpdate } from './lib/mmmVersionCheck.js';
import { version } from './version.js';

hasUpdate(version, logger).then((update) => {
  if (update.hasUpdate) {
    logger.log(`There is a new version of MMM available: ${update.latestVersion} from ${update.releasedOn}`);
    logger.log(`You can download it from ${update.latestVersionUrl}`);
  }
  program.parse(process.argv);
});

// TODO test the console logs everywhere
