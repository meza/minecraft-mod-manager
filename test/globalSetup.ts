import chanceSetup from 'jest-chance';

export const setup = () => {

  process.env.TZ = 'GMT';

  const chanceSeed = chanceSetup();

  process.env.GITHUB_STEP_SUMMARY += `
  ### Chance Seed

  ${chanceSeed}
`;

  // @ts-ignore
  process.env.FORCE_COLOR = 0;
  console.log('Turning colours off in chalk for test consistency');
};

