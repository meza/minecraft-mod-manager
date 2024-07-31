import { getHWID } from 'hwid';
import { PostHog } from 'posthog-node';
import { posthogApiKey } from '../env.js';
import { ModsJson } from '../lib/modlist.types.js';
import { DefaultOptions } from '../mmm.js';
import { version } from '../version.js';

export interface CommandTelemetry {
  command: string;
  config?: ModsJson;
  arguments: Record<string, unknown> | DefaultOptions;
  success: boolean;
  duration: number;
  error?: string;
  extra?: Record<string, unknown>;
}

export class Telemetry {
  private posthog: PostHog;

  constructor() {
    this.posthog = new PostHog(posthogApiKey, {
      host: 'https://eu.i.posthog.com',
      flushAt: 1
    });

    ['exit', 'SIGINT', 'SIGUSR1', 'SIGUSR2', 'SIGTERM'].forEach((eventType) => {
      process.on(eventType, this.cleanup.bind(this));
    });
  }

  public async capture(event: string, properties?: Record<string, unknown>): Promise<void> {
    const distinctId = await getHWID();
    this.posthog.capture({
      distinctId: distinctId,
      event: event,
      properties: {
        ...properties,
        version: version
      }
    });
  }

  public async captureCommand(properties: CommandTelemetry): Promise<void> {
    if (!properties.command) {
      return;
    }

    const input: Partial<CommandTelemetry> = { ...properties };
    delete input.command;

    await this.capture(properties.command, {
      ...input,
      type: 'command',
      performance: performance.getEntries()
    });
  }

  public async flush(): Promise<void> {
    await this.posthog.shutdown();
  }

  private async cleanup(): Promise<void> {
    await this.posthog.shutdown(0);
  }
}
