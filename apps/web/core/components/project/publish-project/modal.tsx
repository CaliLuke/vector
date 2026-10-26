/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useState } from "react";
import { useParams } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm, useStore } from "@tanstack/react-form";
import { SPACE_BASE_PATH, SPACE_BASE_URL } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { GlobeIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TProjectPublishSettings } from "@plane/types";
import type { TProjectCatalogueScope } from "@/queries/projects";
import { Loader, ToggleSwitch, ModalCore, EModalWidth, Checkbox } from "@plane/ui";
import { copyTextToClipboard } from "@plane/utils/string";
import { projectPublishingOptions, projectPublishingMutationOptions } from "@/queries/project-publishing";
import { useProjectDetail } from "@/hooks/use-projects";

type Props = { isOpen: boolean; projectId: string; onClose: () => void };
const toggleFields = [
  { name: "is_comments_enabled", label: "Allow comments" },
  { name: "is_reactions_enabled", label: "Allow reactions" },
  { name: "is_votes_enabled", label: "Allow voting" },
] as const;
const layouts = ["list", "kanban"] as const;
function defaults(settings: TProjectPublishSettings) {
  return {
    is_comments_enabled: settings.is_comments_enabled,
    is_reactions_enabled: settings.is_reactions_enabled,
    is_votes_enabled: settings.is_votes_enabled,
    view_props: settings.view_props ?? { list: true, kanban: true },
  };
}
export function PublishProjectModal(props: Props) {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  return props.isOpen ? (
    <PublishingDialog key={`${workspaceSlug}/${props.projectId}`} {...props} workspaceSlug={workspaceSlug} />
  ) : null;
}
function PublishingDialog({ workspaceSlug, projectId, onClose }: Props & { workspaceSlug: string }) {
  const client = useQueryClient();
  const projectQuery = useProjectDetail(workspaceSlug, projectId);
  const settings = useQuery(projectPublishingOptions(client, projectQuery.scope, Boolean(projectQuery.project)));
  if (projectQuery.accessDenied)
    return (
      <ModalCore isOpen handleClose={onClose} width={EModalWidth.XXL}>
        <div role="alert" className="p-5">
          You do not have access to this project.
        </div>
      </ModalCore>
    );
  if (projectQuery.project && settings.data && !settings.isError)
    return (
      <PublishingForm
        key={`${projectQuery.scope.accountId}/${projectQuery.scope.sessionGeneration}/${projectQuery.scope.projectId}`}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        projectScope={projectQuery.scope}
        settings={settings.data}
        onClose={onClose}
      />
    );
  return (
    <ModalCore isOpen handleClose={onClose} width={EModalWidth.XXL}>
      <div className="p-5">
        {projectQuery.isError || settings.isError ? (
          <div role="alert">
            Unable to load publishing settings.{" "}
            <button
              type="button"
              onClick={() => {
                if (projectQuery.isError) void projectQuery.retryAccess();
                else void settings.refetch();
              }}
            >
              Retry
            </button>
          </div>
        ) : (
          <Loader>
            <Loader.Item height="80px" />
          </Loader>
        )}
      </div>
    </ModalCore>
  );
}
function PublishingForm({
  workspaceSlug,
  projectId,
  projectScope,
  settings,
  onClose,
}: {
  workspaceSlug: string;
  projectId: string;
  projectScope: TProjectCatalogueScope;
  settings: TProjectPublishSettings;
  onClose: () => void;
}) {
  const client = useQueryClient();
  const mutation = useMutation(projectPublishingMutationOptions(client, projectScope));
  const projectQuery = useProjectDetail(workspaceSlug, projectId);
  const [unpublishError, setUnpublishError] = useState("");
  const [settingsQuery] = useState(() =>
    client.getQueryCache().find({ queryKey: projectPublishingOptions(client, projectScope).queryKey, exact: true })
  );
  const refreshProject = () => {
    if (
      !settingsQuery ||
      client
        .getQueryCache()
        .find({ queryKey: projectPublishingOptions(client, projectScope).queryKey, exact: true }) !== settingsQuery
    )
      return;
    void projectQuery
      .refetch()
      .then((result) => {
        if (result.isError) throw result.error;
        return result;
      })
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Refresh failed",
          message: "Publishing changed, but project details could not refresh. Reload the page to refresh them.",
        })
      );
  };
  const form = useForm({
    defaultValues: defaults(settings),
    validators: {
      onSubmit: ({ value }): string | undefined =>
        layouts.some((layout) => value.view_props[layout]) ? undefined : "Select at least one view layout.",
    },
    onSubmit: async ({ value, formApi }) => {
      const result = await mutation.mutateAsync({ type: "save", id: settings.id, settings: value });
      if (result.settings) formApi.reset(defaults(result.settings));
      refreshProject();
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Publishing settings saved." });
    },
  });
  const submitting = useStore(form.store, (state) => state.isSubmitting);
  const dirty = useStore(form.store, (state) => !state.isDefaultValue);
  const selected = useStore(form.store, (state) => state.values.view_props);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const busy = submitting || mutation.isPending;
  const unpublish = async () => {
    if (!settings.id) return;
    setUnpublishError("");
    try {
      await mutation.mutateAsync({ type: "unpublish", id: settings.id });
      refreshProject();
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Project unpublished." });
    } catch {
      setUnpublishError("Could not unpublish the project. Please try again.");
    }
  };
  return (
    <ModalCore
      isOpen
      handleClose={() => {
        if (!busy) onClose();
      }}
      width={EModalWidth.XXL}
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {
            form.setErrorMap({
              onSubmit: { form: "Could not save publishing settings. Please try again.", fields: {} },
            });
          });
        }}
        className="space-y-5 p-5"
      >
        <div className="flex items-center justify-between gap-2">
          <h5 className="text-18 font-medium text-secondary">Publish project</h5>
          {settings.anchor && (
            <Button type="button" variant="error-fill" disabled={busy} onClick={() => void unpublish()}>
              Unpublish
            </Button>
          )}
        </div>
        {settings.anchor && <PublishedProjectLink anchor={settings.anchor} />}
        <fieldset disabled={busy} className="space-y-4">
          <legend className="mb-2 text-13">Views</legend>
          <div className="flex gap-4">
            {layouts.map((layout) => (
              <label key={layout} className="flex items-center gap-2 text-13 capitalize">
                <Checkbox
                  checked={!!selected[layout]}
                  onChange={() =>
                    form.setFieldValue("view_props", (current) => ({ ...current, [layout]: !current[layout] }))
                  }
                />
                {layout}
              </label>
            ))}
          </div>
          {toggleFields.map(({ name, label }) => (
            <form.Field key={name} name={name}>
              {(field) => (
                <div className="flex items-center justify-between gap-2">
                  <span className="text-13">{label}</span>
                  <ToggleSwitch
                    label={label}
                    value={field.state.value}
                    onChange={field.handleChange}
                    disabled={busy}
                    size="sm"
                  />
                </div>
              )}
            </form.Field>
          ))}
        </fieldset>
        {(formError || unpublishError) && (
          <p role="alert" className="text-13 text-danger-primary">
            {formError || unpublishError}
          </p>
        )}
        <div className="flex items-center justify-between gap-2 border-t border-subtle pt-4">
          <span className="flex items-center gap-1 text-13 text-placeholder">
            <GlobeIcon className="size-3.5" />
            Anyone with the link can access
          </span>
          <div className="flex gap-2">
            <Button type="button" variant="secondary" disabled={busy} onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              loading={submitting}
              disabled={busy || (!!settings.anchor && !dirty)}
            >
              {settings.anchor ? "Update settings" : "Publish"}
            </Button>
          </div>
        </div>
      </form>
    </ModalCore>
  );
}
function PublishedProjectLink({ anchor }: { anchor: string }) {
  const url = `${SPACE_BASE_URL.trim() || window.location.origin}${SPACE_BASE_PATH}/issues/${anchor}`;
  const copy = useMutation({
    mutationFn: () => copyTextToClipboard(url),
    onSuccess: () =>
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Link copied", message: "Published page link copied successfully." }),
  });
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 rounded-md border border-strong p-2">
        <a className="grow truncate text-13" href={url} target="_blank" rel="noopener noreferrer">
          {url}
        </a>
        <Button type="button" variant="secondary" onClick={() => copy.mutate()}>
          Copy link
        </Button>
      </div>
      <p className="text-13 text-accent-primary">This project is now live on the web.</p>
      {copy.isError && (
        <p role="alert" className="text-13 text-danger-primary">
          Could not copy the link.
        </p>
      )}
    </div>
  );
}
