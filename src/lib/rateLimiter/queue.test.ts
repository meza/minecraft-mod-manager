import { describe, it, expect } from 'vitest';
import { Queue } from './queue.js';
import { chance } from 'jest-chance';

interface TestType {
  prop: boolean;
}

describe('The queue', () => {
  it('can be instantiated', () => {
    const q = new Queue<TestType>();
    expect(q).not.toBeNull();
  });

  it('has an initial size of 0', () => {
    const q = new Queue<TestType>();
    expect(q.length).toEqual(0);
    expect(q.size()).toEqual(0);
  });

  it('can report isEmpty for a new queue', () => {
    const q = new Queue<TestType>();
    expect(q.isEmpty()).toBeTruthy();
  });

  it('can accept an item', () => {
    const q = new Queue<TestType>();
    q.enqueue({ prop: chance.bool() });

    expect(q.length).toEqual(1);
    expect(q.size()).toEqual(1);
    expect(q.isEmpty()).toBeFalsy();
  });

  it('can return an item', () => {
    const q = new Queue<TestType>();
    const item = { prop: chance.bool() };
    q.enqueue(item);

    const actual = q.dequeue();

    expect(actual).toBe(item);

    expect(q.length).toEqual(0);
    expect(q.size()).toEqual(0);
    expect(q.isEmpty()).toBeTruthy();

  });

  it('can peek the top item', () => {
    const q = new Queue<TestType>();
    const item1 = { prop: chance.bool() };
    const item2 = { prop: chance.bool() };
    q.enqueue(item1);
    q.enqueue(item2);

    const actual = q.peek();

    expect(actual).toBe(item1);
  });

  it('returns undefined when peeking an empty queue', () => {
    const q = new Queue<TestType>();
    expect(q.peek()).toBeUndefined();
  });

  it('can handle larger sizes', () => {
    const size = chance.integer({ min: 10, max: 100 });
    const q = new Queue<TestType>();
    for (let i = 0; i < size; i++) {
      q.enqueue({ prop: chance.bool() });
    }

    expect(q.length).toEqual(size);
    expect(q.size()).toEqual(size);

  });

  it('can clear the queue', () => {
    const size = chance.integer({ min: 10, max: 100 });
    const q = new Queue<TestType>();
    for (let i = 0; i < size; i++) {
      q.enqueue({ prop: chance.bool() });
    }

    expect(q.isEmpty()).toBeFalsy();

    q.clear();

    expect(q.length).toEqual(0);
    expect(q.size()).toEqual(0);
    expect(q.isEmpty()).toBeTruthy();

  });
});
