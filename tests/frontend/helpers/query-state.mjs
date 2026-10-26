export function waitForQueryState(client, queryKey, predicate) {
  const cache = client.getQueryCache();
  const find = () => cache.find({ queryKey, exact: true });
  const existing = find();
  if (predicate(existing?.state)) return Promise.resolve(existing.state);
  return new Promise((resolve, reject) => {
    const unsubscribe = cache.subscribe((event) => {
      if (event.type !== "updated" || event.query !== find() || !predicate(event.query.state)) return;
      clearTimeout(timeout);
      unsubscribe();
      resolve(event.query.state);
    });
    const timeout = setTimeout(() => {
      unsubscribe();
      reject(new Error("Query did not reach the expected state"));
    }, 3000);
  });
}
