/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect, useState } from "react";
import { InviteMemberInput, type FormValues } from "../../invite-member-input";
import { observer } from "mobx-react";
import { useFieldArray, useForm } from "react-hook-form";
import { Button } from "@plane/propel/button";
import { PlusIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EOnboardingSteps } from "@plane/types";
import { Spinner } from "@plane/ui";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// services
import { WorkspaceService } from "@/services/workspace.service";
// components
import { CommonOnboardingHeader } from "../common";

const workspaceService = new WorkspaceService();

type Props = {
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => void;
};

export const InviteTeamStep = observer(function InviteTeamStep(props: Props) {
  const { handleStepChange } = props;

  const [isInvitationDisabled, setIsInvitationDisabled] = useState(true);

  const { workspaces } = useWorkspace();
  const workspacesList = Object.values(workspaces ?? {});
  const workspace = workspacesList[0];

  const {
    control,
    watch,
    getValues,
    setValue,
    handleSubmit,
    formState: { isSubmitting, errors, isValid },
  } = useForm<FormValues>();

  const { fields, append, remove } = useFieldArray({
    control,
    name: "emails",
  });

  const nextStep = async () => {
    await handleStepChange(EOnboardingSteps.INVITE_MEMBERS);
  };

  const onSubmit = async (formData: FormValues) => {
    if (!workspace) return;

    let payload = { ...formData };
    payload = { emails: payload.emails.filter((email) => email.email !== "") };

    await workspaceService
      .inviteWorkspace(workspace.slug, {
        emails: payload.emails.map((email) => ({
          email: email.email,
          role: email.role,
        })),
      })
      .then(async () => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Invitations sent successfully.",
        });
        await nextStep();
      })
      .catch((err) => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: err?.error,
        });
      });
  };

  const appendField = () => {
    append({ email: "", role: 15, role_active: false });
  };

  useEffect(() => {
    if (fields.length === 0) {
      append(
        [
          { email: "", role: 15, role_active: false },
          { email: "", role: 15, role_active: false },
          { email: "", role: 15, role_active: false },
        ],
        {
          focusIndex: 0,
        }
      );
    }
  }, [fields, append]);

  return (
    <form
      className="flex flex-col gap-10"
      onSubmit={handleSubmit(onSubmit)}
      onKeyDown={(e) => {
        if (e.code === "Enter") e.preventDefault();
      }}
    >
      <CommonOnboardingHeader
        title="Invite your teammates"
        description="Work in plane happens best with your team. Invite them now to use Vector to its potential."
      />
      <div className="w-full py-4 text-13">
        <div className="group relative mx-8 grid grid-cols-10 gap-4 py-2">
          <div className="col-span-6 px-1 text-13 font-medium text-secondary">Email</div>
          <div className="col-span-4 px-1 text-13 font-medium text-secondary">Role</div>
        </div>
        <div className="mb-3 space-y-3 sm:space-y-4">
          {fields.map((field, index) => (
            <InviteMemberInput
              emailColumnClassName="col-span-6"
              watch={watch}
              getValues={getValues}
              setValue={setValue}
              isInvitationDisabled={isInvitationDisabled}
              setIsInvitationDisabled={(value: boolean) => setIsInvitationDisabled(value)}
              control={control}
              errors={errors}
              fields={fields}
              index={index}
              remove={remove}
              key={field.id}
            />
          ))}
        </div>
        <button
          type="button"
          className="mx-8 flex items-center gap-1.5 bg-transparent text-13 font-medium text-accent-primary outline-accent-strong"
          onClick={appendField}
        >
          <PlusIcon className="h-4 w-4" strokeWidth={2} />
          Add another
        </button>
      </div>
      <div className="mx-auto flex w-full flex-col items-center justify-center gap-4 px-8 sm:px-2">
        <Button
          variant="primary"
          type="submit"
          size="xl"
          className="w-full"
          disabled={isInvitationDisabled || !isValid || isSubmitting}
        >
          {isSubmitting ? <Spinner height="20px" width="20px" /> : "Continue"}
        </Button>
        <Button variant="ghost" size="xl" className="w-full" onClick={nextStep}>
          I’ll do it later
        </Button>
      </div>
    </form>
  );
});
