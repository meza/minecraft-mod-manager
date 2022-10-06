export const setup = () => {
  // @ts-ignore
  process.env.CHANCE_SEED = process.env.CHANCE_SEED || '1234';

  // @ts-ignore
  process.env.FORCE_COLOR = 0;
  console.log('Turning colours off in chalk for test consistency');

  // @ts-ignore
  console.log(`Using Chance Seed: ${process.env.CHANCE_SEED}`);
};

