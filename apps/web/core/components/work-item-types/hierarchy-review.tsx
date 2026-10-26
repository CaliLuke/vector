import { useEffect, useId } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Button } from "@plane/propel/button";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { workItemHierarchyService } from "@/services/work-item-hierarchy.service";
import type { HierarchyCommit, HierarchyConfiguration } from "@/services/work-item-hierarchy.service";
import { Confirmation, PreviewDialog } from "./controls";

type Props = {
  accountId: string | undefined;
  workspaceSlug: string;
  input: HierarchyConfiguration;
  pending: boolean;
  onCommit: (input: HierarchyCommit) => Promise<void>;
  onClose: () => void;
};
export function HierarchyReviewDialog({ accountId, workspaceSlug, input, pending, onCommit, onClose }: Props) {
  const instance = useId();
  const client = useQueryClient();
  const query = useQuery({
    // Review receipts must survive catalogue invalidation after ambiguous writes.
    queryKey: ["work-item-hierarchy-review", accountId, workspaceSlug, instance, input],
    queryFn: ({ signal }) => workItemHierarchyService.preview(workspaceSlug, input, signal),
    staleTime: Infinity,
    gcTime: 0,
    retry: false,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });
  useEffect(() => {
    if (query.error)
      void client.invalidateQueries({ queryKey: ["workspace", workspaceSlug, "hierarchy", accountId], exact: true });
  }, [query.error, client, workspaceSlug, accountId]);
  if (!query.data)
    return (
      <PreviewDialog
        title="Review hierarchy changes"
        onClose={onClose}
        footer={
          <Button variant="secondary" onClick={onClose}>
            Go back and fix
          </Button>
        }
      >
        {query.isPending ? (
          <output>Checking parent relationships…</output>
        ) : (
          <div className="space-y-3">
            <p role="alert">{typeErrorMessage(query.error)}</p>
            <Button
              onClick={() => {
                void query.refetch();
              }}
            >
              Try again
            </Button>
          </div>
        )}
      </PreviewDialog>
    );
  const review = query.data;
  return (
    <Confirmation
      title={review.links ? "Saving this will break existing links" : "Save hierarchy changes"}
      action={review.links ? "Save anyway" : "Save changes"}
      token={review.links ? "confirm" : undefined}
      onClose={() => {
        if (!pending) onClose();
      }}
      onConfirm={async ({ confirmation }) => onCommit({ ...input, review_id: review.id, confirm: confirmation })}
    >
      <p>
        {review.links
          ? `${review.links} parent links do not fit this hierarchy. Saving removes these links and records the change. The work items remain.`
          : "All existing parent links fit this hierarchy. Saving applies the levels and default shown in the builder."}
      </p>
      {review.pairs.length > 0 && (
        <ul className="max-h-64 space-y-2 overflow-y-auto rounded border border-subtle p-3">
          {review.pairs.map((pair) => (
            <li key={pair.child_id}>
              <Link
                to="/$workspaceSlug/projects/$projectId/issues/$issueId"
                params={{ workspaceSlug, projectId: pair.parent_project_id, issueId: pair.parent_id }}
                className="text-accent-primary hover:underline"
              >
                {pair.parent_name}
              </Link>
              {" → "}
              <Link
                to="/$workspaceSlug/projects/$projectId/issues/$issueId"
                params={{ workspaceSlug, projectId: pair.child_project_id, issueId: pair.child_id }}
                className="text-accent-primary hover:underline"
              >
                {pair.child_name}
              </Link>
            </li>
          ))}
        </ul>
      )}
      {review.restricted_links > 0 && (
        <p>
          {review.restricted_links} additional links belong to private drafts or work items you cannot currently view.
        </p>
      )}
      <Button variant="secondary" disabled={pending} onClick={onClose}>
        Go back and fix
      </Button>
    </Confirmation>
  );
}
