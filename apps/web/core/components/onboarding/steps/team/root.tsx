/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { InviteMemberFields } from "../../invite-member-input";
import { useMutation } from "@tanstack/react-query";
import { useStore } from "@tanstack/react-form";
import { useInvitationForm, emptyInvitation, type InvitationFormValues } from "@/hooks/use-workspace-invitation";
import { PlusIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EOnboardingSteps } from "@plane/types";
// hooks
import { useWorkspaces } from "@/hooks/use-workspaces";
// services
import { WorkspaceService } from "@/services/workspace.service";
// components
import { CommonOnboardingHeader, OnboardingActions } from "../common";

const workspaceService = new WorkspaceService();

type Props = {
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => Promise<void>;
};

export function InviteTeamStep(props: Props) {
  const { handleStepChange } = props;

  const { data: workspaces = [] } = useWorkspaces();
  const workspace = workspaces[0];

  const nextStep = async () => {
    await handleStepChange(EOnboardingSteps.INVITE_MEMBERS);
  };

  const invitation = useMutation({
    mutationFn: async (data: InvitationFormValues) => {
      if (!workspace) throw new Error("Workspace not found");
      return workspaceService.inviteWorkspace(workspace.slug, data);
    },
    onSuccess: () =>
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Invitations sent successfully." }),
    onError: () =>
      setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message: "Unable to send invitations. Please try again." }),
  });
  const onSubmit = async (data: InvitationFormValues) => {
    if (!invitation.isSuccess) await invitation.mutateAsync(data);
    try {
      await nextStep();
    } catch (error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Unable to continue",
        message: "Your invitations were sent. Retry to finish onboarding.",
      });
      throw error;
    }
  };

  const form = useInvitationForm({ onSubmit, initialRows: 3, allowEmptyRows: true });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const canSubmit = useStore(
    form.store,
    (state) => state.canSubmit && state.values.emails.some((entry) => entry.email.trim())
  );
  const appendField = () => form.pushFieldValue("emails", emptyInvitation());

  return (
    <form
      className="flex flex-col gap-10"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit().catch(() => undefined);
      }}
    >
      <CommonOnboardingHeader
        title="Invite your teammates"
        description="Work in Vector happens best with your team. Invite them now to use Vector to its potential."
      />
      <fieldset disabled={isSubmitting || invitation.isSuccess} className="w-full py-4 text-13">
        <div className="group relative mx-8 grid grid-cols-10 gap-4 py-2">
          <div className="col-span-6 px-1 text-13 font-medium text-secondary">Email</div>
          <div className="col-span-4 px-1 text-13 font-medium text-secondary">Role</div>
        </div>
        <div className="mb-3 space-y-3 sm:space-y-4">
          <InviteMemberFields form={form} emailColumnClassName="col-span-6" />
        </div>
        <button
          type="button"
          className="mx-8 flex items-center gap-1.5 bg-transparent text-13 font-medium text-accent-primary outline-accent-strong"
          onClick={appendField}
          disabled={isSubmitting}
        >
          <PlusIcon className="h-4 w-4" strokeWidth={2} />
          Add another
        </button>
      </fieldset>
      <OnboardingActions
        isSubmitting={isSubmitting}
        canSubmit={canSubmit}
        onSkip={nextStep}
        skipLabel={invitation.isSuccess ? "Finish onboarding" : "I’ll do it later"}
      />
    </form>
  );
}
