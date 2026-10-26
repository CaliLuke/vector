/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { processAssetDuplication } from "@plane/editor";
import { workItemSchemaService } from "@/services/work-item-schema.service";
import { workItemEditsService } from "@/services/work-item-edits.service";
import { captureItemReadScope } from "@/queries/work-item-read-scope";
import { workspaceDraftDetailOptions } from "@/queries/workspace-drafts";
import type { WorkspaceDraftScope } from "@/queries/workspace-drafts";
import { useProjectPermissions, useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { skipToken, useQuery, useQueryClient } from "@tanstack/react-query";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import type { TIssue, IPartialProject } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { IssueService } from "@/services/issue/issue.service";
type LoadedSource = { item: Partial<TIssue>; sourceVersion?: string };
type Source = {
  data?: Partial<TIssue>;
  isDraft?: boolean;
  retainedPublication?: Partial<TIssue>;
  isOpen: boolean;
  fetchIssueDetails: boolean;
  workspaceSlug: string;
  routerProjectId?: string;
  workItem?: string;
  defaultProjectId?: string;
  getProjectByIdentifier: (identifier: string) => IPartialProject | undefined;
};
function resolveProject({
  data,
  isDraft,
  routerProjectId,
  workItem,
  defaultProjectId,
  getProjectByIdentifier,
}: Source) {
  if (isDraft) return null;
  if (data?.project_id !== undefined) return data.project_id;
  const identifier = workItem?.split("-")[0];
  const projectFromIssue = identifier ? getProjectByIdentifier(identifier)?.id : undefined;
  return data?.project_id ?? routerProjectId ?? projectFromIssue ?? defaultProjectId;
}
export function useIssueModalSource(source: Source) {
  const { data, isOpen, workspaceSlug } = source;
  const userId = useCurrentUserId();
  const client = useQueryClient();
  const sessionGeneration = useIdentityGeneration(client);
  const initialProjectId = resolveProject(source);
  const { accessState, scope, retryAccess } = useSourcePermissions(source, userId, initialProjectId);
  const sourceIssueId = data?.id ?? data?.sourceIssueId;
  const needsDescription = shouldReadSource(source, sourceIssueId, initialProjectId);
  const queryKey = [
    "work-item-modal",
    userId,
    sessionGeneration,
    workspaceSlug,
    source.isDraft ? "draft" : "issue",
    initialProjectId,
    sourceIssueId,
    data?.type_id,
    scope.membershipRevision,
    scope.membershipRole,
    scope.projectRole,
  ];
  const descriptionQuery = useQuery({
    queryKey,
    queryFn:
      userId && sourceIssueId && needsDescription && accessState === "ready"
        ? async ({ signal }) => {
            const current = captureItemReadScope(client, scope, queryKey);
            if (!current()) throw new Error("The account session changed.");
            const loaded = await readSource(source, initialProjectId, sourceIssueId, signal, client, {
              ...scope,
              sessionGeneration,
            });
            if (!current()) throw new Error("The account session changed.");
            return loaded;
          }
        : skipToken,
    enabled: isOpen && needsDescription && accessState === "ready",
    retry: false,
    staleTime: 0,
    gcTime: 0,
    refetchOnWindowFocus: false,
  });
  return {
    ...modalSourceFields(
      source,
      initialProjectId,
      needsDescription,
      accessState === "ready" && !sourceAccessDenied(descriptionQuery.error) ? descriptionQuery.data : undefined
    ),
    descriptionQuery,
    sourceAccessAllowed: accessState === "ready" && !sourceAccessDenied(descriptionQuery.error),
    accessKey: JSON.stringify([scope, sessionGeneration]),
    accessState,
    retryAccess,
  };
}

function useSourcePermissions(source: Source, userId: string | undefined, initialProjectId: string | null | undefined) {
  const { workspaceSlug } = source;
  const projectAccess = useProjectPermissions(
    workspaceSlug,
    source.isDraft ? undefined : (initialProjectId ?? undefined)
  );
  const workspaceAccess = useWorkspacePermissions(workspaceSlug);
  const access = source.isDraft ? workspaceAccess : projectAccess;
  const membership = access.membershipQuery.isSuccess ? access.membershipQuery.data : undefined;
  const permitted = source.isDraft ? workspaceAccess.can([5, 15, 20]) : projectAccess.can([15, 20]);
  const accessState = access.isError ? "error" : access.isPending ? "pending" : permitted ? "ready" : "denied";
  const scope = {
    accountId: userId,
    workspaceSlug,
    membershipRole: access.workspaceRole,
    membershipRevision: membership?.updated_at,
    projectId: source.isDraft ? undefined : (initialProjectId ?? undefined),
    projectRole: projectAccess.projectRole,
  };
  return {
    accessState,
    scope,
    retryAccess: async () => {
      await access.membershipQuery.refetch();
      if (!source.isDraft) await projectAccess.rolesQuery.refetch();
    },
  };
}

function modalSourceFields(
  source: Source,
  initialProjectId: string | null | undefined,
  needsDescription: boolean,
  fetched: LoadedSource | undefined
) {
  const fields = source.retainedPublication ?? (needsDescription ? fetched?.item : source.data);
  const activeProjectId = source.isDraft && fields?.project_id !== undefined ? fields.project_id : initialProjectId;
  const description = needsDescription ? fetched?.item.description_html : (fields?.description_html ?? "<p></p>");
  return { fields, activeProjectId, description, sourceVersion: fetched?.sourceVersion };
}

function shouldReadSource(source: Source, id: string | undefined, project: string | null | undefined) {
  return (
    !source.retainedPublication &&
    !!id &&
    (!!source.data?.id || source.isDraft || (!!project && source.fetchIssueDetails))
  );
}
async function readSource(
  source: Source,
  project: string | null | undefined,
  id: string,
  signal: AbortSignal,
  client: ReturnType<typeof useQueryClient>,
  draftScope: WorkspaceDraftScope
): Promise<LoadedSource> {
  const { workspaceSlug, data } = source;
  if (data?.id) {
    if (source.isDraft) await fetchDraftRecord(client, draftScope, id, signal);
    const saved = await workItemEditsService.source(
      workspaceSlug,
      source.isDraft ? "draft" : "edit",
      id,
      source.isDraft ? undefined : project!,
      signal
    );
    return { item: saved.item, sourceVersion: saved.source_version };
  }
  if (source.isDraft) {
    const draft = await fetchDraftRecord(client, draftScope, id, signal);
    return { item: await copyDraftSource(workspaceSlug, id, draft, data, signal) };
  }
  const issue = await readDuplicateSourceIssue(workspaceSlug, project!, id, Boolean(data?.is_epic), signal);
  return {
    item: {
      type_id: issue.type_id,
      description_html: issue.description_html ?? "<p></p>",
      custom_properties: issue.custom_properties ?? {},
    },
  };
}

async function readDuplicateSourceIssue(
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  isEpic: boolean,
  signal: AbortSignal
) {
  const service = new IssueService(isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES);
  const issue = await service.retrieve(workspaceSlug, projectId, issueId, undefined, { signal });
  if (!issue) throw new Error("Could not load work item description.");
  if (issue.id !== issueId || issue.project_id !== projectId)
    throw new Error("The server returned a different source work item.");
  return issue;
}

async function fetchDraftRecord(
  client: ReturnType<typeof useQueryClient>,
  scope: WorkspaceDraftScope,
  issueId: string,
  signal: AbortSignal
) {
  const options = workspaceDraftDetailOptions(client, scope, issueId);
  const cancel = () => void client.cancelQueries({ queryKey: options.queryKey, exact: true });
  if (signal.aborted) throw new Error("The draft read was cancelled.");
  signal.addEventListener("abort", cancel, { once: true });
  try {
    const draft = await client.fetchQuery(options);
    if (!draft) throw new Error("Draft not found.");
    return draft;
  } finally {
    signal.removeEventListener("abort", cancel);
  }
}

async function copyDraftSource(
  workspaceSlug: string,
  id: string,
  issue: Partial<TIssue>,
  data: Partial<TIssue> | undefined,
  signal: AbortSignal
): Promise<Partial<TIssue>> {
  const schema = await workItemSchemaService.schema(
    workspaceSlug,
    { mode: "draft", draft_id: id, project_id: issue.project_id ?? undefined, type_id: issue.type_id ?? "" },
    signal
  );
  const custom = { ...issue.custom_properties };
  for (const property of schema.properties) {
    const html = custom[property.id];
    if (property.kind === "Rich text" && typeof html === "string")
      custom[property.id] = processAssetDuplication(html).processedHtml;
  }
  return {
    ...issue,
    id: undefined,
    name: data?.name ?? issue.name,
    description_html: processAssetDuplication(issue.description_html ?? "<p></p>").processedHtml,
    custom_properties: custom,
  };
}

function sourceAccessDenied(error: unknown) {
  if (!error || typeof error !== "object") return false;
  const failure = error as { status?: number; response?: { status?: number } };
  return [401, 403, 404].includes(failure.response?.status ?? failure.status ?? 0);
}
