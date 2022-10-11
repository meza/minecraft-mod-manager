import chalk from 'chalk';

const prepareRelease = (release: any) => {
  release.numericVersion = release.tag_name.replace('v', '');
  release.versionParts = release.numericVersion.split('.').map((part: string) => parseInt(part, 10));
  return release;
};

export const githubReleases = async () => {
  const url = 'https://api.github.com/repos/meza/minecraft-mod-manager/releases';
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Github releases not found');
  }
  const json = await response.json();
  const prereleases = json.filter((release: any) => release.prerelease).map(prepareRelease);
  const releases = json.filter((release: any) => !release.prerelease && !release.draft).map(prepareRelease);

  return releases.length > 0 ? releases : prereleases;
};

const isFirstLetterANumber = (input: string) => {
  return (/^\d/).test(input);
};

export const hasUpdate = async (currentVersion: string): Promise<{
  hasUpdate: boolean,
  latestVersion: string,
  latestVersionUrl: string
}> => {

  const releases = await githubReleases();
  const latestVersion = releases[0];

  if (!isFirstLetterANumber(currentVersion)) {
    console.log(chalk.bgYellowBright(chalk.whiteBright('\n[update] You are running a development version of MMM. Please update to the latest release.')));
    console.log(chalk.bgYellowBright(chalk.whiteBright(`[update] You can download it from ${latestVersion.html_url}\n`)));
    return {
      hasUpdate: false,
      latestVersion: latestVersion.tag_name,
      latestVersionUrl: latestVersion.html_url
    };
  }

  const currentVersionParts = currentVersion.split('.').map((part: string) => parseInt(part, 10));
  const latestVersionParts = latestVersion.versionParts;
  if (latestVersionParts[0] > currentVersionParts[0]) {
    return {
      hasUpdate: true,
      latestVersion: latestVersion.tag_name,
      latestVersionUrl: latestVersion.html_url
    };
  }
  if (latestVersionParts[1] > currentVersionParts[1]) {
    return {
      hasUpdate: true,
      latestVersion: latestVersion.tag_name,
      latestVersionUrl: latestVersion.html_url
    };
  }
  if (latestVersionParts[2] > currentVersionParts[2]) {
    return {
      hasUpdate: true,
      latestVersion: latestVersion.tag_name,
      latestVersionUrl: latestVersion.html_url
    };
  }
  return {
    hasUpdate: false,
    latestVersion: latestVersion.tag_name,
    latestVersionUrl: latestVersion.html_url
  };
};
