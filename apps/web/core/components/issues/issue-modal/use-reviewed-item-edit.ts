import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useMutationState, useQueryClient } from "@tanstack/react-query";
import type { TIssue } from "@plane/types";
import type { Mutation } from "@tanstack/react-query";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { capturePermissionChanges } from "@/queries/permission-mutations";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import type {
  ItemEditCommand,
  ItemEditCommit,
  ItemEditReview,
  SavedItemEdit,
} from "@/services/work-item-edits.service";

type Attempt = { command: ItemEditCommit; recovering: boolean; draft?: Partial<TIssue>; checkAccess?: () => void };
type Operations = {
  captureAccess?: () => () => void;
  preview: (command: ItemEditCommand, beforeSend: () => void) => Promise<ItemEditReview>;
  commit: (command: ItemEditCommit, beforeSend: () => void) => Promise<SavedItemEdit>;
};
type EditMutation = Mutation<SavedItemEdit, unknown, Attempt>;
type Snapshot = Pick<EditMutation, "mutationId" | "state">;
export class ItemEditCancelled extends Error {
  constructor() {
    super("Changes were not saved.");
  }
}
function retainsAttempt(entry: Snapshot | undefined) {
  if (!entry) return false;
  const { status, error, variables } = entry.state;
  return (
    status === "pending" ||
    status === "success" ||
    (status === "error" && !canReviewAgain(error) && (!!variables?.recovering || !isRejectedEdit(error)))
  );
}
function canReviewAgain(error: unknown) {
  if (!error || typeof error !== "object") return false;
  const failure = error as { response?: { data?: { fields?: { review_again?: boolean } } } };
  return failure.response?.data?.fields?.review_again === true;
}
function isRejectedEdit(error: unknown) {
  if (!error || typeof error !== "object") return false;
  const failure = error as { status?: number; response?: { status?: number } };
  const status = failure.response?.status ?? failure.status;
  return status !== undefined && [400, 401, 403, 404, 409, 422, 429].includes(status);
}
function retainedCommand(entry: Snapshot | undefined) {
  return retainsAttempt(entry) ? entry?.state.variables?.command : undefined;
}
function recoveryCommand(entry: Snapshot | undefined, startedHere: boolean) {
  if (!entry || (entry.state.status === "pending" && !entry.state.variables?.recovering && startedHere))
    return undefined;
  return retainedCommand(entry);
}
function latestSnapshot(history: Snapshot[]) {
  return history.reduce<Snapshot | undefined>(
    (result, entry) => (!result || entry.mutationId > result.mutationId ? entry : result),
    undefined
  );
}

/** Keep the exact reviewed command after an uncertain response, including across
 * modal unmounts. A retry cannot silently save a different draft or repeat effects. */
export function useReviewedItemEdit(workspaceSlug: string, itemId: string | undefined, operations: Operations) {
  const client = useQueryClient();
  const userId = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const scope = useMemo(
    () => ({ client, userId, generation, workspaceSlug, itemId }),
    [client, userId, generation, workspaceSlug, itemId]
  );
  const activeScope = useRef<typeof scope | null>(null);
  const [startedHere, setStartedHere] = useState(false);
  const [review, setReview] = useState<{
    value: ItemEditReview;
    command: ItemEditCommand;
    scope: typeof scope;
    checkAccess?: () => void;
  }>();
  const confirmation = useRef<{ scope: typeof scope; resolve: (accepted: boolean) => void } | undefined>(undefined);
  const inFlight = useRef(false);
  useEffect(() => {
    activeScope.current = scope;
    return () => {
      if (activeScope.current === scope) activeScope.current = null;
      if (confirmation.current?.scope === scope) confirmation.current.resolve(false);
    };
  }, [scope]);
  const identity = captureIdentityScope(client, userId);
  const isCurrent = () =>
    activeScope.current === scope &&
    generation === getIdentitySessionGeneration(client) &&
    isIdentityScopeCurrent(client, identity);
  const ensureCurrent = () => {
    if (!isCurrent()) throw new Error("The account session has changed or this editor was closed.");
  };
  const ensureVerified = () => {
    if (!isIdentityScopeVerified(client, identity))
      throw new Error("The account session has changed or is unverified.");
  };
  const mutationKey = ["reviewed-item-edit", userId, generation, workspaceSlug, itemId] as const;
  const filters = { mutationKey, exact: true };
  const history = useMutationState<Snapshot, EditMutation>({
    filters,
    select: (entry) => ({ mutationId: entry.mutationId, state: entry.state }),
  });
  const latest = latestSnapshot(history);
  const preview = useMutation({
    mutationFn: ({ command, checkAccess }: { command: ItemEditCommand; checkAccess?: () => void }) => {
      ensureCurrent();
      ensureVerified();
      return operations
        .preview(command, () => {
          ensureCurrent();
          ensureVerified();
          checkAccess?.();
        })
        .then((result) => {
          ensureCurrent();
          ensureVerified();
          checkAccess?.();
          return result;
        });
    },
    retry: false,
  });
  const commit = useMutation<SavedItemEdit, unknown, Attempt>({
    mutationKey,
    gcTime: Infinity,
    retry: false,
    mutationFn: async ({ command, checkAccess }) => {
      ensureCurrent();
      ensureVerified();
      const changes = capturePermissionChanges(client, userId, workspaceSlug);
      try {
        const result = await operations.commit(command, () => {
          ensureCurrent();
          ensureVerified();
          checkAccess?.();
        });
        void changes.refreshWorkspace();
        return result;
      } catch (error) {
        if (canReviewAgain(error) && isCurrent()) {
          await client.invalidateQueries({ queryKey: ["workspace", workspaceSlug, "work-item-schema", userId] });
        }
        throw error;
      }
    },
  });
  const decide = (accepted: boolean) => {
    confirmation.current?.resolve(accepted);
    confirmation.current = undefined;
    setReview(undefined);
  };
  useEffect(() => {
    if (!review) return;
    try {
      review.checkAccess?.();
    } catch {
      confirmation.current?.resolve(false);
    }
  });
  const reviewCommand = async (
    input: ItemEditCommand,
    checkAccess: (() => void) | undefined
  ): Promise<ItemEditCommit> => {
    if (!itemId || input.item_id !== itemId) throw new Error("Select a work item to save.");
    const requested = structuredClone(input);
    const result = await preview.mutateAsync({ command: requested, checkAccess });
    ensureCurrent();
    checkAccess?.();
    if (result.preserved_properties > 0) {
      const accepted = await new Promise<boolean>((resolve) => {
        confirmation.current = { scope, resolve };
        setReview({ value: result, command: requested, scope, checkAccess });
      });
      ensureCurrent();
      if (!accepted) throw new ItemEditCancelled();
    }
    return { ...requested, review_id: result.id };
  };
  const submitWithReceipt = async (input: ItemEditCommand, draft?: Partial<TIssue>) => {
    ensureCurrent();
    const checkAccess = operations.captureAccess?.();
    checkAccess?.();
    const cache = client.getMutationCache();
    const latestId = cache.findAll(filters).reduce((id, entry) => Math.max(id, entry.mutationId), -1);
    const previous = cache.find<SavedItemEdit, unknown, Attempt>({
      ...filters,
      predicate: (entry) => entry.mutationId === latestId,
    });
    if (inFlight.current || previous?.state.status === "pending") throw new Error("Saving is already in progress.");
    inFlight.current = true;
    try {
      const retained = retainedCommand(previous);
      let command = retained;
      if (!command) command = await reviewCommand(input, checkAccess);
      ensureCurrent();
      checkAccess?.();
      setStartedHere(true);
      const saved = await commit.mutateAsync({
        command,
        checkAccess,
        recovering: !!retained,
        draft: retained ? previous?.state.variables?.draft : structuredClone(draft),
      });
      ensureCurrent();
      return { saved, command };
    } finally {
      confirmation.current = undefined;
      setReview(undefined);
      inFlight.current = false;
    }
  };
  return {
    submit: async (input: ItemEditCommand, draft?: Partial<TIssue>) => (await submitWithReceipt(input, draft)).saved,
    submitWithReceipt,
    decide,
    review: review?.scope === scope ? review : undefined,
    isReviewCurrent: () => {
      if (!review || review.scope !== scope) return false;
      try {
        review.checkAccess?.();
        return isCurrent();
      } catch {
        return false;
      }
    },
    isPending: preview.isPending || review?.scope === scope || latest?.state.status === "pending",
    recoveryCommand: recoveryCommand(latest, startedHere),
    rejectedAttempt: history.some((entry) => canReviewAgain(entry.state.error)) ? latest?.state.variables : undefined,
    acknowledge: () => {
      ensureCurrent();
      const cache = client.getMutationCache();
      for (const entry of cache.findAll(filters)) cache.remove(entry);
      commit.reset();
    },
    isCurrent,
  };
}
