/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback, useEffect, useState } from "react";
import type { TNameDescriptionLoader } from "@plane/types";
import useReloadConfirmations from "@/hooks/use-reload-confirmation";

type TextStatuses = { title: TNameDescriptionLoader; description: TNameDescriptionLoader };
const saved: TextStatuses = { title: "saved", description: "saved" };

export function useTextSaveStatus(scope: string) {
  const [state, setState] = useState({ scope, ...saved });
  const statuses = state.scope === scope ? state : saved;
  const values = new Set([statuses.title, statuses.description]);
  const status: TNameDescriptionLoader = values.has("error")
    ? "error"
    : values.has("submitting")
      ? "submitting"
      : values.has("submitted")
        ? "submitted"
        : "saved";
  const unsaved = status === "submitting" || status === "error";
  const { setShowAlert } = useReloadConfirmations(unsaved);
  useEffect(() => setShowAlert(unsaved), [setShowAlert, unsaved]);
  useEffect(() => {
    if (status !== "submitted") return undefined;
    const timer = setTimeout(
      () => setState((current) => (current.scope === scope ? { scope, ...saved } : current)),
      2000
    );
    return () => clearTimeout(timer);
  }, [scope, status]);
  const setTitleStatus = useCallback(
    (title: TNameDescriptionLoader) =>
      setState((current) => ({ ...(current.scope === scope ? current : saved), scope, title })),
    [scope]
  );
  const setDescriptionStatus = useCallback(
    (description: TNameDescriptionLoader) =>
      setState((current) => ({ ...(current.scope === scope ? current : saved), scope, description })),
    [scope]
  );
  return { status, setTitleStatus, setDescriptionStatus };
}
