import chalk from 'chalk';
import { GithubReleasesNotFoundException } from '../errors/GithubReleasesNotFoundException.js';
import { Logger } from './Logger.js';

const prepareRelease = (release: any) => {
  release.numericVersion = release.tag_name.replace('v', '');
  release.versionParts = release.numericVersion.split('.').map((part: string) => parseInt(part, 10));
  return release;
};

const githubReleases = async () => {
  const url = 'https://api.github.com/repos/meza/minecraft-mod-manager/releases';
  const response = await fetch(url);
  if (!response.ok) {
    throw new GithubReleasesNotFoundException();
    // TODO handle failed fetch
  }
  const json = await response.json();
  const prereleases = json.filter((release: any) => release.prerelease).map(prepareRelease);
  const releases = json.filter((release: any) => !release.prerelease && !release.draft).map(prepareRelease);

  return releases.length > 0 ? releases : prereleases;
};

const isFirstLetterANumber = (input: string) => {
  return (/^\d/).test(input);
};

const formatDateFromTimeString = (timeString: string) => {
  const date = new Date(timeString);
  return date.toString();
};

export const hasUpdate = async (currentVersion: string, logger: Logger): Promise<{
  hasUpdate: boolean,
  latestVersion: string,
  latestVersionUrl: string,
  releasedOn: string
}> => {

  const releases = await githubReleases();
  const latestVersion = releases[0];
  const releasedOn = formatDateFromTimeString(latestVersion.published_at);
  if (!isFirstLetterANumber(currentVersion)) {
    logger.log(chalk.bgYellowBright(chalk.black(`\n[update] You are running a development version of MMM. Please update to the latest release from ${releasedOn}.`)));
    logger.log(chalk.bgYellowBright(chalk.black(`[update] You can download it from ${latestVersion.html_url}\n`)));
    // Todo move console up one level
    return {
      hasUpdate: false,
      latestVersion: latestVersion.tag_name,
      latestVersionUrl: latestVersion.html_url,
      releasedOn: releasedOn
    };
  }

  const currentVersionParts = currentVersion.split('.').map((part: string) => parseInt(part, 10));
  const latestVersionParts = latestVersion.versionParts;
  if (latestVersionParts[0] > currentVersionParts[0]) {
    return {
      hasUpdate: true,
      latestVersion: latestVersion.tag_name,
      latestVersionUrl: latestVersion.html_url,
      releasedOn: formatDateFromTimeString(latestVersion.published_at)
    };
  }
  if (latestVersionParts[1] > currentVersionParts[1]) {
    return {
      hasUpdate: true,
      latestVersion: latestVersion.tag_name,
      latestVersionUrl: latestVersion.html_url,
      releasedOn: formatDateFromTimeString(latestVersion.published_at)
    };
  }
  if (latestVersionParts[2] > currentVersionParts[2]) {
    return {
      hasUpdate: true,
      latestVersion: latestVersion.tag_name,
      latestVersionUrl: latestVersion.html_url,
      releasedOn: formatDateFromTimeString(latestVersion.published_at)
    };
  }
  return {
    hasUpdate: false,
    latestVersion: latestVersion.tag_name,
    latestVersionUrl: latestVersion.html_url,
    releasedOn: formatDateFromTimeString(latestVersion.published_at)
  };
};
