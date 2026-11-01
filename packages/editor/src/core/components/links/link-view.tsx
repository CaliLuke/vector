/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/react";
import { FloatingFocusManager } from "@floating-ui/react";
import type { FloatingContext } from "@floating-ui/react";
import { useMemo, useState } from "react";
// components
import { LinkEditView, LinkPreview } from "@/components/links";

export type LinkViews = "LinkPreview" | "LinkEditView";

export type LinkViewProps = {
  view?: LinkViews;
  editor: Editor;
  from: number;
  to: number;
  url: string;
  text?: string;
  closeLinkView: () => void;
};

export function LinkView(props: LinkViewProps & { focusContext: FloatingContext }) {
  const editorFocusTarget = useMemo(() => ({ current: props.editor.view.dom }), [props.editor]);
  const [currentView, setCurrentView] = useState<LinkViews>(props.view ?? "LinkPreview");
  const [prevFrom, setPrevFrom] = useState(props.from);

  const switchView = (view: LinkViews) => {
    setCurrentView(view);
  };

  if (props.from !== prevFrom) {
    setCurrentView("LinkPreview");
    setPrevFrom(props.from);
  }

  return (
    <>
      {currentView === "LinkPreview" && <LinkPreview viewProps={props} switchView={switchView} />}
      {currentView === "LinkEditView" && (
        <FloatingFocusManager context={props.focusContext} modal={false} returnFocus={editorFocusTarget} restoreFocus>
          <LinkEditView viewProps={props} />
        </FloatingFocusManager>
      )}
    </>
  );
}
