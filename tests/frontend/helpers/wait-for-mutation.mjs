export function waitForMutation(client, predicate) {
  const cache = client.getMutationCache();
  const find = () => cache.getAll().find(predicate);
  const existing = find();
  if (existing) return Promise.resolve(existing);
  return new Promise((resolve, reject) => {
    const unsubscribe = cache.subscribe(() => {
      const mutation = find();
      if (!mutation) return;
      clearTimeout(timeout);
      unsubscribe();
      resolve(mutation);
    });
    const timeout = setTimeout(() => {
      unsubscribe();
      reject(new Error("Mutation did not reach the expected state"));
    }, 3000);
  });
}
