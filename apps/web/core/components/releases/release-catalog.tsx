import { useState } from "react";
import { Search, Plus } from "lucide-react";
import { Button } from "@plane/propel/button";
import type { ProjectRelease, ReleaseCatalog as Catalog, ReleaseMutation } from "@/services/project-releases.service";
import { EmptyState, inputStyle } from "@/components/work-item-types/controls";
import { ReleaseEditor } from "./release-editor";
import { ReleaseRemoval } from "./release-removal";

type Props = {
  catalog: Catalog;
  pending: boolean;
  onSave: (input: ReleaseMutation) => Promise<void>;
  onReload: () => Promise<Catalog>;
};
export function ReleaseCatalog({ catalog, pending, onSave, onReload }: Props) {
  const [search, setSearch] = useState("");
  const [editing, setEditing] = useState<ProjectRelease | "new">();
  const [removing, setRemoving] = useState<ProjectRelease>();
  const releases = catalog.releases.filter((release) =>
    `${release.name} ${release.description}`.toLocaleLowerCase().includes(search.trim().toLocaleLowerCase())
  );
  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <label className="relative min-w-48 flex-1 sm:max-w-xs">
          <Search className="pointer-events-none absolute top-2.5 left-3 size-4 text-tertiary" aria-hidden />
          <input
            aria-label="Search releases"
            type="search"
            className={`${inputStyle} pl-9`}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>
        {catalog.can_manage && (
          <Button disabled={pending} prependIcon={<Plus />} onClick={() => setEditing("new")}>
            Create release
          </Button>
        )}
      </div>
      {!catalog.releases.length ? (
        <EmptyState title="No releases yet" description="Create releases to use in work item properties." />
      ) : !releases.length ? (
        <EmptyState title="No matching releases" description="Try another search." />
      ) : (
        <ul className="divide-y divide-subtle rounded-lg border border-subtle">
          {releases.map((release) => (
            <li key={release.id} className="flex flex-wrap items-start gap-3 p-4">
              <div className="min-w-0 flex-1">
                <h3 className="text-14 font-medium break-words">{release.name}</h3>
                {release.description && (
                  <p className="mt-1 text-13 whitespace-pre-wrap text-secondary">{release.description}</p>
                )}
                {release.target_date && (
                  <p className="mt-2 text-12 text-tertiary">
                    Target <time dateTime={release.target_date}>{release.target_date}</time>
                  </p>
                )}
              </div>
              {catalog.can_manage && (
                <div className="flex gap-2">
                  <Button
                    variant="secondary"
                    disabled={pending}
                    aria-label={`Edit ${release.name}`}
                    onClick={() => setEditing(release)}
                  >
                    Edit
                  </Button>
                  <Button
                    variant="secondary"
                    disabled={pending}
                    aria-label={`Delete ${release.name}`}
                    onClick={() => setRemoving(release)}
                  >
                    Delete
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
      {editing && (
        <ReleaseEditor
          key={editing === "new" ? "new" : editing.id}
          release={editing === "new" ? undefined : editing}
          revision={catalog.revision}
          onSave={onSave}
          onReload={onReload}
          onClose={() => setEditing(undefined)}
        />
      )}
      {removing && (
        <ReleaseRemoval
          release={removing}
          revision={catalog.revision}
          pending={pending}
          onSave={onSave}
          onReload={onReload}
          onClose={() => setRemoving(undefined)}
        />
      )}
    </div>
  );
}
