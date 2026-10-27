import { PropertyIcon } from "./property-icon";
import { forgetItemReads, itemAccessOptions, retryItemAccess } from "@/queries/work-item-access";
import { isPropertyHistoryAccessDenied } from "@/queries/work-item-property-history";
import { useEffect, useEffectEvent, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { RichTextEditor } from "@/components/editor/rich-text";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { itemDetailSchemaOptions } from "@/queries/work-item-detail-schema";
import type { PublishedItemEditScope } from "@/queries/published-item-edit";
import type { ItemSchema } from "@/services/work-item-schema.service";
import { editableProperty } from "./item-property-draft";
import { ItemPropertyValue } from "./item-property-value";
import { ItemPropertyHistory } from "./item-property-history";
import { InlinePropertyEditor } from "./inline-property-editor";

type Target = { workspaceSlug: string; projectId: string; issueId: string; disabled?: boolean };
export const ItemCustomProperties = function ItemCustomProperties({ disabled = false, ...target }: Target) {
  const accountId = useCurrentUserId();
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const accessScope = { ...target, accountId, generation };
  const denied = useQuery(itemAccessOptions(accessScope));
  const clearPrivateReads = useEffectEvent(() => forgetItemReads(client, accessScope));
  useEffect(() => {
    if (denied.data) clearPrivateReads();
  }, [denied.data]);
  const permissions = useProjectPermissions(target.workspaceSlug, target.projectId);
  const detail = useWorkItemDetail(target.workspaceSlug, target.projectId, target.issueId);
  const item = detail.issue;
  if (denied.data)
    return (
      <p role="alert" className="text-12 text-danger-primary">
        Custom properties are no longer available with your current access.{" "}
        <button type="button" onClick={() => retryItemAccess(client, accessScope)}>
          Retry
        </button>
      </p>
    );
  if (permissions.isPending) return <p className="text-12 text-tertiary">Loading custom properties…</p>;
  if (permissions.isError)
    return (
      <p role="alert" className="text-12 text-danger-primary">
        Could not verify work item access.{" "}
        <button
          type="button"
          onClick={() => {
            void permissions.membershipQuery.refetch();
            void permissions.rolesQuery.refetch();
          }}
        >
          Retry
        </button>
      </p>
    );
  if (permissions.projectRole === undefined || (item && item.project_id !== target.projectId)) return null;
  const editable = !disabled && permissions.can([15, 20]) && !!item && !item.archived_at && !item.is_draft;
  const scope = {
    ...target,
    accountId,
    generation,
    membershipRole: permissions.workspaceRole,
    membershipRevision: permissions.membershipQuery.data?.updated_at,
    projectRole: permissions.projectRole,
  };
  return (
    <>
      <CustomPropertySection
        key={JSON.stringify([scope, editable])}
        scope={scope}
        editable={editable}
        refreshVersion={item?.updated_at}
      />
      <ItemPropertyHistory {...target} disabled={disabled} />
    </>
  );
};
function CustomPropertySection({
  scope,
  editable,
  refreshVersion,
}: {
  scope: PublishedItemEditScope;
  editable: boolean;
  refreshVersion?: string;
}) {
  const [editing, setEditing] = useState(false);
  const [saved, setSaved] = useState(false);
  const client = useQueryClient();
  const schema = useQuery(itemDetailSchemaOptions({ ...scope, refreshVersion }));
  const blockAccess = () => forgetItemReads(client, scope);
  if (isPropertyHistoryAccessDenied(schema.error)) return null;
  // A background read refresh must not unmount an open local draft.
  if (editing)
    return (
      <section aria-label="Custom properties" className="my-4 min-w-0 border-t border-subtle pt-4">
        <h6 className="mb-3 text-body-xs-medium">Custom properties</h6>
        <InlinePropertyEditor
          scope={scope}
          onClose={() => setEditing(false)}
          onAccessLost={blockAccess}
          onSaved={() => {
            setEditing(false);
            setSaved(true);
          }}
        />
      </section>
    );
  if (schema.isError)
    return (
      <p role="alert" className="text-12 text-danger-primary">
        Could not load custom properties.{" "}
        <button type="button" onClick={() => void schema.refetch()}>
          Retry
        </button>
      </p>
    );
  if (!schema.isSuccess) return <p className="text-12 text-tertiary">Loading custom properties…</p>;
  if (!schema.data.enabled || !schema.data.properties.length) return null;
  const hasEditable = schema.data.properties.some(editableProperty);
  return (
    <section aria-label="Custom properties" className="my-4 min-w-0 border-t border-subtle pt-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <h6 className="text-body-xs-medium">Custom properties</h6>
        {editable && hasEditable && (
          <button
            type="button"
            className="rounded px-1 py-1 text-12 text-secondary hover:bg-layer-2 focus-visible:outline-2 focus-visible:outline-accent-strong"
            onClick={() => {
              setSaved(false);
              setEditing(true);
            }}
          >
            Edit properties
          </button>
        )}
      </div>
      {saved && <output className="mb-2 text-12 text-secondary">Work item saved.</output>}
      <SavedPropertyList scope={scope} snapshot={schema.data} />
    </section>
  );
}
function SavedPropertyList({ scope, snapshot }: { scope: PublishedItemEditScope; snapshot: ItemSchema }) {
  const workspace = useWorkspaceBySlug(scope.workspaceSlug);
  return (
    <>
      <dl className="space-y-4">
        {snapshot.properties.map((property) => (
          <div key={property.id} className="min-w-0">
            <dt
              className="mb-1 flex items-center gap-2 text-12 text-secondary"
              title={property.description || undefined}
            >
              <PropertyIcon kind={property.kind} icon={property.icon} />
              {property.title}
            </dt>
            <dd className="min-w-0 text-13 break-words text-primary">
              <ItemPropertyValue
                property={property}
                schema={snapshot}
                renderRichText={(html) =>
                  workspace.data ? (
                    <RichTextEditor
                      key={`${scope.issueId}:${property.id}:${snapshot.item_updated_at}`}
                      id={`item-property-${scope.issueId}-${property.id}`}
                      initialValue={html}
                      workspaceSlug={scope.workspaceSlug}
                      workspaceId={workspace.data.id}
                      projectId={scope.projectId}
                      editable={false}
                      containerClassName="border-none"
                    />
                  ) : (
                    <span>
                      {workspace.isError ? "Could not load rich text." : "Loading rich text…"}
                      {workspace.isError && (
                        <button type="button" onClick={() => void workspace.refetch()}>
                          Retry
                        </button>
                      )}
                    </span>
                  )
                }
              />
            </dd>
          </div>
        ))}
      </dl>
      {snapshot.properties.some((property) => property.kind === "Formula") && (
        <p className="mt-3 text-11 text-tertiary">
          Calculated <time dateTime={snapshot.evaluated_at}>{new Date(snapshot.evaluated_at).toLocaleString()}</time>
        </p>
      )}
    </>
  );
}
