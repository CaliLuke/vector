/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useEffectEvent, useMemo, useSyncExternalStore } from "react";
import { createYjsSession } from "@/helpers/yjs-session";
import type { CollaborationState } from "@/types/collaboration";

type UseYjsSetupArgs = {
  docId: string;
  serverUrl: string;
  authToken: string;
  onStateChange?: (state: CollaborationState) => void;
  options?: { maxConnectionAttempts?: number };
};

export function useYjsSetup({ docId, serverUrl, authToken, onStateChange, options }: UseYjsSetupArgs) {
  const maxConnectionAttempts = options?.maxConnectionAttempts ?? 3;
  const session = useMemo(
    () => createYjsSession({ docId, serverUrl, authToken, maxConnectionAttempts }),
    [docId, serverUrl, authToken, maxConnectionAttempts]
  );
  const snapshot = useSyncExternalStore(session.subscribe, session.getSnapshot, session.getServerSnapshot);
  useEffect(() => session.start(), [session]);
  const notifyState = useEffectEvent((stage: CollaborationState["stage"]) =>
    onStateChange?.({
      stage,
      isServerSynced: stage.kind === "synced",
      isServerDisconnected: stage.kind === "disconnected",
    })
  );
  const stage = snapshot?.state.stage;
  useEffect(() => {
    if (stage) notifyState(stage);
  }, [stage]);
  return snapshot;
}
