// @vitest-environment jsdom

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { act, createRef, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { Tooltip as PropelTooltip } from "@plane/propel/tooltip";
import { Tooltip } from "./tooltip";

describe.each([
  ["legacy", Tooltip],
  ["propel", PropelTooltip],
] as const)("%s Tooltip target behavior", (_name, Component) => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  beforeAll(() => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  });

  afterEach(() => {
    if (root) act(() => root?.unmount());
    root = undefined;
    container?.remove();
    container = undefined;
    vi.useRealTimers();
  });

  const renderTooltip = (props: {
    renderByDefault?: boolean;
    disabled?: boolean;
    tooltipHeading?: string;
    tooltipContent?: ReactNode;
    openDelay?: number;
    onFocus?: () => void;
    onMouseEnter?: () => void;
  }) => {
    const { onFocus, onMouseEnter, ...tooltipProps } = props;
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
    const ref = createRef<HTMLButtonElement>();

    const rerender = (disabled = props.disabled) =>
      act(() => {
        root?.render(
          <Component tooltipContent="Tooltip body" openDelay={0} {...tooltipProps} disabled={disabled}>
            <button ref={ref} onFocus={onFocus} onMouseEnter={onMouseEnter}>
              Target
            </button>
          </Component>
        );
      });
    rerender();

    return { ref, target: container.querySelector("button"), rerender };
  };

  it("preserves the child element, ref, and mouse handler before the tooltip mounts", async () => {
    const onMouseEnter = vi.fn();
    const { ref, target } = renderTooltip({ renderByDefault: false, onMouseEnter });

    expect(target).not.toBeNull();
    expect(target?.parentElement).toBe(container);
    expect(ref.current).toBe(target);

    await act(() => target?.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })));
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
        <Component tooltipContent="Help">
          <button ref={childRef}>Target</button>
        </Component>
      );
    });
    expect(childRef).toHaveBeenCalledWith(container.querySelector("button"));
    act(() => root?.unmount());
    root = undefined;
    expect(cleanups.length).toBeGreaterThan(0);
    for (const cleanup of cleanups) expect(cleanup).toHaveBeenCalledOnce();
  });

  it("keeps a disabled tooltip's target focusable without showing its content", async () => {
    vi.useFakeTimers();
    const onFocus = vi.fn();
    const { target, ref } = renderTooltip({ disabled: true, tooltipContent: "Disabled help", onFocus });

    await act(async () => {
      target?.focus();
      await vi.runAllTimersAsync();
    });

    expect(document.activeElement).toBe(target);
    expect(ref.current).toBe(target);
    expect(onFocus).toHaveBeenCalledOnce();
    expect(document.body.textContent).not.toContain("Disabled help");
  });

  it("renders a heading and rich block content without invalid paragraph nesting", async () => {
    vi.useFakeTimers();
    const { target } = renderTooltip({
      tooltipHeading: "Details",
      tooltipContent: <section aria-label="Rich help">Extra information</section>,
    });

    await act(async () => {
      target?.focus();
      await vi.runAllTimersAsync();
    });

    const content = document.querySelector('section[aria-label="Rich help"]');
    expect(content).not.toBeNull();
    expect(content?.closest("p")).toBeNull();
    expect(document.body.textContent).toContain("Details");
  });

  it("closes when disabled and waits for a new interaction after being enabled", async () => {
    vi.useFakeTimers();
    const { target, rerender } = renderTooltip({ tooltipContent: "Conditional help" });
    await act(async () => {
      target?.focus();
      await vi.runAllTimersAsync();
    });
    expect(document.body.textContent).toContain("Conditional help");

    await act(async () => {
      rerender(true);
      await vi.runAllTimersAsync();
    });
    expect(document.body.textContent).not.toContain("Conditional help");

    await act(async () => {
      target?.blur();
      rerender(false);
      await vi.runAllTimersAsync();
    });
    expect(document.body.textContent).not.toContain("Conditional help");

    await act(async () => {
      target?.focus();
      await vi.runAllTimersAsync();
    });
    expect(document.body.textContent).toContain("Conditional help");
  });

  it("dismisses on Escape while retaining focus on the original target", async () => {
    vi.useFakeTimers();
    const { target } = renderTooltip({ tooltipContent: "Dismissible help" });
    await act(async () => {
      target?.focus();
      await vi.runAllTimersAsync();
    });
    expect(document.body.textContent).toContain("Dismissible help");

    await act(async () => {
      target?.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
      await vi.runAllTimersAsync();
    });

    expect(document.body.textContent).not.toContain("Dismissible help");
    expect(document.activeElement).toBe(target);
  });

  it("waits for the configured hover delay before showing content", async () => {
    vi.useFakeTimers();
    const { target } = renderTooltip({ tooltipContent: "Delayed help", openDelay: 80 });

    await act(async () => {
      target?.dispatchEvent(new MouseEvent("mouseover", { bubbles: true }));
      target?.dispatchEvent(new MouseEvent("mouseenter"));
      target?.dispatchEvent(new MouseEvent("mousemove", { bubbles: true }));
      await vi.advanceTimersByTimeAsync(79);
    });
    expect(document.body.textContent).not.toContain("Delayed help");

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(document.body.textContent).toContain("Delayed help");
  });
});
