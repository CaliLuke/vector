import { useEffect, useMemo, useRef } from "react";
import { useMutation, useMutationState, useQueryClient } from "@tanstack/react-query";
import type { Mutation, MutationCache, MutationFilters } from "@tanstack/react-query";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { captureIdentityScope, isIdentityScopeCurrent, isIdentityScopeVerified } from "@/queries/identity";
import { workItemTypeRemovalService } from "@/services/work-item-type-removal.service";
import type { ReviewedTypeRemoval, TypeRemovalCommand } from "@/services/work-item-type-removal.service";

import { isTypeRemovalQuery } from "./type-removal-cache";

type RemovalMutation = Mutation<void, unknown, ReviewedTypeRemoval>;
type Snapshot = Pick<RemovalMutation, "mutationId" | "state">;
export function canReviewTypeRemovalAgain(error: unknown) {
  if (!error || typeof error !== "object" || !("response" in error)) return false;
  const response = error.response;
  if (!response || typeof response !== "object" || !("data" in response)) return false;
  const data = response.data;
  return !!data && typeof data === "object" && "review_again" in data && data.review_again === true;
}
function latestAttempt<T extends { mutationId: number }>(entries: T[]) {
  return entries.reduce<T | undefined>(
    (latest, entry) => (!latest || entry.mutationId > latest.mutationId ? entry : latest),
    undefined
  );
}

function cachedAttempt(cache: MutationCache, filters: MutationFilters) {
  const id = latestAttempt(cache.findAll(filters))?.mutationId;
  return cache.find<void, unknown, ReviewedTypeRemoval>({ ...filters, predicate: (entry) => entry.mutationId === id });
}

/** Retain exact deletion receipts until acknowledged or proven uncommitted. */
export function useTypeRemoval(
  workspaceSlug: string,
  typeId: string,
  operations: Pick<typeof workItemTypeRemovalService, "preview" | "commit"> = workItemTypeRemovalService
) {
  const client = useQueryClient();
  const userId = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const scope = useMemo(
    () => ({ workspaceSlug, typeId, userId, generation }),
    [workspaceSlug, typeId, userId, generation]
  );
  const active = useRef<typeof scope | null>(null);
  useEffect(() => {
    active.current = scope;
    return () => {
      if (active.current === scope) active.current = null;
    };
  }, [scope]);
  const identity = captureIdentityScope(client, userId);
  const isCurrent = () => active.current === scope && isIdentityScopeCurrent(client, identity);
  const ensureCurrent = () => {
    if (!isCurrent()) throw new Error("The account session changed or this dialog was closed.");
  };
  const ensureVerified = () => {
    if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
  };
  const mutationKey = ["type-removal", userId, generation, workspaceSlug, typeId] as const;
  const filters = { mutationKey, exact: true };
  const snapshots = useMutationState<Snapshot, RemovalMutation>({
    filters,
    select: (entry) => ({ mutationId: entry.mutationId, state: entry.state }),
  });
  const latest = latestAttempt(snapshots);
  const retained = latest && !canReviewTypeRemovalAgain(latest.state.error) ? latest.state.variables : undefined;
  const preview = useMutation({
    retry: false,
    mutationFn: (input: TypeRemovalCommand) => {
      ensureCurrent();
      ensureVerified();
      return operations.preview(workspaceSlug, input, () => {
        ensureCurrent();
        ensureVerified();
      });
    },
  });
  const commit = useMutation<void, unknown, ReviewedTypeRemoval>({
    mutationKey,
    gcTime: Infinity,
    retry: false,
    mutationFn: async ({ command }) => {
      ensureCurrent();
      ensureVerified();
      const queries = client
        .getQueryCache()
        .findAll({ predicate: (query) => isTypeRemovalQuery(query.queryKey, workspaceSlug, userId) });
      await operations.commit(workspaceSlug, command, () => {
        ensureCurrent();
        ensureVerified();
      });
      if (isIdentityScopeCurrent(client, identity)) {
        await client.invalidateQueries({ predicate: (query) => queries.includes(query) });
      }
    },
  });
  return {
    retained,
    rejected: latest && canReviewTypeRemovalAgain(latest.state.error) ? latest.state.variables : undefined,
    isCurrent,
    error: latest?.state.error,
    busy: preview.isPending || latest?.state.status === "pending",
    preview: async (input: TypeRemovalCommand) => {
      ensureCurrent();
      if (input.type_id !== typeId) throw new Error("The selected type changed. Reopen its deletion dialog.");
      const cache = client.getMutationCache();
      const previous = cachedAttempt(cache, filters);
      if (previous && !canReviewTypeRemovalAgain(previous.state.error))
        throw new Error("Retry the original deletion before reviewing another replacement.");
      for (const entry of cache.findAll(filters)) cache.remove(entry);
      const result = await preview.mutateAsync(structuredClone(input));
      ensureCurrent();
      ensureVerified();
      return result;
    },
    commit: async (input: ReviewedTypeRemoval) => {
      ensureCurrent();
      if (input.command.type_id !== typeId) throw new Error("The selected type changed. Reopen its deletion dialog.");
      const previous = cachedAttempt(client.getMutationCache(), filters);
      if (previous?.state.status === "pending") throw new Error("Deletion is already in progress.");
      if (previous?.state.status === "success") return;
      const saved = previous && !canReviewTypeRemovalAgain(previous.state.error) ? previous.state.variables : undefined;
      await commit.mutateAsync(structuredClone(saved ?? input));
      ensureCurrent();
    },
    acknowledge: () => {
      ensureCurrent();
      const cache = client.getMutationCache();
      for (const entry of cache.findAll(filters)) cache.remove(entry);
    },
  };
}
