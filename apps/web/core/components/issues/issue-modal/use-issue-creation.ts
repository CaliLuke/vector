/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useId } from "react";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { captureIdentityScope, isIdentityScopeCurrent, isIdentityScopeVerified } from "@/queries/identity";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { TIssue, TWorkspaceDraftIssue } from "@plane/types";

export type CreatedWorkItem = TIssue | TWorkspaceDraftIssue;
type Request<TPayload extends Partial<TIssue> = Partial<TIssue>> = {
  payload: TPayload;
  isDraft: boolean;
  assetReferences?: string[];
};
type Operations<TPayload extends Partial<TIssue>> = {
  create: (request: Request<TPayload>) => Promise<CreatedWorkItem>;
  finalizeAssets?: (issue: CreatedWorkItem, request: Request<TPayload>) => Promise<void>;
  assignCycle: (issue: CreatedWorkItem, request: Request<TPayload>) => Promise<void>;
  assignModules: (issue: CreatedWorkItem, request: Request<TPayload>) => Promise<void>;
  onSaved?: (issue: CreatedWorkItem) => Promise<void>;
};

/** Keep completed durable steps when a later step fails, so Retry cannot create a second issue. */
export class IssueCreationIncompleteError extends Error {
  constructor(
    readonly issueId: string,
    cause: unknown
  ) {
    super("The work item was created. Retry to finish the remaining steps.", { cause });
    this.name = "IssueCreationIncompleteError";
  }
}
export function useIssueCreation<TPayload extends Partial<TIssue> = Partial<TIssue>>(
  operations: Operations<TPayload>,
  scopeKey?: string
) {
  const client = useQueryClient();
  const userId = useCurrentUserId(),
    generation = useIdentityGeneration(client),
    instance = useId();
  const owner = JSON.stringify([instance, userId, generation, scopeKey]);
  const ensureIdentity = (identity: ReturnType<typeof captureIdentityScope>) => {
    if (!isIdentityScopeCurrent(client, identity)) throw new Error("The account session has changed.");
  };
  const ensureVerifiedIdentity = (identity: ReturnType<typeof captureIdentityScope>) => {
    if (!isIdentityScopeVerified(client, identity))
      throw new Error("The account session has changed or is unverified.");
  };
  const create = useMutation({
    mutationKey: ["issue-creation", owner, "create"],
    mutationFn: async (request: Request<TPayload>) => {
      const snapshot = structuredClone(request);
      const identity = captureIdentityScope(client, userId);
      ensureVerifiedIdentity(identity);
      return { issue: await operations.create(snapshot), request: snapshot, identity, owner };
    },
    retry: false,
  });
  const assets = useMutation({
    mutationKey: ["issue-creation", owner, "assets"],
    mutationFn: async ({ issue, request }: { issue: CreatedWorkItem; request: Request<TPayload> }) => {
      await operations.finalizeAssets?.(issue, request);
    },
    retry: false,
  });
  const cycle = useMutation({
    mutationKey: ["issue-creation", owner, "cycle"],
    mutationFn: ({ issue, request }: { issue: CreatedWorkItem; request: Request<TPayload> }) =>
      operations.assignCycle(issue, request),
    retry: false,
  });
  const modules = useMutation({
    mutationKey: ["issue-creation", owner, "modules"],
    mutationFn: ({ issue, request }: { issue: CreatedWorkItem; request: Request<TPayload> }) =>
      operations.assignModules(issue, request),
    retry: false,
  });
  const saved = useMutation({
    mutationKey: ["issue-creation", owner, "saved"],
    mutationFn: async (issue: CreatedWorkItem) => {
      await operations.onSaved?.(issue);
    },
    retry: false,
  });
  const submit = async (request: Request<TPayload>) => {
    const resumed = create.data?.owner === owner;
    const created = resumed && create.data ? create.data : await create.mutateAsync(request);
    const { issue, identity } = created;
    try {
      ensureIdentity(identity);
      request = created.request;
      if (operations.finalizeAssets && (!resumed || !assets.isSuccess)) {
        ensureVerifiedIdentity(identity);
        await assets.mutateAsync(created);
      }
      ensureIdentity(identity);
      if (!resumed || !cycle.isSuccess) {
        ensureVerifiedIdentity(identity);
        await cycle.mutateAsync({ issue, request });
      }
      ensureIdentity(identity);
      if (!resumed || !modules.isSuccess) {
        ensureVerifiedIdentity(identity);
        await modules.mutateAsync({ issue, request });
      }
      ensureIdentity(identity);
      if (operations.onSaved && (!resumed || !saved.isSuccess)) {
        ensureVerifiedIdentity(identity);
        await saved.mutateAsync(issue);
      }
      ensureIdentity(identity);
    } catch (cause) {
      throw new IssueCreationIncompleteError(issue.id, cause);
    }
    return issue;
  };
  const reset = () => {
    create.reset();
    assets.reset();
    cycle.reset();
    modules.reset();
    saved.reset();
  };
  return { submit, reset, hasCreated: create.isSuccess };
}
