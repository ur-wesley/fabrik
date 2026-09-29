import { describe, expect, test } from 'bun:test';
import { add } from './math';

describe('math', () => {
  test('add', () => {
    expect(add(2, 3)).toBe(5);
  });
});
