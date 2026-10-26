/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { FormErrors } from "@plane/ui";
import { useRef, type ReactNode } from "react";
import { useLocation, useRouter } from "@tanstack/react-router";
import { useForm, useStore } from "@tanstack/react-form";
import { useQueryClient } from "@tanstack/react-query";
import { ORGANIZATION_SIZE, RESTRICTED_URLS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspace } from "@plane/types";
import { Input, CustomSelect } from "@plane/ui";
import { validateWorkspaceName, validateSlug } from "@plane/utils/validation";
import { useCreateWorkspace } from "@/hooks/use-workspaces";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { WorkspaceService } from "@/services/workspace.service";

type Props = {
  onSubmit: (workspace: IWorkspace) => Promise<void>;
  secondaryButton?: ReactNode;
  primaryButtonText?: { loading: string; default: string };
};
const workspaceService = new WorkspaceService();
const defaultValues = { name: "", slug: "", organization_size: "" };
const slugFromName = (name: string) => name.toLowerCase().trim().replace(/\s+/g, "-");

export function CreateWorkspaceForm({ onSubmit, secondaryButton, primaryButtonText }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const userId = useCurrentUserId();
  const routeHref = useLocation({ select: (location) => location.href });
  const router = useRouter();
  const create = useCreateWorkspace();
  const acceptedCreateScope = useRef<{
    identity: ReturnType<typeof captureIdentityScope>;
    href: string;
  } | null>(null);
  const form = useForm({
    defaultValues,
    validators: {
      onSubmitAsync: async ({ value }) => {
        if (create.data) return undefined;
        try {
          const availability = await workspaceService.workspaceSlugCheck(value.slug);
          return availability.status
            ? undefined
            : { fields: { slug: t("workspace_creation.errors.validation.url_already_taken") } };
        } catch {
          return { fields: { slug: "Could not verify this workspace URL. Please try again." } };
        }
      },
    },
    onSubmit: async ({ value }) => {
      let workspace = create.data;
      let continuationScope = acceptedCreateScope.current;
      try {
        if (!workspace) {
          continuationScope = { identity: captureIdentityScope(queryClient, userId), href: routeHref };
          workspace = await create.mutateAsync(value);
          acceptedCreateScope.current = continuationScope;
          if (
            !isIdentityScopeCurrent(queryClient, continuationScope.identity) ||
            router.state.location.href !== continuationScope.href
          )
            throw new Error("The account or page changed while the workspace was being created.");
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("workspace_creation.toast.success.title"),
            message: t("workspace_creation.toast.success.message"),
          });
        }
        if (
          !continuationScope ||
          !isIdentityScopeCurrent(queryClient, continuationScope.identity) ||
          router.state.location.href !== continuationScope.href
        )
          throw new Error("The account or page changed before setup could continue.");
        await onSubmit(workspace);
      } catch (error) {
        if (
          continuationScope &&
          isIdentityScopeCurrent(queryClient, continuationScope.identity) &&
          router.state.location.href === continuationScope.href
        ) {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("workspace_creation.toast.error.title"),
            message: workspace
              ? "Your workspace was created. Retry to continue."
              : t("workspace_creation.toast.error.message"),
          });
        }
        throw error;
      }
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const canSubmit = useStore(
    form.store,
    (state) => !isSubmitting && Boolean(state.values.name.trim() && state.values.slug && state.values.organization_size)
  );
  return (
    <form
      className="space-y-6 sm:space-y-9"
      onSubmit={(event) => {
        event.preventDefault();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => {
          form.setErrorMap({
            onSubmit: {
              form: create.data
                ? "Your workspace was created. Retry to continue."
                : "Could not create workspace. Please try again.",
              fields: {},
            },
          });
        });
      }}
    >
      <fieldset disabled={isSubmitting || create.isSuccess} className="space-y-6">
        <form.Field
          name="name"
          validators={{
            onChange: ({ value }) => {
              if (!value.trim()) return t("common.errors.required");
              if (value.length > 80) return t("workspace_creation.errors.validation.name_length");
              const result = validateWorkspaceName(value, true);
              return result === true ? undefined : result;
            },
          }}
        >
          {(field) => (
            <div className="flex flex-col gap-2 text-13">
              <label htmlFor="workspaceName">{t("workspace_creation.form.name.label")}</label>
              <Input
                id="workspaceName"
                name={field.name}
                value={field.state.value}
                onBlur={field.handleBlur}
                onChange={(event) => {
                  const previousSlug = slugFromName(field.state.value);
                  field.handleChange(event.target.value);
                  if (!form.state.values.slug || form.state.values.slug === previousSlug)
                    form.setFieldValue("slug", slugFromName(event.target.value));
                }}
                hasError={field.state.meta.errors.length > 0}
                placeholder={t("workspace_creation.form.name.placeholder")}
                className="w-full"
              />
              <FormErrors errors={field.state.meta.errors} className="text-11 text-danger-primary" />
            </div>
          )}
        </form.Field>
        <form.Field
          name="slug"
          validators={{
            onChange: ({ value }) => {
              if (!value) return t("common.errors.required");
              if (value.length > 48) return t("workspace_creation.errors.validation.url_length");
              if (RESTRICTED_URLS.includes(value)) return t("workspace_creation.errors.validation.url_already_taken");
              const result = validateSlug(value);
              return result === true ? undefined : result;
            },
          }}
        >
          {(field) => (
            <div className="flex flex-col gap-2 text-13">
              <label htmlFor="workspaceUrl">{t("workspace_creation.form.url.label")}</label>
              <div className="flex w-full items-center rounded-md border border-subtle bg-layer-2 px-3">
                <span className="text-12 whitespace-nowrap text-secondary">
                  {typeof window === "undefined" ? "" : window.location.host}/
                </span>
                <Input
                  id="workspaceUrl"
                  name={field.name}
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.value.toLowerCase())}
                  hasError={field.state.meta.errors.length > 0}
                  placeholder={t("workspace_creation.form.url.placeholder")}
                  className="block w-full rounded-md border-none bg-transparent !px-0 py-2 text-12"
                />
              </div>
              <FormErrors errors={field.state.meta.errors} className="text-11 text-danger-primary" />
            </div>
          )}
        </form.Field>
        <form.Field
          name="organization_size"
          validators={{
            onChange: ({ value }) => (ORGANIZATION_SIZE.includes(value) ? undefined : t("common.errors.required")),
          }}
        >
          {(field) => (
            <div className="flex flex-col gap-2 text-13">
              <span>{t("workspace_creation.form.organization_size.label")}</span>
              <CustomSelect
                value={field.state.value}
                onChange={(value: string) => field.handleChange(value)}
                disabled={isSubmitting || create.isSuccess}
                label={field.state.value || t("workspace_creation.form.organization_size.placeholder")}
                buttonClassName="border border-subtle bg-layer-2 !shadow-none !rounded-md"
                input
              >
                {ORGANIZATION_SIZE.map((size) => (
                  <CustomSelect.Option key={size} value={size}>
                    {size}
                  </CustomSelect.Option>
                ))}
              </CustomSelect>
              <FormErrors errors={field.state.meta.errors} className="text-11 text-danger-primary" />
            </div>
          )}
        </form.Field>
      </fieldset>
      <fieldset disabled={isSubmitting} className="flex items-center gap-4">
        {secondaryButton}
        <Button variant="primary" type="submit" size="xl" disabled={!canSubmit || isSubmitting} loading={isSubmitting}>
          {create.isSuccess
            ? "Continue"
            : t(
                isSubmitting
                  ? (primaryButtonText?.loading ?? "workspace_creation.button.loading")
                  : (primaryButtonText?.default ?? "workspace_creation.button.default")
              )}
        </Button>
      </fieldset>
      {typeof formError === "string" && (
        <p role="alert" className="text-13 text-danger-primary">
          {formError}
        </p>
      )}
    </form>
  );
}
