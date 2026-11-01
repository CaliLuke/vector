/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createPopper, type Instance } from "@popperjs/core";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NESTED_MENU_POSITIONING } from "./nested-menu-position";

describe("nested menu positioning", () => {
  let reference: HTMLButtonElement;
  let popup: HTMLDivElement;
  let instance: Instance | undefined;

  beforeEach(() => {
    reference = document.createElement("button");
    popup = document.createElement("div");
    document.body.append(reference, popup);
    vi.spyOn(document.documentElement, "clientWidth", "get").mockReturnValue(800);
    vi.spyOn(document.documentElement, "clientHeight", "get").mockReturnValue(600);
    vi.spyOn(reference, "offsetWidth", "get").mockReturnValue(24);
    vi.spyOn(reference, "offsetHeight", "get").mockReturnValue(24);
    vi.spyOn(popup, "offsetWidth", "get").mockReturnValue(160);
    vi.spyOn(popup, "offsetHeight", "get").mockReturnValue(80);
    vi.spyOn(popup, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 0, 160, 80));
  });

  afterEach(() => {
    instance?.destroy();
    instance = undefined;
    reference.remove();
    popup.remove();
    vi.restoreAllMocks();
  });

  it("places the submenu four pixels beyond its trigger using fixed positioning", async () => {
    vi.spyOn(reference, "getBoundingClientRect").mockReturnValue(new DOMRect(400, 200, 24, 24));
    instance = createPopper(reference, popup, { ...NESTED_MENU_POSITIONING, placement: "right-start" });
    const state = await instance.update();
    expect(state.placement).toBe("right-start");
    expect(popup.style.position).toBe("fixed");
    expect(state.modifiersData?.popperOffsets).toEqual({ x: 428, y: 200 });
  });

  it("flips to the left rather than overflowing the viewport's right edge", async () => {
    vi.spyOn(reference, "getBoundingClientRect").mockReturnValue(new DOMRect(760, 200, 24, 24));
    instance = createPopper(reference, popup, { ...NESTED_MENU_POSITIONING, placement: "right-start" });
    const state = await instance.update();
    expect(state.placement).toBe("left-start");
    expect(state.modifiersData?.popperOffsets).toEqual({ x: 596, y: 200 });
  });
});
