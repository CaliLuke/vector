/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { act, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Tooltip as PropelTooltip } from "@plane/propel/tooltip";
import { Tooltip, type TPosition } from "./tooltip";

describe("Tooltip placement", () => {
  let root: Root;
  let container: HTMLDivElement;
  let targetBounds: DOMRect;

  beforeEach(() => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    vi.useFakeTimers();
    targetBounds = new DOMRect(740, 280, 40, 40);
    // JSDOM has no layout engine. Supply a viewport and measurable elements while
    // exercising the real positioning middleware and rendered tooltip.
    vi.spyOn(document.documentElement, "clientWidth", "get").mockReturnValue(800);
    vi.spyOn(document.documentElement, "clientHeight", "get").mockReturnValue(600);
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockImplementation(function (this: HTMLElement) {
      return this.getAttribute("role") === "tooltip" ? 120 : 40;
    });
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(40);
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (this: HTMLElement) {
      if (this.tagName === "BUTTON") return targetBounds;
      if (this.getAttribute("role") === "tooltip") return new DOMRect(0, 0, 120, 40);
      return new DOMRect(0, 0, 800, 600);
    });
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  async function openElement(element: ReactNode) {
    await act(async () => {
      root.render(element);
    });
    await act(async () => {
      container.querySelector("button")?.focus();
      await vi.runAllTimersAsync();
    });
    return document.querySelector('[role="tooltip"]');
  }

  function openTooltip(position: TPosition) {
    return openElement(
      <Tooltip position={position} tooltipContent="Positioned help">
        <button>Target</button>
      </Tooltip>
    );
  }

  it.each(["auto", "auto-start", "auto-end"] as const)(
    "%s chooses the side with most room even when top and bottom fit",
    async (position) => {
      const tooltip = await openTooltip(position);
      expect(tooltip?.getAttribute("data-placement")).toBe("left");
      expect(tooltip?.textContent).toContain("Positioned help");
    }
  );

  it.each([
    ["top", "top"],
    ["bottom", "bottom"],
    ["left", "left"],
    ["right", "right"],
    ["top-left", "top-start"],
    ["top-right", "top-end"],
    ["bottom-left", "bottom-start"],
    ["bottom-right", "bottom-end"],
    ["left-top", "left-start"],
    ["left-bottom", "left-end"],
    ["right-top", "right-start"],
    ["right-bottom", "right-end"],
  ] as const)("%s preserves its preferred placement when space permits", async (position, expected) => {
    targetBounds = new DOMRect(380, 280, 40, 40);
    const tooltip = await openTooltip(position);
    expect(tooltip?.getAttribute("data-placement")).toBe(expected);
  });

  it("flips a preferred side when the tooltip would cross the viewport edge", async () => {
    const tooltip = await openTooltip("right-top");
    expect(tooltip?.getAttribute("data-placement")).toBe("left-start");
  });

  it("honors the native tooltip's explicit side and alignment", async () => {
    await openElement(
      <PropelTooltip side="left" align="end" tooltipContent="Positioned help">
        <button>Target</button>
      </PropelTooltip>
    );
    const tooltip = document.querySelector("[data-side]");
    expect(tooltip?.textContent).toContain("Positioned help");
    expect(tooltip?.getAttribute("data-side")).toBe("left");
    expect(tooltip?.getAttribute("data-align")).toBe("end");
  });

  it.each([
    [{}, "top", "center"],
    [{ position: "bottom", side: "left", align: "end" }, "bottom", "center"],
  ] as const)("preserves native defaults and explicit position precedence: %j", async (props, side, align) => {
    await openElement(
      <PropelTooltip {...props} tooltipContent="Positioned help">
        <button>Target</button>
      </PropelTooltip>
    );
    const tooltip = document.querySelector("[data-side]");
    expect(tooltip?.getAttribute("data-side")).toBe(side);
    expect(tooltip?.getAttribute("data-align")).toBe(align);
  });

  it("dismisses when the child handles a custom context menu", async () => {
    const onContextMenu = vi.fn((event: React.MouseEvent) => event.preventDefault());
    await openElement(
      <Tooltip tooltipContent="Context help">
        <button onContextMenu={onContextMenu}>Target</button>
      </Tooltip>
    );
    expect(document.body.textContent).toContain("Context help");

    await act(async () => {
      container
        .querySelector("button")
        ?.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, cancelable: true }));
      await vi.runAllTimersAsync();
    });

    expect(onContextMenu).toHaveBeenCalledOnce();
    expect(document.body.textContent).not.toContain("Context help");
  });
});
