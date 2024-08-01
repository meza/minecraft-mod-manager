import { getHWID } from 'hwid';
import { PostHog } from 'posthog-node';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ModsJson } from '../lib/modlist.types.js';
import { Telemetry } from './telemetry.js';

vi.mock('posthog-node');
vi.mock('../env.ts', () => ({ posthogApiKey: 'REPLACED_KEY' }));
vi.mock('hwid');
vi.mock('../version.js', () => ({ version: '1.0.0' }));

describe('Telemetry', () => {
  let telemetry: Telemetry;
  let posthogInstance: PostHog;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetAllMocks();
    posthogInstance = {
      capture: vi.fn(),
      shutdown: vi.fn()
    } as unknown as PostHog;
    vi.mocked(PostHog).mockImplementation(() => posthogInstance);
    vi.mocked(getHWID).mockResolvedValue('test-hwid');
    telemetry = new Telemetry();
  });

  it('should initialize correctly', () => {
    expect(PostHog).toHaveBeenCalledWith('REPLACED_KEY', {
      host: 'https://eu.i.posthog.com',
      flushAt: 1
    });
  });

  it('should capture event correctly', async () => {
    await telemetry.capture('test-event', { key: 'value' });

    expect(posthogInstance.capture).toHaveBeenCalledWith({
      distinctId: 'test-hwid',
      event: 'test-event',
      properties: {
        key: 'value',
        version: '1.0.0'
      }
    });
  });

  it('should capture command correctly', async () => {
    const commandTelemetry = {
      command: 'test-command',
      config: { mod: 'test-mod' } as unknown as ModsJson,
      arguments: { arg1: 'value1' },
      success: true,
      duration: 100,
      error: 'test-error',
      extra: { extraKey: 'extraValue' }
    };

    await telemetry.captureCommand(commandTelemetry);

    expect(posthogInstance.capture).toHaveBeenCalledWith({
      distinctId: 'test-hwid',
      event: 'test-command',
      properties: {
        config: { mod: 'test-mod' },
        arguments: { arg1: 'value1' },
        success: true,
        duration: 100,
        error: 'test-error',
        extra: { extraKey: 'extraValue' },
        type: 'command',
        performance: expect.any(Array),
        version: '1.0.0'
      }
    });
  });

  it('should flush correctly', async () => {
    await telemetry.flush();
    expect(posthogInstance.shutdown).toHaveBeenCalled();
  });

  it('should handle errors in capture method', async () => {
    vi.mocked(posthogInstance).capture.mockImplementationOnce(() => {
      throw new Error('Capture error');
    });

    await expect(telemetry.capture('test-event', { key: 'value' })).rejects.toThrow('Capture error');
  });

  it('should handle errors in captureCommand method', async () => {
    vi.mocked(posthogInstance).capture.mockImplementationOnce(() => {
      throw new Error('Capture command error');
    });

    const commandTelemetry = {
      command: 'test-command',
      config: { mod: 'test-mod' } as unknown as ModsJson,
      arguments: { arg1: 'value1' },
      success: true,
      duration: 100,
      error: 'test-error',
      extra: { extraKey: 'extraValue' }
    };

    await expect(telemetry.captureCommand(commandTelemetry)).rejects.toThrow('Capture command error');
  });

  it('should handle errors in flush method', async () => {
    vi.mocked(posthogInstance).shutdown.mockImplementationOnce(() => {
      throw new Error('Shutdown error');
    });

    await expect(telemetry.flush()).rejects.toThrow('Shutdown error');
  });

  it('should handle edge cases in capture method', async () => {
    await telemetry.capture('', {});

    expect(posthogInstance.capture).toHaveBeenCalledWith({
      distinctId: 'test-hwid',
      event: '',
      properties: {
        version: '1.0.0'
      }
    });
  });

  it('should handle edge cases in captureCommand method', async () => {
    const commandTelemetry = {
      command: '',
      config: {} as unknown as ModsJson,
      arguments: {},
      success: false,
      duration: 0,
      error: '',
      extra: {}
    };

    await telemetry.captureCommand(commandTelemetry);

    expect(posthogInstance.capture).not.toHaveBeenCalled();
  });

  it('should call cleanup on process exit events', async () => {
    // @ts-ignore
    process.emit('exit');
    process.emit('SIGINT');
    process.emit('SIGUSR1');
    process.emit('SIGUSR2');
    process.emit('SIGTERM');

    expect(posthogInstance.shutdown).toHaveBeenCalledTimes(5);
  });
});
