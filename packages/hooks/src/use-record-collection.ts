import { useCallback, useEffect, useMemo, useSyncExternalStore } from "react";
import { useLiveQuery } from "@tanstack/react-db";
import { QueryObserver, useQuery, useQueryClient } from "@tanstack/react-query";
import type { QueryKey, QueryState } from "@tanstack/react-query";
import { getRecordCollection } from "./record-collection";
import type { RecordCollectionOptions } from "./record-collection";

export type RecordCollectionResult<T extends object> = {
  byId: ReadonlyMap<string, T> | undefined;
  isCurrent: () => boolean;
  get: (key: string | undefined | null) => T | undefined;
  isPending: boolean;
  isError: boolean;
  isLoading: boolean;
  isFetching: boolean;
  fetchStatus: "fetching" | "paused" | "idle";
  error: Error | null;
  dataUpdatedAt: number;
  refetch: (options?: { throwOnError?: boolean }) => Promise<void>;
} & ({ isSuccess: true; data: T[] } | { isSuccess: false; data: T[] | undefined });

function recordId(row: object) {
  if (!("id" in row) || typeof row.id !== "string" || !row.id)
    throw new Error("A collection record requires a stable ID.");
  return row.id;
}

/** Complete record lists have one Query-backed DB collection and live indexes.
 * Native Query observers own per-consumer fetching policy; DB observes passively.
 */
export function useRecordCollection<T extends { id?: string | undefined }, TKey extends QueryKey>(
  options: RecordCollectionOptions<T, TKey>,
  getKey?: (row: T) => string
): RecordCollectionResult<T>;
export function useRecordCollection<T extends object, TKey extends QueryKey>(
  options: RecordCollectionOptions<T, TKey>,
  getKey: (row: T) => string
): RecordCollectionResult<T>;
export function useRecordCollection<T extends object, TKey extends QueryKey>(
  options: RecordCollectionOptions<T, TKey>,
  getKey: (row: T) => string = recordId
) {
  return useKeyedRecordCollection(options, getKey);
}

function useKeyedRecordCollection<T extends object, TKey extends QueryKey>(
  options: RecordCollectionOptions<T, TKey>,
  getKey: (row: T) => string
): RecordCollectionResult<T> {
  const client = useQueryClient();
  useQuery(options);
  const { query, source, collection } = getRecordCollection(client, options, getKey);
  const subscribe = useCallback(
    (notify: () => void) =>
      client.getQueryCache().subscribe((event) => {
        if (event.query === query) notify();
      }),
    [client, query]
  );
  const snapshot = useCallback(
    () => (Object.is(client.getQueryCache().get(query.queryHash), query) ? query.state : undefined),
    [client, query]
  );
  const state = useSyncExternalStore(subscribe, snapshot, snapshot);
  const live = useLiveQuery(() => collection);
  const error = state?.error ?? source?.utils.lastError ?? null;
  const status = collectionStatus(Boolean(collection), live.isReady, state, error);
  const { isSuccess } = status;
  // A failed initial subset retires its live graph. Query may recover through
  // invalidation, a canonical cache write, or Retry; each reopens the same view.
  useEffect(() => {
    if (!collection || live.status !== "error" || state?.status !== "success" || error) return;
    void collection
      .cleanup()
      .then(async () => {
        if (Object.is(client.getQueryCache().get(query.queryHash), query)) await collection.preload();
        return undefined;
      })
      .catch(() => undefined); // Query and live-query error state remain observable.
  }, [client, collection, error, live.status, query, state?.status]);
  const refetch = useCallback(
    async (refetchOptions?: { throwOnError?: boolean }) => {
      if (source && Object.is(client.getQueryCache().get(query.queryHash), query))
        // Bind an imperative refetch to this captured query, not the mounted
        // hook's observer, which may have moved to another scope since capture.
        await new QueryObserver(client, options).refetch(refetchOptions);
    },
    [client, options, query, source]
  );
  // Keep DB metadata outside REST/Form values; both views share the DB order.
  const records = useMemo(() => new Map(live.data?.map((row) => [row.key, row.value])), [live.data]);
  const rows = useMemo(() => live.data?.map((row) => row.value) ?? [], [live.data]);
  const isCurrent = useCallback(
    () =>
      isSuccess &&
      Object.is(client.getQueryCache().get(query.queryHash), query) &&
      query.state.data === state?.data &&
      query.state.status === "success",
    [client, isSuccess, query, state?.data]
  );
  const result = collectionData(isSuccess, Boolean(collection), live.isReady, state?.data, rows);
  return {
    ...result,
    byId: isSuccess ? records : undefined,
    isCurrent,
    get: (key: string | undefined | null) => (isCurrent() && key ? records.get(key) : undefined),
    isPending: status.isPending,
    isError: status.isError,
    isLoading: status.isLoading,
    isFetching: status.isFetching,
    fetchStatus: status.fetchStatus,
    error,
    dataUpdatedAt: state?.dataUpdatedAt ?? 0,
    refetch,
  };
}

function collectionStatus(enabled: boolean, ready: boolean, state: QueryState | undefined, error: Error | null) {
  const isError = state?.status === "error" || error !== null;
  const isSuccess = enabled && state?.status === "success" && ready && !isError;
  const isPending = !isSuccess && !isError;
  const isFetching = state?.fetchStatus === "fetching";
  return {
    isSuccess,
    isPending,
    isError,
    isFetching,
    isLoading: isPending && isFetching,
    fetchStatus: state?.fetchStatus ?? "idle",
  };
}

function collectionData<T extends object>(
  success: boolean,
  enabled: boolean,
  ready: boolean,
  cached: T[] | undefined,
  rows: T[]
) {
  if (success) return { isSuccess: true as const, data: rows };
  const data = enabled ? (cached !== undefined && ready ? rows : undefined) : cached;
  return { isSuccess: false as const, data };
}
