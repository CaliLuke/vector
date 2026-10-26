import { useState } from "react";
import { Button } from "@plane/propel/button";
import type { ProjectRelease, ReleaseCatalog, ReleaseMutation } from "@/services/project-releases.service";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { PreviewDialog } from "@/components/work-item-types/controls";
import { emptyReleaseValues } from "./release-draft";

type Props = {
  release: ProjectRelease;
  revision: number;
  pending: boolean;
  onSave: (input: ReleaseMutation) => Promise<void>;
  onReload: () => Promise<ReleaseCatalog>;
  onClose: () => void;
};
export function ReleaseRemoval({ release, revision, pending, onSave, onReload, onClose }: Props) {
  const [snapshot, setSnapshot] = useState({ release, revision });
  const [error, setError] = useState<unknown>();
  const [saving, setSaving] = useState(false);
  const [needsReload, setNeedsReload] = useState(false);
  const [removed, setRemoved] = useState(false);
  const remove = async () => {
    if (needsReload || removed) return;
    setSaving(true);
    setError(undefined);
    try {
      await onSave({
        action: "delete",
        id: snapshot.release.id,
        revision: snapshot.revision,
        values: emptyReleaseValues,
      });
      onClose();
    } catch (cause) {
      setError(cause);
      if (requiresReload(cause)) setNeedsReload(true);
    } finally {
      setSaving(false);
    }
  };
  const reload = async () => {
    setSaving(true);
    try {
      const catalog = await onReload(),
        current = catalog.releases.find((item) => item.id === snapshot.release.id);
      if (current) setSnapshot({ release: current, revision: catalog.revision });
      else setRemoved(true);
      setNeedsReload(false);
      setError(undefined);
    } catch (cause) {
      setError(cause);
    } finally {
      setSaving(false);
    }
  };
  const busy = pending || saving;
  return (
    <PreviewDialog
      title={removed ? "Release removed" : `Delete ${snapshot.release.name}?`}
      onClose={onClose}
      dismissDisabled={busy}
      footer={
        <>
          <Button variant="secondary" disabled={busy} onClick={onClose}>
            {removed ? "Close" : "Cancel"}
          </Button>
          {!removed &&
            (needsReload ? (
              <Button disabled={busy} loading={saving} onClick={() => void reload()}>
                Reload release
              </Button>
            ) : (
              <Button disabled={busy} loading={saving} onClick={() => void remove()}>
                Delete release
              </Button>
            ))}
        </>
      }
    >
      {removed ? (
        <p className="text-13">This release is no longer in the project catalog.</p>
      ) : (
        <>
          <p className="text-13 text-secondary">
            This release will be removed from the project's catalog. Releases referenced by a work item or draft cannot
            be deleted.
          </p>
          <p className="mt-3 text-13 whitespace-pre-wrap">{snapshot.release.description}</p>
          {snapshot.release.target_date && (
            <p className="mt-1 text-12 text-secondary">Target {snapshot.release.target_date}</p>
          )}
        </>
      )}
      {error !== undefined && (
        <p role="alert" className="mt-3 text-13 text-danger-primary">
          {typeErrorMessage(error)}
        </p>
      )}
    </PreviewDialog>
  );
}
function requiresReload(cause: unknown) {
  if (!cause || typeof cause !== "object") return false;
  const error = cause as { status?: number; response?: { status?: number } };
  return [404, 409].includes(error.response?.status ?? error.status ?? 0);
}
