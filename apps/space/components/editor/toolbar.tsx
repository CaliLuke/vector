/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type React from "react";
// plane imports
import { TOOLBAR_ITEMS, EditorToolbarButtons, useEditorToolbar } from "@plane/editor";
import type { ToolbarMenuItem, EditorRefApi } from "@plane/editor";
import { Button } from "@plane/propel/button";

type Props = {
  executeCommand: (item: ToolbarMenuItem) => void;
  handleSubmit: (event: React.MouseEvent<HTMLButtonElement>) => void;
  isCommentEmpty: boolean;
  isSubmitting: boolean;
  showSubmitButton: boolean;
  editorRef: EditorRefApi | null;
};

const toolbarItems = TOOLBAR_ITEMS.lite;

export function IssueCommentToolbar(props: Props) {
  const { executeCommand, handleSubmit, isCommentEmpty, editorRef, isSubmitting, showSubmitButton } = props;
  const { activeStates } = useEditorToolbar(editorRef, toolbarItems);

  return (
    <div className="flex h-9 w-full items-stretch gap-1.5 overflow-x-scroll">
      <div className="flex w-full items-stretch justify-between gap-2 rounded-sm border-[0.5px] border-subtle p-1">
        <div className="flex items-stretch">
          <EditorToolbarButtons
            groups={toolbarItems}
            activeStates={activeStates}
            executeCommand={executeCommand}
            variant="publicComment"
          />
        </div>
        {showSubmitButton && (
          <div className="sticky right-1">
            <Button
              type="button"
              variant="primary"
              onClick={handleSubmit}
              disabled={isCommentEmpty}
              loading={isSubmitting}
            >
              Comment
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
