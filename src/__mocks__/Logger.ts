import { vi } from 'vitest';
import { Logger as Original } from '../lib/Logger.js';

export const Logger = vi.importMock('../lib/Logger.js') as unknown as Original;
