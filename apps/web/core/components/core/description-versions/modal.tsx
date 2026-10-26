/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useRef } from "react";
// plane imports
import type { EditorRefApi } from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { CopyIcon, ChevronLeftIcon, ChevronRightIcon } from "@plane/propel/icons";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import type { IUserLite, TDescriptionVersion } from "@plane/types";
import { Avatar, EModalPosition, EModalWidth, Loader, ModalCore } from "@plane/ui";
import { calculateTimeAgo } from "@plane/utils/datetime";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
// components
import { RichTextEditor } from "@/components/editor/rich-text";
// hooks
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import { useWorkspaceMarkdownCopy } from "@/hooks/use-workspace-markdown-copy";
import { IconButton } from "@plane/propel/icon-button";

type Props = {
  activeVersionDescription: string | undefined;
  activeVersionDetails: TDescriptionVersion | undefined;
  handleClose: () => void;
  handleNavigation: (direction: "prev" | "next") => void;
  handleRestore: (descriptionHTML: string) => void;
  isNextDisabled: boolean;
  isOpen: boolean;
  isPrevDisabled: boolean;
  isRestoreDisabled: boolean;
  versionCreator: IUserLite | undefined;
  projectId: string | undefined;
  workspaceSlug: string;
};

export function DescriptionVersionsModal(props: Props) {
  const {
    activeVersionDescription,
    activeVersionDetails,
    handleClose,
    handleNavigation,
    handleRestore,
    isNextDisabled,
    isPrevDisabled,
    isOpen,
    isRestoreDisabled,
    versionCreator,
    projectId,
    workspaceSlug,
  } = props;
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  // workspace
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  const {
    actionLabel: markdownActionLabel,
    actionDisabled: markdownActionDisabled,
    state: markdownCopyState,
    statusRole,
    statusMessage,
    handleAction,
  } = useWorkspaceMarkdownCopy(workspaceSlug);
  // derived values
  const activeVersionId = activeVersionDetails?.id;
  const workspaceId = workspaceDetails?.id;
  // translation
  const { t } = useTranslation();

  const handleCopyMarkdown = useCallback(() => {
    handleAction(
      () => editorRef.current,
      () => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: "Markdown copied to clipboard.",
        });
      }
    );
  }, [handleAction, t]);

  if (!workspaceId) return null;

  return (
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.TOP} width={EModalWidth.XXXXL}>
      <div className="p-4" data-prevent-outside-click>
        {/* Header */}
        <div className="flex items-center justify-between gap-2 py-0.5">
          <div className="flex flex-shrink-0 items-center gap-2 text-13">
            <p className="flex items-center gap-1">
              {t("description_versions.edited_by")}
              <span className="flex-shrink-0">
                <Avatar
                  size="sm"
                  src={getFileURL(versionCreator?.avatar_url ?? "")}
                  name={versionCreator?.display_name}
                />
              </span>
            </p>
            <p className="flex-shrink-0 text-secondary">
              {calculateTimeAgo(activeVersionDetails?.last_saved_at ?? "")}
            </p>
          </div>
          <div className="flex flex-shrink-0 items-center">
            <button
              type="button"
              onClick={() => handleNavigation("prev")}
              className={cn("grid size-6 place-items-center rounded-sm text-secondary transition-colors outline-none", {
                "hover:bg-layer-1": !isPrevDisabled,
                "opacity-50": isPrevDisabled,
              })}
              disabled={isPrevDisabled}
            >
              <ChevronLeftIcon className="size-4" />
            </button>
            <button
              type="button"
              onClick={() => handleNavigation("next")}
              className={cn("grid size-6 place-items-center rounded-sm text-secondary transition-colors outline-none", {
                "hover:bg-layer-1": !isNextDisabled,
                "opacity-50": isNextDisabled,
              })}
              disabled={isNextDisabled}
            >
              <ChevronRightIcon className="size-4" />
            </button>
          </div>
        </div>
        {/* End header */}
        {/* Version description */}
        <div className="mt-4 pb-4">
          {activeVersionId && activeVersionDescription ? (
            <RichTextEditor
              key={activeVersionId}
              editable={false}
              containerClassName="p-0 !pl-0 border-none"
              editorClassName="pl-0"
              id={activeVersionId}
              initialValue={activeVersionDescription}
              projectId={projectId}
              ref={editorRef}
              workspaceId={workspaceId}
              workspaceSlug={workspaceSlug}
            />
          ) : (
            <div className="space-y-1">
              <Loader.Item width="300px" height="15px" />
              <Loader.Item width="400px" height="15px" />
              <div className="flex items-center gap-2">
                <Loader.Item width="20px" height="15px" />
                <Loader.Item width="500px" height="15px" />
              </div>
              <div className="flex items-center gap-2">
                <Loader.Item width="20px" height="15px" />
                <Loader.Item width="200px" height="15px" />
              </div>
              <Loader.Item width="300px" height="15px" />
              <Loader.Item width="200px" height="15px" />
            </div>
          )}
        </div>
        {/* End version description */}
        {/* Footer */}
        <div className="flex items-center justify-between gap-2 border-t-[0.5px] border-subtle pt-4">
          <Tooltip
            tooltipContent={markdownCopyState === "ready" ? t("common.actions.copy_markdown") : markdownActionLabel}
          >
            <IconButton
              type="button"
              variant="ghost"
              size="base"
              onClick={handleCopyMarkdown}
              icon={CopyIcon}
              disabled={markdownActionDisabled}
              aria-label={markdownCopyState === "ready" ? t("common.actions.copy_markdown") : markdownActionLabel}
            />
          </Tooltip>
          {statusMessage && (
            <span role={statusRole} className="sr-only">
              {statusMessage}
            </span>
          )}
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="lg" onClick={handleClose}>
              {t("common.cancel")}
            </Button>
            {!isRestoreDisabled && (
              <Button
                variant="primary"
                size="lg"
                onClick={() => {
                  handleRestore(activeVersionDescription ?? "<p></p>");
                  handleClose();
                }}
              >
                {t("common.actions.restore")}
              </Button>
            )}
          </div>
        </div>
        {/* End footer */}
      </div>
    </ModalCore>
  );
}
