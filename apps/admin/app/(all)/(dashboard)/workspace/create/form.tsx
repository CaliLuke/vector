/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useStore } from "@tanstack/react-form";
import { Link, useNavigate } from "react-router";
import { WEB_BASE_URL, ORGANIZATION_SIZE, RESTRICTED_URLS } from "@plane/constants";
import { Button, getButtonStyling } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { InstanceWorkspaceService } from "@plane/services";
import { validateSlug, validateWorkspaceName } from "@plane/utils/validation";
import { useSettingsForm, requiredField, SettingsForm } from "@/components/common/settings-form";
import { createWorkspaceOptions } from "@/queries/workspaces";

const service = new InstanceWorkspaceService();

export function WorkspaceCreateForm() {
  const navigate = useNavigate();
  const client = useQueryClient();
  const createWorkspace = useMutation(createWorkspaceOptions(client));
  const form = useSettingsForm({
    defaultValues: { name: "", slug: "", organization_size: "" },
    validators: {
      onSubmitAsync: async ({ value }) => {
        if (createWorkspace.data) return undefined;
        try {
          const availability = await service.slugCheck(value.slug);
          return availability.status ? undefined : { fields: { slug: "This URL is taken. Try something else." } };
        } catch {
          return { fields: { slug: "Could not verify this workspace URL. Please try again." } };
        }
      },
    },
    onSubmit: async ({ value }) => {
      try {
        if (!createWorkspace.data) await createWorkspace.mutateAsync(value);
      } catch (error) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Unable to create workspace",
          message: "Your changes are still available. Please try again.",
        });
        throw error;
      }
      try {
        await navigate("/workspace");
      } catch (error) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Workspace created",
          message: "Your workspace was created, but we could not open it. Please try again.",
        });
        throw error;
      }
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Workspace created", message: "Workspace created successfully." });
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  return (
    <SettingsForm onSubmit={() => form.handleSubmit()} className="space-y-8">
      <fieldset
        disabled={isSubmitting || createWorkspace.isSuccess}
        className="grid w-full max-w-4xl grid-cols-1 items-start gap-x-10 gap-y-6 lg:grid-cols-2"
      >
        <form.AppField
          name="name"
          validators={{
            onChange: ({ value }) => {
              const result = validateWorkspaceName(value, true);
              return result === true ? undefined : result;
            },
          }}
          listeners={{
            onChange: ({ value, fieldApi }) => {
              if (!fieldApi.form.getFieldMeta("slug")?.isTouched) {
                fieldApi.form.setFieldValue("slug", value.toLowerCase().trim().replace(/ /g, "-"), {
                  dontUpdateMeta: true,
                });
              }
            },
          }}
        >
          {(field) => (
            <field.TextField
              type="text"
              label="Name your workspace"
              placeholder="Something familiar and recognizable"
              required
            />
          )}
        </form.AppField>
        <form.AppField
          name="slug"
          validators={{
            onChange: ({ value }) => {
              const result = validateSlug(value);
              if (result !== true) return result;
              return RESTRICTED_URLS.includes(value) ? "This URL is reserved. Try something else." : undefined;
            },
          }}
        >
          {(field) => (
            <field.TextField
              type="text"
              label="Set your workspace's URL"
              placeholder="workspace-name"
              required
              description={
                WEB_BASE_URL
                  ? `Your workspace will be under ${WEB_BASE_URL}`
                  : "Your workspace will use this instance's address."
              }
            />
          )}
        </form.AppField>
        <form.Field name="organization_size" validators={{ onChange: requiredField("Workspace size") }}>
          {(field) => (
            <div className="flex flex-col gap-1">
              <label htmlFor="organization_size" className="text-13 text-tertiary">
                How many people will use this workspace?
              </label>
              <select
                id="organization_size"
                name={field.name}
                required
                value={field.state.value}
                onBlur={field.handleBlur}
                onChange={(event) => field.handleChange(event.target.value)}
                className="rounded-md border border-subtle bg-surface-1 px-3 py-2 text-13"
              >
                <option value="">Select a range</option>
                {ORGANIZATION_SIZE.map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
              {field.state.meta.errors.length > 0 && (
                <p role="alert" className="text-13 text-danger-primary">
                  {field.state.meta.errors.join(" ")}
                </p>
              )}
            </div>
          )}
        </form.Field>
      </fieldset>
      <div className="flex max-w-4xl items-center gap-4 py-1">
        <form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting, state.submissionAttempts]}>
          {([canSubmit, submitting, submissionAttempts]) => (
            <Button
              type="submit"
              variant="primary"
              size="lg"
              disabled={!canSubmit && submissionAttempts === 0}
              loading={Boolean(submitting)}
            >
              {submitting ? "Creating workspace" : "Create workspace"}
            </Button>
          )}
        </form.Subscribe>
        <Link className={getButtonStyling("secondary", "lg")} to="/workspace">
          Go back
        </Link>
      </div>
    </SettingsForm>
  );
}
