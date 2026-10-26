/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";
import { useStore } from "@tanstack/react-form";

// plane imports
import { useTranslation } from "@plane/i18n";
import type { IWorkspaceBulkInviteFormData } from "@plane/types";
import { EModalWidth, EModalPosition, ModalCore } from "@plane/ui";
// components
import { InvitationModalActions } from "@/components/workspace/invite-modal/actions";
import { InvitationFields } from "@/components/workspace/invite-modal/fields";
import { InvitationForm } from "@/components/workspace/invite-modal/form";
// hooks
import { useInvitationForm, emptyInvitation } from "@/hooks/use-workspace-invitation";

export type TSendWorkspaceInvitationModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: IWorkspaceBulkInviteFormData) => Promise<void> | undefined;
};

export function SendWorkspaceInvitationModal(props: TSendWorkspaceInvitationModalProps) {
  if (!props.isOpen) return null;
  return <WorkspaceInvitationForm {...props} />;
}

function WorkspaceInvitationForm({ onClose, onSubmit }: TSendWorkspaceInvitationModalProps) {
  // store hooks
  const { t } = useTranslation();
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const form = useInvitationForm({ onSubmit });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const handleClose = () => {
    if (!isSubmitting) onClose();
  };

  return (
    <ModalCore isOpen handleClose={handleClose} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <InvitationForm
        title={t("workspace_settings.settings.members.modal.title")}
        description={t("workspace_settings.settings.members.modal.description")}
        onSubmit={() => form.handleSubmit().catch(() => undefined)}
        actions={
          <InvitationModalActions
            isSubmitting={isSubmitting}
            handleClose={handleClose}
            appendField={() => form.pushFieldValue("emails", emptyInvitation())}
          />
        }
        className="p-5"
      >
        <InvitationFields workspaceSlug={workspaceSlug} form={form} />
      </InvitationForm>
    </ModalCore>
  );
}
