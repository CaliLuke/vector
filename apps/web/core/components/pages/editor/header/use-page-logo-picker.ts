/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { EmojiIconPickerTypes } from "@plane/propel/emoji-icon-picker";
import type { TChangeHandlerProps } from "@plane/propel/emoji-icon-picker";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import type { TPage } from "@plane/types";
import { logoPropsFromSelection } from "@/helpers/emoji.helper";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { usePagePermissions } from "@/hooks/use-page-permissions";
import { useUpdateProjectPage } from "@/hooks/use-project-pages";

export function usePageLogoPicker(page: TPage, workspaceSlug: string, projectId?: string) {
  const [isOpen, handleToggle] = useState(false);
  const { isContentEditable } = usePagePermissions(page, workspaceSlug);
  const ownerProjectId = projectId ?? page.project_ids?.[0];
  const update = useUpdateProjectPage(workspaceSlug, ownerProjectId, page.id);
  const accountId = useCurrentUserId();
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const current = useCommittedScope(JSON.stringify([accountId, generation, workspaceSlug, ownerProjectId, page.id]));
  const logo = page.logo_props;
  const onChange = (value: TChangeHandlerProps) => {
    if (!isContentEditable) return;
    update.mutate(
      { logo_props: logoPropsFromSelection(value) },
      {
        onError: () => {
          if (current())
            setToast({ type: TOAST_TYPE.ERROR, title: "Could not update page icon", message: "Please try again." });
        },
      }
    );
  };
  return {
    isContentEditable,
    isLogoSelected: !!logo?.in_use,
    pickerProps: {
      isOpen,
      handleToggle,
      onChange,
      defaultIconColor: logo?.in_use === "icon" ? logo.icon?.color : undefined,
      defaultOpen: logo?.in_use === "emoji" ? EmojiIconPickerTypes.EMOJI : EmojiIconPickerTypes.ICON,
      disabled: !isContentEditable,
    },
  };
}
