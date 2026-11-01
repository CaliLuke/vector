/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useSyncExternalStore, useCallback } from "react";
// plane imports
import type { EditorRefApi, IMarking } from "@plane/editor";
import { cn } from "@plane/utils/common";
// components
import type { THeadingComponentProps } from "./heading-components";
import { OutlineHeading1, OutlineHeading2, OutlineHeading3 } from "./heading-components";

type Props = {
  className?: string;
  emptyState?: React.ReactNode;
  editorRef: EditorRefApi | null;
  setSidePeekVisible?: (sidePeekState: boolean) => void;
  showOutline?: boolean;
};

const EMPTY_HEADINGS: IMarking[] = [];

const HeadingComponent: {
  [key: number]: React.FC<THeadingComponentProps>;
} = {
  1: OutlineHeading1,
  2: OutlineHeading2,
  3: OutlineHeading3,
};

export function PageContentBrowser(props: Props) {
  const { className, editorRef, emptyState, setSidePeekVisible, showOutline = false } = props;
  const subscribe = useCallback((notify: () => void) => editorRef?.onHeadingChange(notify) ?? (() => {}), [editorRef]);
  const getSnapshot = useCallback(() => editorRef?.getHeadings() ?? EMPTY_HEADINGS, [editorRef]);
  const headings = useSyncExternalStore(subscribe, getSnapshot, () => EMPTY_HEADINGS);

  const handleOnClick = useCallback(
    (marking: IMarking) => {
      editorRef?.scrollSummary(marking);
      setSidePeekVisible?.(false);
    },
    [editorRef, setSidePeekVisible]
  );

  if (headings.length === 0) return emptyState ?? null;

  return (
    <div
      className={cn(
        "mt-2 flex h-full flex-col items-start gap-y-1",
        {
          "gap-y-2": showOutline,
        },
        className
      )}
    >
      {headings.map((marking) => {
        const Component = HeadingComponent[marking.level];
        if (!Component) return null;
        if (showOutline)
          return (
            <div
              key={`${marking.level}-${marking.sequence}`}
              className="h-0.5 flex-shrink-0 self-end rounded-xs bg-layer-3"
              style={{
                width: marking.level === 1 ? "20px" : marking.level === 2 ? "18px" : "14px",
              }}
            />
          );
        return (
          <Component
            key={`${marking.level}-${marking.sequence}`}
            marking={marking}
            onClick={() => handleOnClick(marking)}
          />
        );
      })}
    </div>
  );
}
