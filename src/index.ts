import chalk from 'chalk';
import { hasUpdate } from './lib/mmmVersionCheck.js';
import { logger, program, telemetry } from './mmm.js';
import { version } from './version.js';

program.parseAsync(process.argv).then(async () => {
  hasUpdate(version, logger).then((update) => {
    if (update.hasUpdate) {
      logger.log(
        chalk.bgYellowBright(
          chalk.black(`There is a new version of MMM available: ${update.latestVersion} from ${update.releasedOn}`)
        )
      );
      logger.log(chalk.bgYellowBright(chalk.black(`You can download it from ${update.latestVersionUrl}`)));
    }
  });
  await telemetry.flush();
});
