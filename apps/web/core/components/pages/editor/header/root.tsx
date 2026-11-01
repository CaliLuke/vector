/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { SmilePlus } from "lucide-react";
// plane imports
import { EmojiPicker } from "@plane/propel/emoji-icon-picker";
import { cn } from "@plane/utils/common";
import { usePageLogoPicker } from "./use-page-logo-picker";
// store
// local imports
import { PageEditorHeaderLogoPicker } from "./logo-picker";

type Props = {
  page: TPage;
  projectId?: string;
  workspaceSlug: string;
};

export function PageEditorHeaderRoot(props: Props) {
  const { page, projectId, workspaceSlug } = props;
  const { pickerProps, isLogoSelected } = usePageLogoPicker(page, workspaceSlug, projectId);
  const { name } = page;
  const isTitleEmpty = !name || name.trim() === "";

  return (
    <>
      <div className="flex h-[48px] items-end text-left">
        {!isLogoSelected && (
          <div
            className={cn("opacity-0 transition-all duration-200 group-hover/page-header:opacity-100", {
              "opacity-100": isTitleEmpty,
            })}
          >
            <EmojiPicker
              {...pickerProps}
              className="flex items-center justify-center"
              buttonClassName="flex items-center justify-center"
              label={
                <span
                  className={cn(
                    "flex items-center gap-1 rounded-sm p-1 text-13 font-medium text-tertiary transition-colors outline-none hover:bg-layer-1",
                    {
                      "bg-layer-1": pickerProps.isOpen,
                    }
                  )}
                >
                  <SmilePlus className="size-4 flex-shrink-0" />
                  Icon
                </span>
              }
            />
          </div>
        )}
      </div>
      <PageEditorHeaderLogoPicker
        className="mt-2 flex w-full flex-shrink-0"
        page={page}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      />
    </>
  );
}
