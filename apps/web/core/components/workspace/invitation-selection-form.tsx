/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Form } from "@plane/ui";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { ROLE } from "@plane/constants";
import { Button } from "@plane/propel/button";
import type { IWorkspaceMemberInvitation } from "@plane/types";
import { Checkbox, FormErrors } from "@plane/ui";
import { WorkspaceService } from "@/services/workspace.service";
import { useCurrentUser } from "@/hooks/use-current-user";
import {
  captureIdentityScope,
  getCurrentUserId,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { workspaceKeys } from "@/queries/workspaces";
import { WorkspaceLogo } from "./logo";

const service = new WorkspaceService();
type Props = {
  invitations: IWorkspaceMemberInvitation[];
  onJoined: (invitationIds: string[]) => Promise<void>;
  onCancel: () => void;
  submitLabel: string;
  cancelLabel: string;
};
export function InvitationSelectionForm({ invitations, onJoined, onCancel, submitLabel, cancelLabel }: Props) {
  const queryClient = useQueryClient();
  const router = useRouter();
  const { data: user } = useCurrentUser();
  const identityScope = captureIdentityScope(queryClient, user?.id);
  const routeHref = router.state.location.href;
  const join = useMutation({
    mutationFn: async (ids: string[]) => {
      if (!isIdentityScopeVerified(queryClient, identityScope) || router.state.location.href !== routeHref)
        throw new Error("The account session or invitations route changed.");
      await service.joinWorkspaces({ invitations: ids });
      if (isIdentityScopeCurrent(queryClient, identityScope) && identityScope) {
        await queryClient.invalidateQueries({ queryKey: workspaceKeys.list(identityScope.userId), exact: true });
      }
      return { ids, userId: identityScope?.userId, generation: identityScope?.generation };
    },
  });
  const form = useForm({
    defaultValues: { invitations: [] as string[] },
    onSubmit: async ({ value }) => {
      const result = join.data ?? (await join.mutateAsync(value.invitations));
      if (
        !identityScope ||
        result.userId !== getCurrentUserId(queryClient) ||
        result.generation !== getIdentitySessionGeneration(queryClient) ||
        !isIdentityScopeCurrent(queryClient, identityScope) ||
        router.state.location.href !== routeHref
      )
        throw new Error("The account session or invitations route changed.");
      await onJoined(result.ids);
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  return (
    <Form
      onSubmit={() => {
        form.setErrorMap({ onSubmit: undefined });
        return form.handleSubmit().catch(() => {
          form.setErrorMap({
            onSubmit: { form: "Could not finish joining these workspaces. Please try again.", fields: {} },
          });
        });
      }}
      className="space-y-6"
    >
      <fieldset
        disabled={busy || join.isSuccess}
        inert={busy || join.isSuccess}
        className="max-h-[37vh] space-y-3 overflow-y-auto"
      >
        <form.Field
          name="invitations"
          validators={{ onChange: ({ value }) => (value.length ? undefined : "Select at least one invitation.") }}
        >
          {(field) => (
            <>
              {invitations.map((invitation) => {
                const selected = field.state.value.includes(invitation.id);
                return (
                  <label
                    key={invitation.id}
                    htmlFor={`invitation-${invitation.id}`}
                    className="flex cursor-pointer items-center gap-3 rounded-lg border border-subtle px-3 py-3 hover:bg-surface-2"
                  >
                    <WorkspaceLogo
                      logo={invitation.workspace.logo_url}
                      name={invitation.workspace.name}
                      classNames="size-8 shrink-0 rounded-lg"
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-13 font-medium">{invitation.workspace.name}</span>
                      <span className="block text-11 text-secondary">{ROLE[invitation.role]}</span>
                    </span>
                    <Checkbox
                      id={`invitation-${invitation.id}`}
                      checked={selected}
                      onChange={(event) =>
                        field.handleChange(
                          event.target.checked
                            ? [...field.state.value, invitation.id]
                            : field.state.value.filter((id) => id !== invitation.id)
                        )
                      }
                    />
                  </label>
                );
              })}
              <FormErrors errors={field.state.meta.errors} />
            </>
          )}
        </form.Field>
      </fieldset>
      <FormErrors errors={[formError]} />
      <div className="flex items-center gap-3">
        <Button type="submit" variant="primary" size="lg" disabled={busy} loading={busy}>
          {submitLabel}
        </Button>
        <Button type="button" variant="secondary" size="lg" disabled={busy} onClick={onCancel}>
          {cancelLabel}
        </Button>
      </div>
    </Form>
  );
}
