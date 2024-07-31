import { expect, vi } from 'vitest';
import { telemetry } from '../src/mmm.js';
import { CommandTelemetry } from '../src/telemetry/telemetry.js';

vi.mock('../mmm.js');

const getCommandArgsFor = (position: number): CommandTelemetry => {
  return vi.mocked(telemetry.captureCommand).mock.calls[position][0];
};

const getCommandArgs = (): CommandTelemetry => {
  return getCommandArgsFor(0);
};

export const expectCommandStartTelemetry = (expectations: Partial<CommandTelemetry>) => {
  expect(telemetry.captureCommand).toHaveBeenCalled();
  const telemetryCalledWith = getCommandArgs();
  expect(telemetryCalledWith).toMatchObject(expect.objectContaining(expectations));
};

export const expectCommandStartTelemetryNth = (position: number, expectations: Partial<CommandTelemetry>) => {
  expect(telemetry.captureCommand).toHaveBeenCalled();
  const telemetryCalledWith = getCommandArgsFor(position - 1);
  expect(telemetryCalledWith).toMatchObject(expect.objectContaining(expectations));
};
