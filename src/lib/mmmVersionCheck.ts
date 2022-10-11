export const githubReleases = async () => {
  const url = 'https://api.github.com/repos/meza/minecraft-mod-manager/releases';
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Github releases not found');
  }
  const json = await response.json();
  //const prereleases = json.filter((release: any) => release.prerelease);
  const releases = json.filter((release: any) => !release.prerelease && !release.draft)
    .map((release: any) => {
      release.numericVersion = release.tag_name.replace('v', '');
      release.versionParts = release.numericVersion.split('.').map((part: string) => parseInt(part, 10));
      return release;
    });

  return releases;
};

const isFirstLetterANumber = (input: string) => {
  return (/^\d/).test(input);
};

export const hasUpdate = async (currentVersion: string): Promise<boolean> => {
  if (!isFirstLetterANumber(currentVersion)) {
    console.log('You are running a development version of MMM. Skipping update check.');
    return false;
  }

  const releases = await githubReleases();
  const latestVersion = releases[0];
  const currentVersionParts = currentVersion.split('.').map((part: string) => parseInt(part, 10));
  const latestVersionParts = latestVersion.versionParts;
  if (latestVersionParts[0] > currentVersionParts[0]) {
    return true;
  }
  if (latestVersionParts[1] > currentVersionParts[1]) {
    return true;
  }
  if (latestVersionParts[2] > currentVersionParts[2]) {
    return true;
  }
  return false;
};
