/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

// plane imports
import { EmojiPicker, Logo } from "@plane/propel/emoji-icon-picker";
import { cn } from "@plane/utils/common";
import { usePageLogoPicker } from "./use-page-logo-picker";
// store

type Props = {
  className?: string;
  page: TPage;
  projectId?: string;
  workspaceSlug: string;
};

export function PageEditorHeaderLogoPicker(props: Props) {
  const { className, page, projectId, workspaceSlug } = props;
  const { pickerProps, isContentEditable, isLogoSelected } = usePageLogoPicker(page, workspaceSlug, projectId);
  const { logo_props } = page;

  return (
    <div
      className={cn(className, "pointer-events-none max-h-0 transition-all duration-300 ease-linear", {
        "pointer-events-auto max-h-[56px]": isLogoSelected,
      })}
    >
      <EmojiPicker
        {...pickerProps}
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
      />
    </div>
  );
}
