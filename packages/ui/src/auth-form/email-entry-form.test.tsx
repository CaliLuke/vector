/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EmailEntryForm } from "./email-entry-form";
import { AuthPasswordInput } from "./auth-password-input";
import { AuthConfirmPasswordInput } from "./auth-confirm-password-input";

let root: Root;
let container: HTMLDivElement;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});
afterEach(() => {
  act(() => root.unmount());
  container.remove();
});
async function change(input: HTMLInputElement, value: string) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const submit = () =>
  act(async () => {
    container.querySelector("form")?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
it("validates email, clears and refocuses it, and retains failed submissions for retry", async () => {
  const save = vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValue(undefined);
  act(() => root.render(<EmailEntryForm defaultEmail="" onSubmit={save} />));
  const input = container.querySelector("input")!;
  expect(container.querySelector<HTMLButtonElement>('[type="submit"]')!.disabled).toBe(true);
  await change(input, "invalid");
  await act(async () => {
    input.dispatchEvent(new FocusEvent("focusout", { bubbles: true }));
  });
  expect(container.querySelector('[role="alert"]')?.textContent).toBe("Email is invalid");
  await submit();
  expect(save).not.toHaveBeenCalled();
  await act(async () => {
    container.querySelector<HTMLButtonElement>('[aria-label="Clear email"]')!.click();
  });
  expect(input.value).toBe("");
  expect(document.activeElement).toBe(input);
  await change(input, "person@example.com");
  await submit();
  expect(input.value).toBe("person@example.com");
  expect(container.textContent).toContain("Could not continue");
  await submit();
  expect(save).toHaveBeenCalledTimes(2);
  expect(save).toHaveBeenLastCalledWith({ email: "person@example.com" });
  expect(container.textContent).not.toContain("Could not continue");
});
it("locks the email and clear controls until submission resolves", async () => {
  let finish!: () => void;
  const pending = new Promise<void>((resolve) => {
    finish = resolve;
  });
  act(() => root.render(<EmailEntryForm defaultEmail="person@example.com" onSubmit={() => pending} />));
  await submit();
  expect(container.querySelector("input")!.disabled).toBe(true);
  expect([...container.querySelectorAll("button")].every((button) => button.disabled)).toBe(true);
  await act(async () => {
    finish();
    await pending;
  });
  expect(container.querySelector("input")!.disabled).toBe(false);
});
it.each(["password", "confirmation"])("preserves %s field focus/blur handlers and autocomplete", async (kind) => {
  const focus = vi.fn(),
    blur = vi.fn();
  const props = {
    value: "long-password",
    autoComplete: "new-password",
    onFocus: focus,
    onBlur: blur,
    onChange: vi.fn(),
  };
  act(() =>
    root.render(
      kind === "password" ? (
        <AuthPasswordInput {...props} />
      ) : (
        <AuthConfirmPasswordInput {...props} password="long-password" />
      )
    )
  );
  const input = container.querySelector("input")!;
  expect(input.autocomplete).toBe("new-password");
  await act(async () => {
    input.focus();
    input.blur();
  });
  expect(focus).toHaveBeenCalledOnce();
  expect(blur).toHaveBeenCalledOnce();
  const toggle = container.querySelector<HTMLButtonElement>('[aria-label="Show password"]')!;
  act(() => toggle.click());
  expect(input.type).toBe("text");
  expect(toggle.getAttribute("aria-pressed")).toBe("true");
});
