export function parentDir(path: string): string {
  const slash = path.lastIndexOf('/');
  return slash === -1 ? '' : path.slice(0, slash);
}

// Groups by folder in order of first appearance and keeps git's order within a folder, so a
// grouped list, its Tab order and index-based ↑/↓ all walk the files in the same order.
export function orderByFolder<T>(items: T[], pathOf: (item: T) => string): T[] {
  const groups = new Map<string, T[]>();
  for (const item of items) {
    const dir = parentDir(pathOf(item));
    const group = groups.get(dir);
    if (group) {
      group.push(item);
    } else {
      groups.set(dir, [item]);
    }
  }
  return [...groups.values()].flat();
}
