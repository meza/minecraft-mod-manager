import { confirm } from '@inquirer/prompts';

export const shouldCreateConfig = async (configLocation: string): Promise<boolean> => {
  return confirm({
    default: true,
    message: `The config file: (${configLocation}) does not exist. Should we create it?`
  });
};
