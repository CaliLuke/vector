import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "@tanstack/react-router";
import { Button } from "@plane/propel/button";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { workItemHierarchyService } from "@/services/work-item-hierarchy.service";
import type { HierarchyCatalog, HierarchyCommit, HierarchyConfiguration } from "@/services/work-item-hierarchy.service";
import { useConfigurationMutation } from "./configuration-mutation";
import { EmptyState } from "./controls";
import { HierarchyLayout } from "./hierarchy-layout";
import { HierarchyReviewDialog } from "./hierarchy-review";
import { normalizeHierarchy } from "./preview-actions";

export function WorkspaceHierarchy() {
  const accountId = useCurrentUserId();
  const { workspaceSlug = "" } = useParams({ strict: false });
  const query = useQuery({
    queryKey: ["workspace", workspaceSlug, "hierarchy", accountId],
    queryFn: ({ signal }) => workItemHierarchyService.catalog(workspaceSlug, signal),
    enabled: !!workspaceSlug && !!accountId,
  });
  const mutation = useConfigurationMutation(workspaceSlug, (input: HierarchyCommit) =>
    workItemHierarchyService.commit(workspaceSlug, input)
  );
  if (query.isPending)
    return <output aria-label="Loading hierarchy" className="block h-24 animate-pulse rounded-lg bg-layer-1" />;
  if (!query.data)
    return (
      <EmptyState title="Couldn't load hierarchy" description={typeErrorMessage(query.error)}>
        <Button
          onClick={() => {
            void query.refetch();
          }}
        >
          Try again
        </Button>
      </EmptyState>
    );
  return (
    <>
      {query.isError && (
        <p role="alert" className="mb-4 text-13 text-danger-primary">
          {typeErrorMessage(query.error)}{" "}
          <Button
            variant="secondary"
            onClick={() => {
              void query.refetch();
            }}
          >
            Try again
          </Button>
        </p>
      )}
      <HierarchyEditor
        key={`${accountId}:${workspaceSlug}`}
        catalog={query.isError ? { ...query.data, can_manage: false } : query.data}
        pending={mutation.isPending}
        onCommit={mutation.mutateAsync}
        workspaceSlug={workspaceSlug}
        accountId={accountId}
      />
    </>
  );
}

function catalogHierarchy(catalog: HierarchyCatalog) {
  return normalizeHierarchy(
    Object.fromEntries(catalog.types.map((type) => [type.id, type.level])),
    catalog.default_level
  );
}

type EditorProps = {
  catalog: HierarchyCatalog;
  pending: boolean;
  onCommit: (input: HierarchyCommit) => Promise<HierarchyCatalog>;
  workspaceSlug: string;
  accountId: string | undefined;
};
export function HierarchyEditor({ catalog, pending, onCommit, workspaceSlug, accountId }: EditorProps) {
  const [saved, setSaved] = useState(catalog);
  const [activated, setActivated] = useState(false);
  const [review, setReview] = useState<HierarchyConfiguration>();
  const form = useForm({
    defaultValues: { hierarchy: catalogHierarchy(saved) },
    onSubmit: ({ value }) => {
      if (!catalog.can_manage || catalog.revision !== saved.revision) return;
      setReview({
        revision: saved.revision,
        default_level: value.hierarchy.defaultLevel,
        placements: Object.entries(value.hierarchy.levels).map(([type_id, level]) => ({ type_id, level })),
      });
    },
  });
  const hierarchy = useStore(form.store, (state) => state.values.hierarchy);
  const changed =
    !saved.hierarchy_enabled ||
    hierarchy.defaultLevel !== saved.default_level ||
    saved.types.some((type) => type.level !== hierarchy.levels[type.id]);
  const stale = catalog.revision !== saved.revision;
  const readOnly = !catalog.can_manage || pending || !!review;
  const reset = (next: HierarchyCatalog) => {
    setSaved(next);
    form.reset({ hierarchy: catalogHierarchy(next) });
    setActivated(false);
  };
  if (!catalog.enabled)
    return (
      <EmptyState title="Enable work item types first" description="Hierarchy uses your workspace's work item types.">
        <Link
          to="/$workspaceSlug/settings/work-item-types"
          params={{ workspaceSlug }}
          search={{ screen: undefined, state: undefined }}
          className="text-accent-primary hover:underline"
        >
          Manage work item types
        </Link>
      </EmptyState>
    );
  if (!saved.hierarchy_enabled && !activated)
    return (
      <EmptyState
        title="Give your work a structure"
        description="Choose which types can contain other types. Once saved, hierarchy cannot be turned off."
      >
        <Button
          disabled={readOnly}
          onClick={() => {
            reset(catalog);
            setActivated(true);
          }}
        >
          Enable hierarchy
        </Button>
      </EmptyState>
    );
  return (
    <>
      {stale && (
        <div role="alert" className="mb-4 space-y-2 text-13">
          <p>
            Hierarchy configuration changed. Your draft is still here. Reload the saved configuration before starting a
            new review.
          </p>
          <Button variant="secondary" disabled={pending || !!review} onClick={() => reset(catalog)}>
            Discard and reload
          </Button>
        </div>
      )}
      <HierarchyLayout
        types={saved.types}
        levels={hierarchy.levels}
        defaultLevel={hierarchy.defaultLevel}
        readOnly={readOnly}
        onMove={(id, level) => {
          if (!readOnly && level <= 20 && saved.types.some((type) => type.id === id))
            form.setFieldValue(
              "hierarchy",
              normalizeHierarchy({ ...hierarchy.levels, [id]: level }, hierarchy.defaultLevel)
            );
        }}
        onDefault={(level) => {
          if (!readOnly) form.setFieldValue("hierarchy.defaultLevel", level);
        }}
      >
        <div className="flex flex-wrap items-center justify-between gap-4 border-t border-subtle pt-5">
          <p className="text-12 text-tertiary">
            {changed ? "You have unsaved changes." : "Drag types between levels, or use the level selector."}
          </p>
          <div className="flex gap-2">
            <Button variant="secondary" disabled={readOnly || !changed} onClick={() => reset(catalog)}>
              Discard
            </Button>
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void form.handleSubmit().catch(() => undefined);
              }}
            >
              <Button type="submit" disabled={readOnly || !changed || stale}>
                Save changes
              </Button>
            </form>
          </div>
        </div>
      </HierarchyLayout>
      {review && (
        <HierarchyReviewDialog
          accountId={accountId}
          workspaceSlug={workspaceSlug}
          input={review}
          pending={pending}
          onClose={() => {
            if (!pending) setReview(undefined);
          }}
          onCommit={async (input) => {
            if (!catalog.can_manage) throw new Error("Administrator access must be confirmed before saving.");
            const result = await onCommit(input);
            reset(result);
            setReview(undefined);
          }}
        />
      )}
    </>
  );
}
