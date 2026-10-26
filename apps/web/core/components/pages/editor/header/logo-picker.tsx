/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { logoPropsFromSelection } from "@/helpers/emoji.helper";
import type { TPage } from "@plane/types";

import { useState } from "react";
// plane imports
import { EmojiPicker, EmojiIconPickerTypes, Logo } from "@plane/propel/emoji-icon-picker";
import type { TChangeHandlerProps } from "@plane/propel/emoji-icon-picker";
import { cn } from "@plane/utils/common";
import { usePagePermissions } from "@/hooks/use-page-permissions";
import { useUpdateProjectPage } from "@/hooks/use-project-pages";
// store

type Props = {
  className?: string;
  page: TPage;
  projectId?: string;
  workspaceSlug: string;
};

export function PageEditorHeaderLogoPicker(props: Props) {
  const { className, page, projectId, workspaceSlug } = props;
  // states
  const [isLogoPickerOpen, setIsLogoPickerOpen] = useState(false);
  const { isContentEditable } = usePagePermissions(page, workspaceSlug);
  // derived values
  const { logo_props } = page;
  const update = useUpdateProjectPage(workspaceSlug, projectId ?? page.project_ids?.[0], page.id);
  const updatePageLogo = (value: TChangeHandlerProps) =>
    update.mutateAsync({ logo_props: logoPropsFromSelection(value) });
  const isLogoSelected = !!logo_props?.in_use;

  return (
    <div
      className={cn(className, "pointer-events-none max-h-0 transition-all duration-300 ease-linear", {
        "pointer-events-auto max-h-[56px]": isLogoSelected,
      })}
    >
      <EmojiPicker
        isOpen={isLogoPickerOpen}
        handleToggle={(val) => setIsLogoPickerOpen(val)}
        className="flex items-center justify-center"
        buttonClassName="flex items-center justify-center"
        label={
          <div
            className={cn("-ml-[8px] grid size-[56px] place-items-center rounded-sm transition-colors", {
              "hover:bg-layer-1": isContentEditable,
            })}
          >
            {isLogoSelected && <Logo logo={logo_props} size={48} type="lucide" />}
          </div>
        }
        onChange={updatePageLogo}
        defaultIconColor={logo_props?.in_use && logo_props.in_use === "icon" ? logo_props?.icon?.color : undefined}
        defaultOpen={
          logo_props?.in_use && logo_props?.in_use === "emoji" ? EmojiIconPickerTypes.EMOJI : EmojiIconPickerTypes.ICON
        }
        disabled={!isContentEditable}
      />
    </div>
  );
}
