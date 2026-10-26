/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useState } from "react";
import { useLocation, useRouter } from "@tanstack/react-router";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ORGANIZATION_SIZE, EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspace } from "@plane/types";
import { CustomSelect, Input } from "@plane/ui";
import { copyUrlToClipboard } from "@plane/utils/string";
import { getFileURL } from "@plane/utils/file";
import { validateWorkspaceName } from "@plane/utils/validation";
import { WorkspaceImageUploadModal } from "@/components/core/modals/workspace-image-upload-modal";
import { TimezoneSelect } from "@/components/global/timezone-select";
import { useCurrentWorkspace, useUpdateWorkspace } from "@/hooks/use-workspaces";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { workspacesOptions } from "@/queries/workspaces";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { DeleteWorkspaceSection } from "@/components/workspace/delete-workspace-section";

export function WorkspaceDetails() {
  const { data: workspace } = useCurrentWorkspace();
  return workspace ? <WorkspaceDetailsForm key={workspace.id} workspace={workspace} /> : null;
}
function WorkspaceDetailsForm({ workspace }: { workspace: IWorkspace }) {
  const [showImage, setShowImage] = useState(false);
  const routeHref = useLocation({ select: (location) => location.href });
  const router = useRouter();
  const update = useUpdateWorkspace(workspace.slug);
  const queryClient = useQueryClient();
  const userId = useCurrentUserId();
  const { can: canWorkspace } = useWorkspacePermissions(workspace.slug);
  const isAdmin = canWorkspace([EUserPermissions.ADMIN]);
  const { t } = useTranslation();
  const label = (key: string) => t(`workspace_settings.settings.general.${key}`);
  const identityScope = captureIdentityScope(queryClient, userId);
  const routeWorkspaceId = workspace.id;
  const isCurrentScope = () =>
    isIdentityScopeCurrent(queryClient, identityScope) &&
    router.state.location.href === routeHref &&
    queryClient
      .getQueryData<IWorkspace[]>(workspacesOptions(userId).queryKey)
      ?.some((item) => item.id === routeWorkspaceId) === true;
  const form = useForm({
    defaultValues: {
      name: workspace.name,
      organization_size: workspace.organization_size,
      timezone: workspace.timezone,
    },
    onSubmit: async ({ value, formApi }) => {
      if (!isAdmin) return;
      const saved = await update.mutateAsync(value);
      if (!isCurrentScope()) return;
      formApi.reset({ name: saved.name, organization_size: saved.organization_size, timezone: saved.timezone });
      setToast({ title: "Success!", type: TOAST_TYPE.SUCCESS, message: "Workspace updated successfully." });
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const values = useStore(form.store, (state) => state.values);
  const copy = useMutation({
    mutationFn: () => copyUrlToClipboard(workspace.slug),
    onSuccess: () => setToast({ title: "Workspace URL copied to the clipboard.", type: TOAST_TYPE.SUCCESS }),
  });
  const refreshWorkspaceCatalogue = async () => {
    if (!isCurrentScope()) return;
    try {
      if (!userId) throw new Error("A signed-in user is required to refresh workspace details.");
      await queryClient.fetchQuery({ ...workspacesOptions(userId), staleTime: 0 });
    } catch {
      if (!isCurrentScope()) return;
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Workspace logo saved",
        message: "Workspace details could not refresh. Please reload the page.",
      });
    }
  };
  const removeLogo = async () => {
    if (!isAdmin) throw new Error("Administrator access is required");
    await refreshWorkspaceCatalogue();
  };
  const url = `${window.location.host}/${workspace.slug}`;
  return (
    <>
      {showImage && (
        <WorkspaceImageUploadModal
          isOpen
          workspaceId={workspace.id}
          workspaceSlug={workspace.slug}
          onClose={() => setShowImage(false)}
          handleRemove={removeLogo}
          onSuccess={refreshWorkspaceCatalogue}
          value={workspace.logo_url}
        />
      )}
      <form
        className="flex w-full flex-col gap-7"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {
            form.setErrorMap({ onSubmit: { form: "Could not update the workspace. Please try again.", fields: {} } });
          });
        }}
      >
        <div className="flex items-center gap-5">
          <button
            type="button"
            disabled={!isAdmin || busy}
            onClick={() => setShowImage(true)}
            aria-label={label("edit_logo")}
          >
            {workspace.logo_url ? (
              <img
                src={getFileURL(workspace.logo_url)}
                className="size-14 rounded-md object-cover"
                alt="Workspace logo"
              />
            ) : (
              <span className="grid size-14 place-items-center rounded-md bg-accent-primary text-24 text-on-color uppercase">
                {workspace.name.charAt(0)}
              </span>
            )}
          </button>
          <div className="flex flex-col gap-1">
            <h3 className="text-h5-semibold">{values.name}</h3>
            <button type="button" onClick={() => copy.mutate()} className="text-left text-body-xs-regular">
              {url}
            </button>
            {isAdmin && (
              <button
                type="button"
                disabled={busy}
                onClick={() => setShowImage(true)}
                className="text-left text-13 text-accent-primary"
              >
                {label(workspace.logo_url ? "edit_logo" : "upload_logo")}
              </button>
            )}
          </div>
        </div>
        {copy.isError && (
          <p role="alert" className="text-13 text-danger-primary">
            Could not copy the workspace URL.
          </p>
        )}
        <fieldset disabled={!isAdmin || busy} className="grid grid-cols-1 gap-10 xl:grid-cols-2 2xl:grid-cols-3">
          <form.Field
            name="name"
            validators={{
              onChange: ({ value }) => {
                const result = validateWorkspaceName(value, true);
                return result === true ? undefined : result;
              },
            }}
          >
            {(field) => (
              <label className="flex flex-col gap-2 text-body-sm-medium text-tertiary">
                {label("name")}
                <Input
                  name={field.name}
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  hasError={!field.state.meta.isValid}
                  className="w-full rounded-md"
                />
                {field.state.meta.errors.map((message) => (
                  <span key={String(message)} role="alert" className="text-11 text-danger-primary">
                    {message}
                  </span>
                ))}
              </label>
            )}
          </form.Field>
          <div className="flex flex-col gap-2">
            <h4 className="text-body-sm-medium text-tertiary">{label("company_size")}</h4>
            <CustomSelect
              value={values.organization_size}
              onChange={(value: string) => form.setFieldValue("organization_size", value)}
              label={values.organization_size}
              disabled={!isAdmin || busy}
              input
            >
              {ORGANIZATION_SIZE.map((size) => (
                <CustomSelect.Option key={size} value={size}>
                  {size}
                </CustomSelect.Option>
              ))}
            </CustomSelect>
          </div>
          <label className="flex flex-col gap-2 text-body-sm-medium text-tertiary">
            {label("url")}
            <Input value={url} disabled className="w-full rounded-md !bg-layer-1" />
          </label>
          <div className="flex flex-col gap-2">
            <h4 className="text-body-sm-medium text-tertiary">{label("workspace_timezone")}</h4>
            <TimezoneSelect
              value={values.timezone}
              onChange={(value) => form.setFieldValue("timezone", value)}
              disabled={!isAdmin || busy}
            />
          </div>
        </fieldset>
        {formError && (
          <p role="alert" className="text-13 text-danger-primary">
            {typeof formError === "string" ? formError : "Could not update the workspace. Please try again."}
          </p>
        )}
        {isAdmin && (
          <div>
            <Button type="submit" variant="primary" loading={busy}>
              {label("update_workspace")}
            </Button>
          </div>
        )}
      </form>
      {isAdmin && (
        <div className="mt-10">
          <DeleteWorkspaceSection workspace={workspace} />
        </div>
      )}
    </>
  );
}
