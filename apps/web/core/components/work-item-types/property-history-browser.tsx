import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import { RichTextEditor } from "@/components/editor/rich-text";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import {
  propertyHistoryCatalogOptions,
  propertyVersionPageOptions,
  propertyVersionDetailOptions,
} from "@/queries/work-item-property-history";
import type { PropertyHistoryScope } from "@/queries/work-item-property-history";
import type { PropertyVersion, PropertyVersionDetail } from "@/services/work-item-property-history.service";
import { inputStyle } from "./controls";

export function HistoryLoadError({ retry, children }: { retry: () => void; children: string }) {
  return (
    <p role="alert" className="text-13 text-danger-primary">
      {children}{" "}
      <button type="button" className="underline" onClick={retry}>
        Retry
      </button>
    </p>
  );
}
export function PropertyHistoryBrowser({
  scope,
  editable,
  restorable,
  onRestore,
}: {
  scope: PropertyHistoryScope;
  editable: boolean;
  restorable: string[];
  onRestore: (version: PropertyVersionDetail) => void;
}) {
  const catalog = useQuery(propertyHistoryCatalogOptions(scope));
  const [field, setField] = useState<string>();
  if (field === undefined && catalog.isSuccess) setField(catalog.data[0]?.property_id ?? "");
  if (catalog.isError)
    return <HistoryLoadError retry={() => void catalog.refetch()}>Could not load property history.</HistoryLoadError>;
  if (!catalog.isSuccess) return <output>Loading property history…</output>;
  if (!catalog.data.length)
    return (
      <p className="text-13 text-secondary">No rich-text property revisions have been saved for this work item.</p>
    );
  return (
    <div className="space-y-4">
      <label className="block space-y-2 text-13">
        <span>Property</span>
        <select className={inputStyle} value={field ?? ""} onChange={(event) => setField(event.target.value)}>
          {catalog.data.map((entry) => (
            <option key={entry.property_id} value={entry.property_id}>
              {entry.title || "Removed property"} ({entry.version_count})
            </option>
          ))}
        </select>
      </label>
      {field && (
        <PropertyVersions
          key={field}
          scope={scope}
          propertyId={field}
          editable={editable}
          restorable={restorable}
          onRestore={onRestore}
        />
      )}
    </div>
  );
}
function PropertyVersions({
  scope,
  propertyId,
  editable,
  restorable,
  onRestore,
}: {
  scope: PropertyHistoryScope;
  propertyId: string;
  editable: boolean;
  restorable: string[];
  onRestore: (version: PropertyVersionDetail) => void;
}) {
  const [cursors, setCursors] = useState<(string | null)[]>([null]);
  const [selected, setSelected] = useState<string>();
  const page = useQuery(propertyVersionPageOptions(scope, propertyId, cursors[cursors.length - 1]));
  if (selected === undefined && page.isSuccess) setSelected(page.data.results[0]?.id ?? "");
  return (
    <div className="grid min-h-64 gap-5 md:grid-cols-[15rem_minmax(0,1fr)]">
      <nav aria-label="Property revisions" className="space-y-3 md:border-r md:border-subtle md:pr-4">
        {page.isError ? (
          <HistoryLoadError retry={() => void page.refetch()}>Could not load revisions.</HistoryLoadError>
        ) : !page.isSuccess ? (
          <output>Loading revisions…</output>
        ) : (
          <>
            <ul className="max-h-80 space-y-1 overflow-y-auto">
              {page.data.results.map((version) => (
                <li key={version.id}>
                  <button
                    type="button"
                    aria-current={selected === version.id ? "true" : undefined}
                    onClick={() => setSelected(version.id)}
                    className={`w-full rounded-md p-2 text-left text-12 hover:bg-layer-2 ${selected === version.id ? "bg-layer-2 ring-1 ring-accent-strong" : ""}`}
                  >
                    <VersionAttribution version={version} />
                  </button>
                </li>
              ))}
            </ul>
            {!page.data.results.length && <p className="text-13 text-secondary">No revisions.</p>}
            <div className="flex justify-between gap-2">
              <Button
                variant="secondary"
                disabled={cursors.length === 1 || page.isFetching}
                onClick={() => {
                  setSelected(undefined);
                  setCursors((values) => values.slice(0, -1));
                }}
              >
                Newer
              </Button>
              <Button
                variant="secondary"
                disabled={!page.data.next_cursor || page.isFetching}
                onClick={() => {
                  setSelected(undefined);
                  setCursors((values) => [...values, page.data.next_cursor]);
                }}
              >
                Older
              </Button>
            </div>
          </>
        )}
      </nav>
      <div className="min-w-0">
        {page.isSuccess && selected && (
          <PropertyRevision
            key={selected}
            scope={scope}
            propertyId={propertyId}
            versionId={selected}
            editable={editable}
            restorable={restorable}
            onRestore={onRestore}
          />
        )}
      </div>
    </div>
  );
}
export function VersionAttribution({ version }: { version: PropertyVersion }) {
  const provenance = {
    change: "Saved revision",
    restore: "Restored revision",
    preservation: "Preserved when the field changed",
    baseline: "Earlier saved content · original author unknown",
  }[version.source];
  return (
    <>
      <time className="block font-medium" dateTime={version.created_at}>
        {version.source === "baseline" && "Captured "}
        {new Date(version.created_at).toLocaleString()}
      </time>
      <span className="block text-secondary">{version.actor?.display_name || "Unknown author"}</span>
      <span className="block text-tertiary">{provenance}</span>
    </>
  );
}
function PropertyRevision({
  scope,
  propertyId,
  versionId,
  editable,
  restorable,
  onRestore,
}: {
  scope: PropertyHistoryScope;
  propertyId: string;
  versionId: string;
  editable: boolean;
  restorable: string[];
  onRestore: (version: PropertyVersionDetail) => void;
}) {
  const detail = useQuery(propertyVersionDetailOptions(scope, propertyId, versionId));
  if (detail.isError)
    return <HistoryLoadError retry={() => void detail.refetch()}>Could not load this revision.</HistoryLoadError>;
  if (!detail.isSuccess) return <output>Loading revision…</output>;
  return (
    <article className="space-y-4">
      <header>
        <h3 className="text-16 font-medium">{detail.data.property_title || "Removed property"}</h3>
        <div className="mt-2 text-12">
          <VersionAttribution version={detail.data} />
        </div>
      </header>
      <HistoryRichText scope={scope} versionId={versionId} value={detail.data.value} />
      {editable && restorable.includes(propertyId) && (
        <Button variant="secondary" onClick={() => onRestore(detail.data)}>
          Restore this revision
        </Button>
      )}
    </article>
  );
}
export function HistoryRichText({
  scope,
  versionId,
  value,
}: {
  scope: PropertyHistoryScope;
  versionId: string;
  value: string | null;
}) {
  const workspace = useWorkspaceBySlug(scope.workspaceSlug);
  if (value === null) return <p className="text-13 text-secondary">This field was empty.</p>;
  if (workspace.isError)
    return <HistoryLoadError retry={() => void workspace.refetch()}>Could not load rich text.</HistoryLoadError>;
  if (!workspace.data) return <output>Loading rich text…</output>;
  return (
    <RichTextEditor
      key={versionId}
      id={`property-history-${scope.issueId}-${versionId}`}
      initialValue={value}
      workspaceSlug={scope.workspaceSlug}
      workspaceId={workspace.data.id}
      projectId={scope.projectId}
      editable={false}
      containerClassName="min-h-32 border-none"
    />
  );
}
