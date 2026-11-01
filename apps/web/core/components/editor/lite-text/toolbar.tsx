/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type React from "react";
import type { LucideIcon } from "lucide-react";

import { EIssueCommentAccessSpecifier } from "@plane/constants";
// editor
import type { EditorRefApi } from "@plane/editor";
// i18n
import { useTranslation } from "@plane/i18n";
// ui
import { Button } from "@plane/propel/button";
import { GlobeIcon, LockIcon } from "@plane/propel/icons";
import type { ISvgIcons } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
// constants
import { cn } from "@plane/utils/common";
import type { ToolbarMenuItem } from "@plane/editor";
import { TOOLBAR_ITEMS, EditorToolbarButtons, useEditorToolbar } from "@plane/editor";
// helpers

type Props = {
  accessSpecifier?: EIssueCommentAccessSpecifier;
  executeCommand: (item: ToolbarMenuItem) => void;
  handleAccessChange?: (accessKey: EIssueCommentAccessSpecifier) => void;
  handleSubmit: (event: React.MouseEvent<HTMLButtonElement>) => void;
  isCommentEmpty: boolean;
  isSubmitting: boolean;
  showAccessSpecifier: boolean;
  showSubmitButton: boolean;
  editorRef: EditorRefApi | null;
  submitButtonText?: string;
};

type TCommentAccessType = {
  icon: LucideIcon | React.FC<ISvgIcons>;
  key: EIssueCommentAccessSpecifier;
  label: "Private" | "Public";
};

const COMMENT_ACCESS_SPECIFIERS: TCommentAccessType[] = [
  {
    icon: LockIcon,
    key: EIssueCommentAccessSpecifier.INTERNAL,
    label: "Private",
  },
  {
    icon: GlobeIcon,
    key: EIssueCommentAccessSpecifier.EXTERNAL,
    label: "Public",
  },
];

const toolbarItems = TOOLBAR_ITEMS.lite;

export function IssueCommentToolbar(props: Props) {
  const { t } = useTranslation();
  const {
    accessSpecifier,
    executeCommand,
    handleAccessChange,
    handleSubmit,
    isCommentEmpty,
    isSubmitting,
    showAccessSpecifier,
    showSubmitButton,
    editorRef,
    submitButtonText = "common.comment",
  } = props;
  const { activeStates, isReadyToDiscard } = useEditorToolbar(editorRef, toolbarItems);

  const isSubmitButtonDisabled = isCommentEmpty || !isReadyToDiscard;

  return (
    <div className="flex h-9 w-full items-stretch gap-1.5 overflow-x-scroll bg-surface-2">
      {showAccessSpecifier && (
        <div className="flex flex-shrink-0 items-stretch gap-0.5 rounded-sm border-[0.5px] border-subtle p-1">
          {COMMENT_ACCESS_SPECIFIERS.map((access) => {
            const isAccessActive = accessSpecifier === access.key;

            return (
              <Tooltip key={access.key} tooltipContent={access.label}>
                <button
                  type="button"
                  aria-label={access.label}
                  onClick={() => handleAccessChange?.(access.key)}
                  className={cn("grid aspect-square place-items-center rounded-xs p-1 hover:bg-layer-1", {
                    "bg-layer-1": isAccessActive,
                  })}
                >
                  <access.icon
                    className={cn("h-3.5 w-3.5 text-placeholder", {
                      "text-primary": isAccessActive,
                    })}
                    strokeWidth={2}
                  />
                </button>
              </Tooltip>
            );
          })}
        </div>
      )}
      <div className="flex w-full items-stretch justify-between gap-2 rounded-sm border-[0.5px] border-subtle p-1">
        <div className="flex items-stretch">
          <EditorToolbarButtons
            groups={toolbarItems}
            activeStates={activeStates}
            executeCommand={executeCommand}
            variant="comment"
          />
        </div>
        {showSubmitButton && (
          <div className="sticky right-1">
            <Button
              type="submit"
              variant="primary"
              className="px-2.5 py-1.5 text-11"
              onClick={handleSubmit}
              disabled={isSubmitButtonDisabled}
              loading={isSubmitting}
            >
              {t(submitButtonText)}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
