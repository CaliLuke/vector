/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useCallback, useState } from "react";
import { observer } from "mobx-react";
import type { DropzoneRootProps, FileRejection } from "react-dropzone";
import { useDropzone } from "react-dropzone";
import { PlusIcon } from "@plane/propel/icons";
// plane imports
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TIssueServiceType } from "@plane/types";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// plane web hooks
import { useFileSize } from "@/hooks/use-file-size";
// local imports
import { useAttachmentOperations } from "./helper";
import { IssueDetailWidgetButton } from "../widget-button";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  customButton?: React.ReactNode;
  disabled?: boolean;
  issueServiceType: TIssueServiceType;
};

export const IssueAttachmentActionButton = observer(function IssueAttachmentActionButton(props: Props) {
  const { workspaceSlug, projectId, issueId, customButton, disabled = false, issueServiceType } = props;
  // state
  const [isLoading, setIsLoading] = useState(false);
  // store hooks
  const { setLastWidgetAction, fetchActivities } = useIssueDetail(issueServiceType);
  // file size
  const { maxFileSize } = useFileSize();
  // operations
  const { operations: attachmentOperations } = useAttachmentOperations(
    workspaceSlug,
    projectId,
    issueId,
    issueServiceType
  );
  const customButtonDisabled =
    React.isValidElement<{ disabled?: boolean }>(customButton) && customButton.props.disabled === true;
  const actionDisabled = isLoading || disabled || customButtonDisabled;
  // handlers
  const handleFetchPropertyActivities = useCallback(() => {
    fetchActivities(workspaceSlug, projectId, issueId);
  }, [fetchActivities, workspaceSlug, projectId, issueId]);

  const onDrop = useCallback(
    (acceptedFiles: File[], rejectedFiles: FileRejection[]) => {
      const totalAttachedFiles = acceptedFiles.length + rejectedFiles.length;

      if (rejectedFiles.length === 0) {
        const currentFile: File = acceptedFiles[0];
        if (!currentFile || !workspaceSlug) return;

        setIsLoading(true);
        attachmentOperations
          .create(currentFile)
          .catch(() => {
            setToast({
              type: TOAST_TYPE.ERROR,
              title: "Error!",
              message: "File could not be attached. Try uploading again.",
            });
          })
          .finally(() => {
            handleFetchPropertyActivities();
            setLastWidgetAction("attachments");
            setIsLoading(false);
          });
        return;
      }

      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message:
          totalAttachedFiles > 1
            ? "Only one file can be uploaded at a time."
            : `File must be of ${maxFileSize / 1024 / 1024}MB or less in size.`,
      });
      return;
    },
    [attachmentOperations, maxFileSize, workspaceSlug, handleFetchPropertyActivities, setLastWidgetAction]
  );

  const { getRootProps, getInputProps } = useDropzone({
    onDrop,
    maxSize: maxFileSize,
    multiple: false,
    disabled: actionDisabled,
  });

  const actionButton =
    React.isValidElement<{ disabled?: boolean }>(customButton) &&
    (customButton.type === "button" || customButton.type === IssueDetailWidgetButton) ? (
      React.cloneElement(customButton, { disabled: actionDisabled })
    ) : (
      <button type="button" disabled={actionDisabled}>
        {customButton ?? <PlusIcon className="h-4 w-4" />}
      </button>
    );

  const rootProps = getRootProps<DropzoneRootProps>({ tabIndex: -1, role: "presentation" });
  const handleRootClick = rootProps.onClick;

  return (
    <div
      {...rootProps}
      role="presentation"
      onKeyDown={rootProps.onKeyDown}
      onClick={(event) => {
        handleRootClick?.(event);
        event.stopPropagation();
      }}
    >
      <input {...getInputProps()} />
      {actionButton}
    </div>
  );
});
