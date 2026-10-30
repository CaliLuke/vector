/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";

// plane imports
import { ETabIndices } from "@plane/constants";
import type { EditorRefApi } from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import { EFileAssetType } from "@plane/types";
import { getDescriptionPlaceholderI18n } from "@plane/utils/work-item";
import { getTabIndex } from "@plane/utils/tab-indices";
// components
import { RichTextEditor } from "@/components/editor/rich-text/editor";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// services
import { WorkspaceService } from "@/services/workspace.service";

const workspaceService = new WorkspaceService();

type TInboxIssueDescription = {
  containerClassName?: string;
  workspaceSlug: string;
  projectId: string;
  workspaceId: string;
  value: string;
  onChange: (value: string) => void;
  editorRef: RefObject<EditorRefApi | null>;
  onEnterKeyPress?: () => void;
  onAssetUpload?: (assetId: string) => void;
};

export const InboxIssueDescription = function InboxIssueDescription(props: TInboxIssueDescription) {
  const {
    containerClassName,
    workspaceSlug,
    projectId,
    workspaceId,
    value,
    onChange,
    editorRef,
    onEnterKeyPress,
    onAssetUpload,
  } = props;
  // i18n
  const { t } = useTranslation();
  // store hooks
  const { isMobile } = usePlatformOS();

  const { getIndex } = getTabIndex(ETabIndices.INTAKE_ISSUE_FORM, isMobile);

  return (
    <RichTextEditor
      editable
      id="inbox-modal-editor"
      initialValue={value}
      value={null}
      ref={editorRef}
      workspaceSlug={workspaceSlug}
      workspaceId={workspaceId}
      projectId={projectId}
      dragDropEnabled={false}
      onChange={(_description: object, description_html: string) => onChange(description_html)}
      placeholder={(isFocused, description) => t(`${getDescriptionPlaceholderI18n(isFocused, description)}`)}
      searchMentionCallback={async (payload) =>
        await workspaceService.searchEntity(workspaceSlug ?? "", {
          ...payload,
          project_id: projectId ?? "",
        })
      }
      containerClassName={containerClassName}
      onEnterKeyPress={onEnterKeyPress}
      tabIndex={getIndex("description_html")}
      fileAsset={{ entity_identifier: "", entity_type: EFileAssetType.ISSUE_DESCRIPTION }}
      onAssetUpload={onAssetUpload}
    />
  );
};
