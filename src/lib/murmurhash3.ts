import fs from 'node:fs/promises';

const isWhitespaceCharacter = (b: number) => {
  return b === 9 || b === 10 || b === 13 || b === 32;
};

const toUint = (input: number) => {
  return Uint32Array.from([input])[0];
};

const u32mul = (x: number, y: number) => Number((BigInt(x) * BigInt(y)) & 0xFFFFFFFFn);

const computeNormalizedLength = (buffer: Buffer) => {
  let num1 = 0;
  const length = buffer.length;

  for (let i = 0; i < length; ++i) {
    if (!isWhitespaceCharacter(buffer[i])) {
      ++num1;
    }
  }

  return num1;
};

const computeHash = (fileContents: Buffer) => {
  //num1 OK, length OK

  const multiplex = 1540483477;
  const length = fileContents.length;
  const num1 = computeNormalizedLength(fileContents);
  const testI = 32;

  let num2 = 1 ^ num1;
  let num3 = 0;
  let num4 = 0;

  for (let i = 0; i < length; ++i) {
    const b = fileContents[i];

    if (!isWhitespaceCharacter(b)) {
      num3 |= b << num4;
      num4 += 8;

      if (num4 === 32) {
        const num6 = toUint(u32mul(num3, multiplex));
        const num7 = toUint(u32mul((num6 ^ num6 >> 24), multiplex));
        num2 = toUint(u32mul(num2, multiplex) ^ num7);
        num3 = 0;
        num4 = 0;
      }

      if (i === testI) {
        console.log('b', {
          num1: num1,
          num2: num2,
          num3: num3,
          num4: num4
        });
      }

    }
  }

  if (num4 > 0) {
    num2 = u32mul((num2 ^ num3), multiplex);
  }

  const num6 = u32mul((num2 ^ num2 >> 13), multiplex);

  return toUint(num6 ^ num6 >> 15);

};

export const hashForMod = async (filePath: string): Promise<number> => {
  const contents = await fs.readFile(filePath, { flag: 'r' });
  return computeHash(contents);
};
