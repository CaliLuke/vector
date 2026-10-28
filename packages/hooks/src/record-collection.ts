import { DbClient, collectionOptions, liveQueryCollectionOptions } from "@tanstack/react-db";
import type { Collection, WithVirtualProps } from "@tanstack/react-db";
import type { QueryCollectionUtils } from "@tanstack/query-db-collection";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { skipToken } from "@tanstack/react-query";
import type { Query, QueryClient, QueryFunctionContext, QueryKey, UseQueryOptions } from "@tanstack/react-query";

export type RecordRow<T extends object> = { key: string; position: number; value: T };
export type RecordCollectionOptions<T extends object, TKey extends QueryKey = QueryKey> = UseQueryOptions<
  T[],
  Error,
  T[],
  TKey
>;
type Owner = { id: string; db: DbClient };

// Query removal/recreation is a new ownership boundary, even with the same key.
const owners = new WeakMap<QueryClient, WeakMap<object, Owner>>();
let nextOwnerId = 0;
function databaseFor(client: QueryClient, query: object) {
  let databases = owners.get(client);
  if (!databases) {
    databases = new WeakMap();
    owners.set(client, databases);
    const owned = databases;
    client.getQueryCache().subscribe((event) => {
      if (event.type !== "removed") return;
      const owner = owned.get(event.query);
      owned.delete(event.query);
      // DbClient cleans its dependent live view before retiring source records.
      void owner?.db.cleanup();
    });
  }
  let owner = databases.get(query);
  if (!owner) {
    owner = { id: `records:${++nextOwnerId}`, db: new DbClient({ queryClient: client }) };
    databases.set(query, owner);
  }
  return owner;
}

export function getRecordCollection<T extends object, TKey extends QueryKey>(
  client: QueryClient,
  options: RecordCollectionOptions<T, TKey>,
  getKey: (row: T) => string
): {
  query: Query<T[], Error, T[], TKey>;
  source:
    | Collection<RecordRow<T>, string, QueryCollectionUtils<RecordRow<T>, string, RecordRow<T>, Error>, never>
    | undefined;
  collection: Collection<WithVirtualProps<RecordRow<T>, string>, string | number, {}, never> | undefined;
} {
  const defaults = client.defaultQueryOptions(options);
  const query = client.getQueryCache().build(client, defaults);
  const { queryFn } = defaults;
  const enabled = typeof defaults.enabled === "function" ? defaults.enabled(query) : defaults.enabled !== false;
  if (!enabled || !queryFn || queryFn === skipToken) return { query, collection: undefined, source: undefined };
  const { id, db } = databaseFor(client, query);
  const source = db.collection(
    collectionOptions(`${id}:source`, () => ({
      ...queryCollectionOptions<
        RecordRow<T>,
        (context: QueryFunctionContext<TKey>) => unknown,
        Error,
        TKey,
        string,
        T[]
      >({
        queryClient: client,
        // These endpoints return complete lists; live ordering is local.
        queryKey: () => options.queryKey,
        syncMode: "on-demand",
        // React Query consumers own fetch policy. This observer only synchronizes
        // the canonical cache into the shared DB view.
        enabled: false,
        queryFn,
        select: (data: T[]) =>
          (options.select ? options.select(data) : data).map((value, position) => ({
            key: getKey(value),
            position,
            value,
          })),
        getKey: (row: RecordRow<T>) => row.key,
        // A passive observer must not keep a departed consumer's `static`
        // policy alive and prevent QueryClient invalidation for active readers.
        staleTime: Infinity,
        gcTime: defaults.gcTime,
        retry: defaults.retry,
        retryDelay: defaults.retryDelay,
        networkMode: defaults.networkMode,
        ...(defaults.meta ? { meta: defaults.meta } : {}),
      }),
      // DB uses zero to disable GC. One millisecond releases the adapter after
      // the last view leaves, without shortening the transport cache's lifetime.
      gcTime: 1,
    }))
  );
  const collection = db.collection(
    collectionOptions(`${id}:rows`, () =>
      liveQueryCollectionOptions({
        id: `${id}:rows`,
        query: (q) => q.from({ row: source }).orderBy(({ row }) => row.position, "asc"),
        getKey: (row) => row.key,
        gcTime: 1,
      })
    )
  );
  return { query, source, collection };
}
