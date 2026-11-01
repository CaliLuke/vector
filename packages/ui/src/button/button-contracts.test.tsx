/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { act, createRef } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Button } from "./button";
import { Badge } from "../badge/badge";

describe.each([
  ["Button", Button, "plane-ui-button"],
  ["Badge", Badge, "plane-ui-badge"],
] as const)("%s behavior", (_name, Component, displayName) => {
  let root: Root;
  let container: HTMLDivElement;

  beforeEach(() => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.unstubAllGlobals();
  });

  it("forwards its ref and native attributes and defaults to a non-submit button", async () => {
    const ref = createRef<HTMLButtonElement>();
    const onClick = vi.fn();
    await act(async () =>
      root.render(
        <Component ref={ref} name="action" onClick={onClick}>
          Run
        </Component>
      )
    );
    const button = container.querySelector("button");
    expect(ref.current).toBe(button);
    expect(button?.type).toBe("button");
    expect(button?.name).toBe("action");
    expect(button?.textContent).toBe("Run");
    expect(Component.displayName).toBe(displayName);
    await act(async () => button?.click());
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("retains custom styling, submit behavior and icons on both sides of the content", async () => {
    await act(async () =>
      root.render(
        <Component
          type="submit"
          size="sm"
          variant="primary"
          className="custom-control"
          prependIcon={<svg data-icon="before" strokeWidth={9} />}
          appendIcon={<svg data-icon="after" strokeWidth={1} />}
        >
          Save
        </Component>
      )
    );
    const button = container.querySelector("button");
    expect(button?.type).toBe("submit");
    expect(button?.classList.contains("custom-control")).toBe(true);
    expect(button?.children).toHaveLength(2);
    expect(button?.firstElementChild?.querySelector("svg")?.dataset.icon).toBe("before");
    expect(button?.lastElementChild?.querySelector("svg")?.dataset.icon).toBe("after");
    expect(button?.querySelector('[data-icon="before"]')?.getAttribute("stroke-width")).toBe("2");
    expect(button?.querySelector('[data-icon="after"]')?.getAttribute("stroke-width")).toBe("2");
    expect(button?.textContent).toBe("Save");
  });

  it.each(["loading", "disabled"] as const)("prevents clicks while %s", async (flag) => {
    const onClick = vi.fn();
    await act(async () =>
      root.render(
        <Component {...{ [flag]: true }} onClick={onClick}>
          Run
        </Component>
      )
    );
    const button = container.querySelector("button");
    expect(button?.disabled).toBe(true);
    await act(async () => button?.click());
    expect(onClick).not.toHaveBeenCalled();
  });
});
