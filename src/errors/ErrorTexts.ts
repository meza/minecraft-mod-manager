import chalk from 'chalk';

export class ErrorTexts {
  public static readonly configNotFound = `Configuration file not found. ${chalk.white('Please run mmm init first.')}`;
}
