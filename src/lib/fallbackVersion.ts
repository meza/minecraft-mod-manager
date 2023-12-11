export interface NextVersionInfo {
  major: number;
  minor: number;
  patch: number;
  nextVersionToTry: string;
  canGoDown: boolean;
}

export const getNextVersionDown = (version: string): NextVersionInfo => {
  const [major, minor, patch] = version.split('.').map((num) => parseInt(num, 10));
  const nextVersionToTry = (patch && patch > 1) ? `${major}.${minor}.${patch - 1}` : `${major}.${minor}`;
  return {
    major: major,
    minor: minor,
    patch: patch || 0,
    nextVersionToTry: nextVersionToTry,
    canGoDown: (patch > 1)
  };
};
