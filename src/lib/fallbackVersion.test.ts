import { beforeEach, describe, expect, it } from 'vitest';
import { NextVersionInfo, getNextVersionDown } from './fallbackVersion.js';

describe('The getNextVersionDown function', () => {
  let version: string;
  let expected: NextVersionInfo;

  beforeEach(() => {
    version = '';
    expected = {
      major: 0,
      minor: 0,
      patch: 0,
      nextVersionToTry: '',
      canGoDown: false
    };
  });

  it('should decrease the patch version by 1 when the patch version is greater than 1', () => {
    version = '1.2.3';
    expected = {
      major: 1,
      minor: 2,
      patch: 3,
      nextVersionToTry: '1.2.2',
      canGoDown: true
    };

    const result = getNextVersionDown(version);
    expect(result).toEqual(expected);
  });

  it('should not change the patch version when the patch version is 1 or less', () => {
    version = '1.2.1';
    expected = {
      major: 1,
      minor: 2,
      patch: 1,
      nextVersionToTry: '1.2',
      canGoDown: false
    };

    const result = getNextVersionDown(version);
    expect(result).toEqual(expected);
  });

  it('should decrease the minor version by 1 and set patch to max when the minor version is greater than 1 and patch is 0 or less', () => {
    version = '1.2.0';
    expected = {
      major: 1,
      minor: 2,
      patch: 0,
      nextVersionToTry: '1.2',
      canGoDown: false
    };

    const result = getNextVersionDown(version);
    expect(result).toEqual(expected);
  });

  it('should not change the minor version and set patch to max when the minor version is 1 or less and patch is 0 or less', () => {
    version = '1.1.0';
    expected = {
      major: 1,
      minor: 1,
      patch: 0,
      nextVersionToTry: '1.1',
      canGoDown: false
    };

    const result = getNextVersionDown(version);
    expect(result).toEqual(expected);
  });

  it('should decrease the major version by 1 and set minor and patch to max when the major version is greater than 1, minor and patch are 0 or less', () => {
    version = '2.0';
    expected = {
      major: 2,
      minor: 0,
      patch: 0,
      nextVersionToTry: '2.0',
      canGoDown: false
    };

    const result = getNextVersionDown(version);
    expect(result).toEqual(expected);
  });

  it('should not change the major version and set minor and patch to max when the major version is 1 or less, minor and patch are 0 or less', () => {
    version = '1.0.0';
    expected = {
      major: 1,
      minor: 0,
      patch: 0,
      nextVersionToTry: '1.0',
      canGoDown: false
    };

    const result = getNextVersionDown(version);
    expect(result).toEqual(expected);
  });
});
