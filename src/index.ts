import { program } from './mmm.js';
import { hasUpdate } from './lib/mmmVersionCheck.js';
import { version } from './version.js';

hasUpdate(version).then((update) => {
  if (update.hasUpdate) {
    console.log(`There is a new version of MMM available: ${update.latestVersion}`);
    console.log(`You can download it from ${update.latestVersionUrl}`);
  }
  program.parse(process.argv);
});

// TODO test the console logs everywhere
