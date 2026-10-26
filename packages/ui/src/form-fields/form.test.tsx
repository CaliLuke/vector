/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { act } from "react";
import { createPortal } from "react-dom";
import { createRoot } from "react-dom/client";
import { useForm } from "@tanstack/react-form";
import { expect, it, vi } from "vitest";
import { Form } from "./root";

it("submits a modal form without submitting its enclosing React form", async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const host = document.createElement("div");
  const modal = document.createElement("div");
  document.body.append(host, modal);
  const root = createRoot(host);
  const saveParent = vi.fn(async () => {});
  const saveModal = vi.fn(async () => {});
  function ParentWithModal() {
    const parent = useForm({ defaultValues: {}, onSubmit: saveParent });
    const child = useForm({ defaultValues: {}, onSubmit: saveModal });
    return (
      <Form onSubmit={() => parent.handleSubmit()}>
        {createPortal(<Form onSubmit={() => child.handleSubmit()} />, modal)}
      </Form>
    );
  }
  try {
    await act(async () => root.render(<ParentWithModal />));
    await act(async () => {
      modal.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    });
    expect(saveModal).toHaveBeenCalledOnce();
    expect(saveParent).not.toHaveBeenCalled();
  } finally {
    await act(async () => root.unmount());
    host.remove();
    modal.remove();
  }
});
