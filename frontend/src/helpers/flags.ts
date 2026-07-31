import type { Flag } from '@/api/types/device'

const LETTERS: Array<[keyof Flag, string]> = [
  ['read', 'R'],
  ['write', 'W'],
  ['add_object', 'A'],
  ['system', 'X'],
  ['periodic_read', 'P'],
  ['important', 'I'],
  ['send', 'S'],
]

export function flagToString(flag: Flag): string {
  return LETTERS.filter(([key]) => flag[key])
    .map(([, letter]) => letter)
    .join('')
}
