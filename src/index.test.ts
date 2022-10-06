import { describe, expect, it, vi } from 'vitest';
import { program } from './mmm.js';

vi.mock('./mmm.js');

describe('The main entry point', () => {
  it('calls the main program', async () => {
    await import('./index.js');
    // kinda bogus test but at least it makes sure that we don't forget to call the commander thing
    expect(vi.mocked(program.parse)).toHaveBeenCalledWith(process.argv);
  });
});
