import { curseForgeApiKey } from '../../env.js';

export const lookup = async (fingerprint: number) => {
  const url = 'https://api.curseforge.com/v1/fingerprints';
  const modSearchResult = await fetch(url, {
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
      'x-api-key': curseForgeApiKey
    },
    method: 'POST',
    body: JSON.stringify({
      'fingerprints': [fingerprint]
    })
  });

  const data = await modSearchResult.json();

  if (data.data.exactMatches.length === 0) {
    throw new Error('Cannot find mod');
  }

  return data.data.exactMatches.at(0).id;

};
