import chalk from 'chalk';
import { Command } from 'commander';

export class Logger {
  private readonly program: Command;
  private quietFlag: boolean;
  private debugFlag: boolean;

  constructor(program: Command) {
    this.program = program;
    this.quietFlag = false;
    this.debugFlag = false;
  }

  flagQuiet() {
    this.quietFlag = true;
  }

  flagDebug() {
    this.debugFlag = true;
  }

  log(message: string, forceShow = false) {
    if (!this.quietFlag || forceShow || this.debugFlag) {
      console.log(message);
    }
  }

  debug(message: string) {
    if (this.debugFlag) {
      console.debug(message);
    }
  }

  error(message: string, code = 1): never {
    this.program.error(chalk.red(message), { exitCode: code });
  }
}
