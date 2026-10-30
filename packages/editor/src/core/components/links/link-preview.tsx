/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { useClipboardCopy } from "@plane/hooks";
import { copyTextToClipboard } from "@plane/utils/string";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Link2Off } from "lucide-react";
import { CopyIcon, GlobeIcon, EditIcon } from "@plane/propel/icons";
// components
import type { LinkViewProps, LinkViews } from "@/components/links";

export function LinkPreview({
  viewProps,
  switchView,
}: {
  viewProps: LinkViewProps;
  switchView: (view: LinkViews) => void;
}) {
  const { editor, from, to, url } = viewProps;
  const scope = useMemo(() => ({ editor, from, to, url }), [editor, from, to, url]);
  const { copy } = useClipboardCopy(scope);

  const removeLink = () => {
    editor.view.dispatch(editor.state.tr.removeMark(from, to, editor.schema.marks.link));
    viewProps.closeLinkView();
  };

  const copyLinkToClipboard = async () => {
    const result = await copy(() => copyTextToClipboard(url));
    if (result === "copied") viewProps.closeLinkView();
    else if (result === "failed") {
      setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy link", message: "Please try again." });
    }
  };

  return (
    <div
      className="animate-in fade-in absolute top-0 left-0 max-w-max translate-y-1"
      style={{
        transition: "all 0.2s cubic-bezier(.55, .085, .68, .53)",
      }}
    >
      <div className="shadow-md flex items-center gap-3 rounded-sm border-2 border-subtle bg-layer-1 p-2 text-11 text-tertiary">
        <GlobeIcon width={14} height={14} className="inline-block" />
        <p>{url?.length > 40 ? url.slice(0, 40) + "..." : url}</p>
        <div className="flex gap-2">
          <button
            onClick={() => void copyLinkToClipboard()}
            className="cursor-pointer transition-colors hover:text-primary"
          >
            <CopyIcon width={14} height={14} className="inline-block" />
          </button>
          {editor.isEditable && (
            <>
              <button
                onClick={() => switchView("LinkEditView")}
                className="cursor-pointer transition-colors hover:text-primary"
              >
                <EditIcon width={14} height={14} className="inline-block" />
              </button>
              <button onClick={removeLink} className="cursor-pointer transition-colors hover:text-primary">
                <Link2Off size={14} className="inline-block" />
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
