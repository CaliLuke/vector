/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useStore } from "@tanstack/react-form";
// plane imports
import { ROLE } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CloseIcon } from "@plane/propel/icons";
import { CustomSelect, Input } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { checkEmailValidity } from "@plane/utils/string";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import type { InvitationFormApi } from "@/hooks/use-workspace-invitation";

type TInvitationFieldsProps = {
  workspaceSlug: string;
  form: InvitationFormApi;
  className?: string;
};

export function InvitationFields(props: TInvitationFieldsProps) {
  const { workspaceSlug, form, className } = props;
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { workspaceRole: currentWorkspaceRole } = useWorkspacePermissions(workspaceSlug);
  // derived values

  return (
    <div className={cn("mb-3 space-y-4", className)}>
      <form.Field name="emails" mode="array">
        {(arrayField) =>
          arrayField.state.value.map((entry, index) => (
            <div
              key={entry.id}
              className="group relative mb-1 flex w-full items-start justify-between gap-x-4 text-body-xs-regular"
            >
              <div className="w-full">
                <form.Field
                  name={`emails[${index}].email`}
                  validators={{
                    onChange: ({ value }) =>
                      !value
                        ? t("workspace_settings.settings.members.modal.errors.required")
                        : !checkEmailValidity(value)
                          ? t("workspace_settings.settings.members.modal.errors.invalid")
                          : undefined,
                  }}
                >
                  {(field) => (
                    <>
                      <Input
                        id={field.name}
                        name={field.name}
                        type="email"
                        value={field.state.value}
                        onChange={(event) => field.handleChange(event.target.value)}
                        onBlur={field.handleBlur}
                        disabled={isSubmitting}
                        hasError={!field.state.meta.isValid}
                        placeholder={t("workspace_settings.settings.members.modal.placeholder")}
                        className="w-full text-caption-sm-regular sm:text-body-xs-regular"
                      />
                      {!field.state.meta.isValid && (
                        <span role="alert" className="ml-1 text-caption-sm-regular text-danger-primary">
                          {field.state.meta.errors.join(" ")}
                        </span>
                      )}
                    </>
                  )}
                </form.Field>
              </div>
              <div className="flex shrink-0 items-center justify-between gap-2">
                <div className="flex flex-col gap-1">
                  <form.Field name={`emails[${index}].role`}>
                    {(field) => (
                      <CustomSelect
                        value={field.state.value}
                        label={
                          <span className="text-caption-sm-regular sm:text-body-xs-regular">
                            {ROLE[field.state.value]}
                          </span>
                        }
                        onChange={field.handleChange}
                        disabled={isSubmitting}
                        className="w-24 flex-grow"
                        input
                      >
                        {Object.entries(ROLE)
                          .filter(([role]) => currentWorkspaceRole && currentWorkspaceRole >= Number(role))
                          .map(([role, label]) => (
                            <CustomSelect.Option key={role} value={Number(role)}>
                              {label}
                            </CustomSelect.Option>
                          ))}
                      </CustomSelect>
                    )}
                  </form.Field>
                </div>
                {arrayField.state.value.length > 1 && (
                  <div className="flex-item flex w-6">
                    <button
                      type="button"
                      className="place-items-center self-center rounded-sm"
                      disabled={isSubmitting}
                      aria-label="Remove invitation"
                      onClick={() => arrayField.removeValue(index)}
                    >
                      <CloseIcon className="h-4 w-4 text-secondary" />
                    </button>
                  </div>
                )}
              </div>
            </div>
          ))
        }
      </form.Field>
    </div>
  );
}
