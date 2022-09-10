export const setup = () => {
  // @ts-ignore
  process.env.CHANCE_SEED = process.env.CHANCE_SEED || '1234';

  // @ts-ignore
  console.log(`Using Chance Seed: ${process.env.CHANCE_SEED}`);
};

