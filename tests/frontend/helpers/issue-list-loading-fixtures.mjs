/** Hold a native issue-list transport request until the test resolves it. */
export function deferredIssueListRequest() {
  const started = Promise.withResolvers();
  const result = Promise.withResolvers();
  return {
    started,
    result,
    call: (...args) => {
      const [params, options] = args.slice(-2);
      started.resolve({ params, signal: options.signal });
      return result.promise;
    },
  };
}

/** Subscribe to a native list observer and release its QueryClient after the test. */
export function observeIssueList(t, client, read, QueryObserver, issueListOptions) {
  const observer = new QueryObserver(client, issueListOptions(read));
  const unsubscribe = observer.subscribe(() => {});
  t.after(async () => {
    unsubscribe();
    await client.cancelQueries();
    client.clear();
  });
  return observer;
}
