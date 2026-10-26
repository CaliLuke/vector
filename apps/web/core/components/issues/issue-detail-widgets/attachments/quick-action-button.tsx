/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useCallback } from "react";

import type { DropzoneRootProps } from "react-dropzone";
import { PlusIcon } from "@plane/propel/icons";
// plane imports
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
// hooks
// plane web hooks
// local imports
import { AttachmentUploadError } from "../../attachment/upload-error";
import { useAttachmentDropzone } from "../../attachment/use-attachment-dropzone";
import type { TAttachmentHelpers } from "./helper";
import { IssueDetailWidgetButton } from "../widget-button";

type Props = {
  workspaceSlug: string;
  customButton?: React.ReactNode;
  disabled?: boolean;
  attachmentHelpers: TAttachmentHelpers;
};

export function IssueAttachmentActionButton(props: Props) {
  return <IssueAttachmentActionButtonContent key={props.attachmentHelpers.uploadScopeKey} {...props} />;
}

const IssueAttachmentActionButtonContent = function IssueAttachmentActionButtonContent(props: Props) {
  const { workspaceSlug, customButton, disabled = false, attachmentHelpers } = props;
  // state
  // store hooks
  const { setLastWidgetAction } = useIssueDetailUiState();
  // file size
  // operations
  const customButtonDisabled =
    React.isValidElement<{ disabled?: boolean }>(customButton) && customButton.props.disabled === true;
  const onUploaded = useCallback(() => setLastWidgetAction("attachments"), [setLastWidgetAction]);
  const { getRootProps, getInputProps, isUploading, error, canRetry, retry } = useAttachmentDropzone({
    workspaceSlug,
    create: attachmentHelpers.operations.create,
    disabled: disabled || customButtonDisabled,
    onUploaded,
  });
  const actionDisabled = isUploading || disabled || customButtonDisabled;

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
    <>
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
      <AttachmentUploadError error={error} canRetry={canRetry} retry={retry} disabled={actionDisabled} />
    </>
  );
};
