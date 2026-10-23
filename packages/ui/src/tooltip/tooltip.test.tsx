// @vitest-environment jsdom

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { act, createRef } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { Tooltip } from "./tooltip";

describe("Tooltip target behavior", () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  beforeAll(() => {
    (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
  });

  afterEach(() => {
    if (root) act(() => root?.unmount());
    root = undefined;
    container?.remove();
    container = undefined;
    vi.useRealTimers();
  });

  const renderTooltip = (props: { renderByDefault?: boolean; onFocus?: () => void; onMouseEnter?: () => void }) => {
    const { onFocus, onMouseEnter, ...tooltipProps } = props;
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
    const ref = createRef<HTMLButtonElement>();

    act(() => {
      root?.render(
        <Tooltip tooltipContent="Tooltip body" openDelay={0} {...tooltipProps}>
          <button ref={ref} onFocus={onFocus} onMouseEnter={onMouseEnter}>
            Target
          </button>
        </Tooltip>
      );
    });

    return { ref, target: container.querySelector("button") };
  };

  it("preserves the child element, ref, and mouse handler before the tooltip mounts", () => {
    const onMouseEnter = vi.fn();
    const { ref, target } = renderTooltip({ renderByDefault: false, onMouseEnter });

    expect(target).not.toBeNull();
    expect(target?.parentElement).toBe(container);
    expect(ref.current).toBe(target);

    act(() => target?.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
    expect(onMouseEnter).toHaveBeenCalledOnce();
  });

  it.each([true, false])(
    "opens on keyboard focus with renderByDefault=%s and preserves the child handler",
    async (renderByDefault) => {
      vi.useFakeTimers();
      const onFocus = vi.fn();
      const { target, ref } = renderTooltip({ onFocus, renderByDefault });
      expect(target?.parentElement).toBe(container);

      await act(async () => {
        target?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
        await vi.runAllTimersAsync();
      });

      expect(onFocus).toHaveBeenCalledOnce();
      expect(document.body.textContent).toContain("Tooltip body");
      act(() => root?.unmount());
      root = undefined;
      expect(ref.current).toBeNull();
    }
  );

  it("runs a child callback ref's cleanup when the target unmounts", () => {
    const cleanups: Array<ReturnType<typeof vi.fn>> = [];
    const childRef = vi.fn(() => {
      const cleanup = vi.fn();
      cleanups.push(cleanup);
      return cleanup;
    });
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
    act(() => {
      root?.render(
        <Tooltip tooltipContent="Help">
          <button ref={childRef}>Target</button>
        </Tooltip>
      );
    });
    expect(childRef).toHaveBeenCalledWith(container.querySelector("button"));
    act(() => root?.unmount());
    root = undefined;
    expect(cleanups.length).toBeGreaterThan(0);
    for (const cleanup of cleanups) expect(cleanup).toHaveBeenCalledOnce();
  });
});
