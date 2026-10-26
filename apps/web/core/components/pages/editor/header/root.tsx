/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { logoPropsFromSelection } from "@/helpers/emoji.helper";
import type { TPage } from "@plane/types";

import { useState } from "react";
import { SmilePlus } from "lucide-react";
// plane imports
import { EmojiPicker, EmojiIconPickerTypes } from "@plane/propel/emoji-icon-picker";
import type { TChangeHandlerProps } from "@plane/propel/emoji-icon-picker";
import { cn } from "@plane/utils/common";
import { usePagePermissions } from "@/hooks/use-page-permissions";
import { useUpdateProjectPage } from "@/hooks/use-project-pages";
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
  // states
  const [isLogoPickerOpen, setIsLogoPickerOpen] = useState(false);
  const { isContentEditable } = usePagePermissions(page, workspaceSlug);
  // derived values
  const { logo_props, name } = page;
  const update = useUpdateProjectPage(workspaceSlug, projectId ?? page.project_ids?.[0], page.id);
  const updatePageLogo = (value: TChangeHandlerProps) =>
    update.mutateAsync({ logo_props: logoPropsFromSelection(value) });
  const isLogoSelected = !!logo_props?.in_use;
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
              isOpen={isLogoPickerOpen}
              handleToggle={(val) => setIsLogoPickerOpen(val)}
              className="flex items-center justify-center"
              buttonClassName="flex items-center justify-center"
              label={
                <span
                  className={cn(
                    "flex items-center gap-1 rounded-sm p-1 text-13 font-medium text-tertiary transition-colors outline-none hover:bg-layer-1",
                    {
                      "bg-layer-1": isLogoPickerOpen,
                    }
                  )}
                >
                  <SmilePlus className="size-4 flex-shrink-0" />
                  Icon
                </span>
              }
              onChange={updatePageLogo}
              defaultIconColor={
                logo_props?.in_use && logo_props.in_use === "icon" ? logo_props?.icon?.color : undefined
              }
              defaultOpen={
                logo_props?.in_use && logo_props?.in_use === "emoji"
                  ? EmojiIconPickerTypes.EMOJI
                  : EmojiIconPickerTypes.ICON
              }
              disabled={!isContentEditable}
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
