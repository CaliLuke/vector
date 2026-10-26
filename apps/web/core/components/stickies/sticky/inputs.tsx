/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation } from "@tanstack/react-router";
import { useRef } from "react";

// plane imports
import type { EditorRefApi } from "@plane/editor";
import type { TSticky } from "@plane/types";
import { cn } from "@plane/utils/common";
import { isCommentEmpty } from "@plane/utils/string";
import { StickyEditor } from "@/components/editor/sticky-editor";
// hooks
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";

type TProps = {
  workspaceSlug: string;
  handleUpdate: (payload: Partial<TSticky>) => void;
  descriptionHtml: string;
  handleDescriptionChange: (value: string) => void;
  stickyId: string | undefined;
  showToolbar?: boolean;
  handleChange: (data: Partial<TSticky>) => Promise<void>;
  handleDelete: () => void;
};

export function StickyInput(props: TProps) {
  const {
    workspaceSlug,
    handleUpdate,
    descriptionHtml,
    handleDescriptionChange,
    stickyId,
    handleDelete,
    handleChange,
    showToolbar,
  } = props;
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  // navigation
  const pathname = useLocation({ select: (location) => location.pathname });
  // store hooks
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  // derived values
  const workspaceId = workspaceDetails?.id?.toString() ?? "";
  const isStickiesPage = pathname?.includes("stickies");
  return (
    <div className="flex-1">
      <StickyEditor
        id={`description-${stickyId}`}
        initialValue={descriptionHtml}
        value={null}
        workspaceSlug={workspaceSlug}
        workspaceId={workspaceId}
        onChange={(_description, description_html) => {
          handleDescriptionChange(description_html);
          handleUpdate({ description_html });
        }}
        placeholder={(_, value) => {
          const isContentEmpty = isCommentEmpty(value);
          if (!isContentEmpty) return "";
          return "Click to type here";
        }}
        containerClassName={cn(
          "vertical-scrollbar scrollbar-sm max-h-[540px] min-h-[256px] w-full overflow-y-scroll p-4 text-14",
          {
            "max-h-[588px]": isStickiesPage,
          }
        )}
        showToolbar={showToolbar}
        parentClassName="border-none p-0"
        handleDelete={handleDelete}
        handleColorChange={handleChange}
        ref={editorRef}
      />
    </div>
  );
}
