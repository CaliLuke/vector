/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
// plane types
import type { ILinkDetails } from "@plane/types";
// components
import { ModulesLinksListItem } from "@/components/modules";
// hooks

type Props = {
  workspaceSlug: string | undefined;
  disabled?: boolean;
  handleDeleteLink: (linkId: string) => void;
  handleEditLink: (link: ILinkDetails) => void;
  links: readonly ILinkDetails[];
};

export function ModuleLinksList(props: Props) {
  const { links: moduleLinks, workspaceSlug, handleDeleteLink, handleEditLink, disabled } = props;
  // memoized link handlers
  const memoizedDeleteLink = useCallback((id: string) => handleDeleteLink(id), [handleDeleteLink]);
  const memoizedEditLink = useCallback((link: ILinkDetails) => handleEditLink(link), [handleEditLink]);

  if (!moduleLinks) return null;

  return (
    <>
      {moduleLinks.map((link) => (
        <ModulesLinksListItem
          key={link.id}
          handleDeleteLink={() => memoizedDeleteLink(link.id)}
          handleEditLink={() => memoizedEditLink(link)}
          isEditingAllowed={!disabled}
          link={link}
          workspaceSlug={workspaceSlug}
        />
      ))}
    </>
  );
}
