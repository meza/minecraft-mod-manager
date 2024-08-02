export class Queue<T> {
  private data: T[] = [];

  get length() {
    return this.data.length;
  }

  enqueue(queueItem: T) {
    this.data.push(queueItem);
  }

  dequeue(): T | undefined {
    return this.data.shift();
  }

  peek() {
    if (this.isEmpty()) {
      return undefined;
    }
    return this.data[0];
  }

  size() {
    return this.length;
  }

  isEmpty() {
    return this.length === 0;
  }

  clear() {
    this.data = [];
  }
}
