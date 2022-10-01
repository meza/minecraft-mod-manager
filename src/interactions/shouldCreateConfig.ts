import inquirer from 'inquirer';

export const shouldCreateConfig = async (configLocation: string): Promise<boolean> => {
  const answers = await inquirer.prompt([
    {
      type: 'confirm',
      name: 'create',
      default: false,
      message: `The config file: (${configLocation}) does not exist. Should we create it?`
    }
  ]);
  return answers.create;
};
