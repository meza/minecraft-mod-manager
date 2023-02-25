import { PromptModule, Question } from 'inquirer';
import { vi } from 'vitest';

export const findQuestion = (input: PromptModule, questionName: string): Question => {
  const promptQuestions: Question[] = (vi.mocked(input).mock.calls[0][0] as Question[]);
  const question = promptQuestions.find((q: Question) => q.name === questionName);

  if (question) {
    return question;
  }

  throw new Error('Question could not be found in the input');
};
