import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { Button } from "@plane/propel/button";
import type {
  ProjectRelease,
  ReleaseCatalog,
  ReleaseMutation,
  ReleaseValues,
} from "@/services/project-releases.service";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { Field, PreviewDialog, inputStyle } from "@/components/work-item-types/controls";
import { emptyReleaseValues, reloadReleaseDraft, releaseValues } from "./release-draft";

type Props = {
  release?: ProjectRelease;
  revision: number;
  onClose: () => void;
  onSave: (input: ReleaseMutation) => Promise<void>;
  onReload: () => Promise<ReleaseCatalog>;
};
export function ReleaseEditor({ release, revision, onClose, onSave, onReload }: Props) {
  const [baseline, setBaseline] = useState({ revision, values: releaseValues(release ?? emptyReleaseValues) });
  const [defaults, setDefaults] = useState<ReleaseValues>(() => ({ ...baseline.values }));
  const [action, setAction] = useState<"create" | "update">(release ? "update" : "create");
  const [id] = useState(() => release?.id ?? crypto.randomUUID());
  const [failure, setFailure] = useState<unknown>();
  const [reloading, setReloading] = useState(false);
  const [removed, setRemoved] = useState(false);
  const [needsReload, setNeedsReload] = useState(false);
  const [current, setCurrent] = useState<ProjectRelease>();
  const form = useForm({
    defaultValues: defaults,
    onSubmit: async ({ value }) => {
      if (removed || needsReload || reloading) return;
      setFailure(undefined);
      try {
        await onSave({ id, action, revision: baseline.revision, values: value });
        onClose();
      } catch (cause) {
        setFailure(cause);
        if (isReleaseConflict(cause)) setNeedsReload(true);
      }
    },
  });
  const submitting = useStore(form.store, (state) => state.isSubmitting);
  const busy = submitting || reloading;
  const stale = needsReload;
  const reload = async () => {
    setReloading(true);
    try {
      const catalog = await onReload();
      const latest = catalog.releases.find((item) => item.id === id);
      if (action === "update" && !latest) {
        setRemoved(true);
        return;
      }
      if (latest) setAction("update");
      const values = releaseValues(latest ?? emptyReleaseValues);
      const next = reloadReleaseDraft(form.store.state.values, baseline.values, values);
      setBaseline({ revision: catalog.revision, values });
      setDefaults(next);
      form.reset(next);
      setCurrent(latest);
      setFailure(undefined);
      setNeedsReload(false);
    } catch (cause) {
      setFailure(cause);
    } finally {
      setReloading(false);
    }
  };
  return (
    <PreviewDialog
      title={action === "update" ? "Edit release" : "Create release"}
      onClose={onClose}
      dismissDisabled={busy}
      footer={
        <>
          <Button variant="secondary" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          {stale ? (
            <Button disabled={busy} loading={reloading} onClick={() => void reload()}>
              Reload current values
            </Button>
          ) : (
            <Button type="submit" form={`release-${id}`} disabled={busy || removed} loading={submitting}>
              {action === "update" ? "Save release" : "Create release"}
            </Button>
          )}
        </>
      }
    >
      <form
        id={`release-${id}`}
        onSubmit={(event) => {
          event.preventDefault();
          event.stopPropagation();
          void form.handleSubmit();
        }}
        className="space-y-4"
      >
        <fieldset disabled={busy} inert={busy} className="space-y-4">
          <form.Field
            name="name"
            validators={{ onChange: ({ value }) => (value.trim() ? undefined : "Enter a release name.") }}
          >
            {(field) => (
              <Field label="Name">
                {(inputId) => (
                  <>
                    <input
                      id={inputId}
                      required
                      maxLength={255}
                      className={inputStyle}
                      value={field.state.value}
                      onChange={(event) => field.handleChange(event.target.value)}
                      onBlur={field.handleBlur}
                    />
                    {field.state.meta.errors.length > 0 && (
                      <p role="alert" className="text-12 text-danger-primary">
                        {field.state.meta.errors.join(" ")}
                      </p>
                    )}
                  </>
                )}
              </Field>
            )}
          </form.Field>
          <form.Field name="description">
            {(field) => (
              <Field label="Description">
                {(inputId) => (
                  <textarea
                    id={inputId}
                    maxLength={10000}
                    rows={3}
                    className={inputStyle}
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </Field>
            )}
          </form.Field>
          <form.Field name="target_date">
            {(field) => (
              <Field label="Target date">
                {(inputId) => (
                  <input
                    id={inputId}
                    type="date"
                    className={inputStyle}
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </Field>
            )}
          </form.Field>
        </fieldset>
        {current && (
          <div className="rounded border border-subtle p-3 text-12 text-secondary">
            <p>Current saved release</p>
            <p className="font-medium">{current.name}</p>
            <p className="whitespace-pre-wrap">{current.description}</p>
            <p>{current.target_date || "No target date"}</p>
          </div>
        )}
        {removed && (
          <p role="alert" className="text-13 text-danger-primary">
            This release was removed. Your changes are still here; copy anything you need before closing.
          </p>
        )}
        {failure !== undefined && (
          <p role="alert" className="text-13 text-danger-primary">
            {typeErrorMessage(failure)}
          </p>
        )}
      </form>
    </PreviewDialog>
  );
}
function isReleaseConflict(cause: unknown) {
  if (!cause || typeof cause !== "object") return false;
  const error = cause as { status?: number; response?: { status?: number } };
  return [404, 409].includes(error.response?.status ?? error.status ?? 0);
}
