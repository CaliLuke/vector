/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// plane-i18n
import { useTranslation } from "@plane/i18n";
// types
import type { TIssueAttachment } from "@plane/types";
// ui
import { AlertModalCore } from "@plane/ui";
// helper
import { getFileName } from "@plane/utils/attachment";
// hooks
// types
import type { TAttachmentOperations } from "../issue-detail-widgets/attachments/helper";

export type TAttachmentOperationsRemoveModal = Pick<TAttachmentOperations, "remove">;

type Props = {
  isOpen: boolean;
  onClose: () => void;
  attachment: TIssueAttachment | undefined;
  attachmentOperations: TAttachmentOperationsRemoveModal;
};

export function IssueAttachmentDeleteModal(props: Props) {
  const { t } = useTranslation();
  const { isOpen, onClose, attachment, attachmentOperations } = props;
  // states
  const [loader, setLoader] = useState(false);

  // handlers
  const handleClose = () => {
    onClose();
    setLoader(false);
  };

  const handleDeletion = async (assetId: string) => {
    setLoader(true);
    try {
      await attachmentOperations.remove(assetId);
      handleClose();
    } catch {
      setLoader(false);
    }
  };

  if (!attachment) return <></>;
  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={() => handleDeletion(attachment.id)}
      isSubmitting={loader}
      isOpen={isOpen}
      title={t("attachment.delete")}
      content={
        <>
          {/* TODO: Translate here */}
          Are you sure you want to delete attachment-{" "}
          <span className="font-bold">{getFileName(attachment.attributes.name)}</span>? This attachment will be
          permanently removed. This action cannot be undone.
        </>
      }
    />
  );
}
