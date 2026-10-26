/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it, vi } from "vitest";
import { InputColorPicker } from "./input-color-picker";

it("ignores palette changes when a pending save disables an already open picker", async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  const onChange = vi.fn();
  const render = (disabled: boolean) =>
    root.render(
      <InputColorPicker
        name="state"
        value="#778899"
        onChange={onChange}
        hasError={false}
        placeholder="Color"
        disabled={disabled}
      />
    );
  try {
    await act(async () => render(false));
    await act(async () => container.querySelector<HTMLButtonElement>("button")!.click());
    const swatch = container.querySelector<HTMLElement>('[title="#D0021B"]')!;
    expect(swatch).not.toBeNull();
    await act(async () => swatch.click());
    expect(onChange).toHaveBeenLastCalledWith("#d0021b");
    onChange.mockClear();
    await act(async () => render(true));
    expect(container.querySelector<HTMLInputElement>("input")!.disabled).toBe(true);
    expect(container.querySelector<HTMLButtonElement>("button")!.disabled).toBe(true);
    await act(async () => swatch.click());
    expect(onChange).not.toHaveBeenCalled();
  } finally {
    await act(async () => root.unmount());
    container.remove();
  }
});
