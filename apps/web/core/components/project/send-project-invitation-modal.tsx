/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { EUserPermissions, ROLE } from "@plane/constants";
import { EModalPosition, EModalWidth, ModalCore, Avatar, CustomSelect, CustomSearchSelect } from "@plane/ui";
import { Button } from "@plane/propel/button";
import { PlusIcon, CloseIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { getFileURL } from "@plane/utils/file";
import { useTranslation } from "@plane/i18n";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceMemberRoster, useProjectMemberRoster } from "@/hooks/use-member-rosters";
import { useAddProjectMembers } from "@/hooks/use-member-roster-mutations";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
  projectId: string;
  workspaceSlug: string;
};

type TMemberEntry = { id: string; member_id: string; role: EUserPermissions };

const roleOptions = [EUserPermissions.GUEST, EUserPermissions.MEMBER, EUserPermissions.ADMIN];
const createMemberEntry = (): TMemberEntry => ({
  id: crypto.randomUUID(),
  member_id: "",
  role: EUserPermissions.GUEST,
});

function allowedRoles(workspaceRole: number | undefined, projectRole: number | undefined) {
  if (projectRole === undefined) return [];
  return roleOptions.filter(
    (role) =>
      role <= projectRole &&
      ((workspaceRole !== EUserPermissions.GUEST && workspaceRole !== EUserPermissions.ADMIN) || role === workspaceRole)
  );
}

export function SendProjectInvitationModal(props: Props) {
  return props.isOpen ? <ProjectInvitationForm key={`${props.workspaceSlug}/${props.projectId}`} {...props} /> : null;
}

function ProjectInvitationForm({ onClose, onSuccess, projectId, workspaceSlug }: Props) {
  const { t } = useTranslation();
  const { data: currentUser } = useCurrentUser();
  const { projectRole } = useProjectPermissions(workspaceSlug, projectId);
  const canLoadRoster = projectRole !== undefined;
  const workspaceRoster = useWorkspaceMemberRoster(workspaceSlug);
  const projectRoster = useProjectMemberRoster(workspaceSlug, projectId, { canLoad: canLoadRoster });
  const addMembers = useAddProjectMembers(workspaceSlug, projectId, projectRole);
  const initial = useState(() => ({ members: [createMemberEntry()] }))[0];
  const form = useForm({
    defaultValues: initial,
    validators: {
      onSubmit: ({ value }): string | undefined => {
        const uniqueIds = new Set(value.members.map((member) => member.member_id));
        if (uniqueIds.size !== value.members.length) return "Select each member only once.";
        if (
          value.members.some((member) => {
            const target = workspaceRoster.membersById?.get(member.member_id);
            return (
              !target ||
              target.is_active !== true ||
              target.member.is_bot ||
              projectRoster.membershipsByMemberId?.has(member.member_id) ||
              !allowedRoles(target.role, projectRole).includes(member.role)
            );
          })
        )
          return "Select an active member and a role you can assign.";
        return undefined;
      },
    },
    onSubmit: async ({ value }) => {
      await addMembers.mutateAsync({
        members: value.members.map(({ member_id, role }) => ({ member_id, role })),
      });
      setToast({ title: "Success!", type: TOAST_TYPE.SUCCESS, message: "Members added successfully." });
      onClose();
      onSuccess?.();
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const validationError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const members = useStore(form.store, (state) => state.values.members);
  const rosterFailed = workspaceRoster.query.isError || projectRoster.query.isError;
  const rosterReady =
    canLoadRoster &&
    projectRole === EUserPermissions.ADMIN &&
    workspaceRoster.query.isSuccess &&
    projectRoster.query.isSuccess;
  const close = () => {
    if (!busy) onClose();
  };
  const options = (workspaceRoster.members ?? [])
    .filter((member) => member.is_active === true && !member.member.is_bot && member.member.id !== currentUser?.id)
    .filter((member) => !projectRoster.membershipsByMemberId?.has(member.member.id))
    .filter((member) => allowedRoles(member.role, projectRole).length > 0)
    .map(({ member }) => ({
      value: member.id,
      query: `${member.first_name} ${member.last_name} ${member.display_name}`,
      content: (
        <div className="flex items-center gap-2">
          <Avatar name={member.display_name} src={getFileURL(member.avatar_url)} />
          {member.display_name}
        </div>
      ),
    }));

  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <form
        className="space-y-5 p-5"
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {
            form.setErrorMap({ onSubmit: { form: "Could not add members. Please try again.", fields: {} } });
          });
        }}
        noValidate
      >
        <h3 className="text-16 font-medium text-primary">{t("project_settings.members.invite_members.title")}</h3>
        <p className="text-13 text-secondary">{t("project_settings.members.invite_members.sub_heading")}</p>
        {rosterFailed && (
          <p role="alert" className="text-13 text-danger-primary">
            Unable to load members. Please retry.
            <button
              type="button"
              className="ml-2 underline"
              onClick={() => {
                void workspaceRoster.query.refetch();
                void projectRoster.query.refetch();
              }}
            >
              Retry
            </button>
          </p>
        )}
        {members.map((member, index) => (
          <div key={member.id} className="flex items-start gap-3">
            <form.Field
              name={`members[${index}].member_id`}
              validators={{ onChange: ({ value }) => (value ? undefined : "Please select a member") }}
            >
              {(field) => (
                <div className="grow">
                  <CustomSearchSelect
                    value={field.state.value}
                    disabled={busy || !rosterReady}
                    label={
                      workspaceRoster.membersById?.get(field.state.value)?.member.display_name ?? "Select co-worker"
                    }
                    onChange={(id: string) => {
                      field.handleChange(id);
                      const workspaceRole = workspaceRoster.membersById?.get(id)?.role;
                      const choices = allowedRoles(workspaceRole, projectRole);
                      form.setFieldValue(
                        `members[${index}].role`,
                        choices.find((role) => role === workspaceRole) ?? choices[0] ?? EUserPermissions.GUEST
                      );
                    }}
                    options={options.filter(
                      (option) =>
                        option.value === member.member_id || !members.some((row) => row.member_id === option.value)
                    )}
                  />
                  {field.state.meta.errors.map((message) => (
                    <p key={message} role="alert" className="text-13 text-danger-primary">
                      {message}
                    </p>
                  ))}
                </div>
              )}
            </form.Field>
            <form.Field name={`members[${index}].role`}>
              {(field) => (
                <CustomSelect
                  value={field.state.value}
                  onChange={field.handleChange}
                  disabled={busy || !rosterReady}
                  label={ROLE[field.state.value]}
                >
                  {allowedRoles(workspaceRoster.membersById?.get(member.member_id)?.role, projectRole).map((role) => (
                    <CustomSelect.Option key={role} value={role}>
                      {ROLE[role]}
                    </CustomSelect.Option>
                  ))}
                </CustomSelect>
              )}
            </form.Field>
            {members.length > 1 && (
              <button
                type="button"
                disabled={busy}
                aria-label="Remove member"
                onClick={() => void form.removeFieldValue("members", index)}
              >
                <CloseIcon className="size-4" />
              </button>
            )}
          </div>
        ))}
        {validationError && (
          <p role="alert" className="text-13 text-danger-primary">
            {validationError}
          </p>
        )}
        <div className="flex justify-between gap-2">
          <button
            type="button"
            disabled={busy || !rosterReady}
            className="flex items-center gap-2 text-13 text-accent-primary"
            onClick={() => form.pushFieldValue("members", createMemberEntry())}
          >
            <PlusIcon className="size-4" />
            {t("common.add_more")}
          </button>
          <div className="flex gap-2">
            <Button type="button" variant="secondary" disabled={busy} onClick={close}>
              {t("cancel")}
            </Button>
            <Button type="submit" variant="primary" loading={busy} disabled={busy || !rosterReady}>
              {t("add_members")}
            </Button>
          </div>
        </div>
      </form>
    </ModalCore>
  );
}
