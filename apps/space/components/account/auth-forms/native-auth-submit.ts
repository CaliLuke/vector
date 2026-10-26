/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FormEvent } from "react";
import { AuthService } from "@plane/services";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";

const authService = new AuthService();

export async function submitNativeAuthForm(form: HTMLFormElement | null) {
  try {
    await authService.submitForm(form);
  } catch (error) {
    setToast({ type: TOAST_TYPE.ERROR, title: "Could not sign in", message: "Please try again." });
    throw error;
  }
}

export function handleNativeAuthSubmit(event: FormEvent<HTMLFormElement>, handleSubmit: () => Promise<void>) {
  event.preventDefault();
  void handleSubmit().catch(() => {
    // The failed authentication is already reported with a toast.
  });
}
