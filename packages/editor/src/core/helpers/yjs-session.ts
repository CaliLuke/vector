/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { HocuspocusProvider, WebSocketStatus } from "@hocuspocus/provider";
import { IndexeddbPersistence } from "y-indexeddb";
import type { CollabStage } from "@/types/collaboration";

export type YjsSessionOptions = { docId: string; serverUrl: string; authToken: string; maxConnectionAttempts: number };

type SessionState = {
  stage: CollabStage;
  hasCachedContent: boolean;
  isCacheReady: boolean;
  isServerSynced: boolean;
  isServerDisconnected: boolean;
  isDocReady: boolean;
};

type SessionSnapshot = {
  provider: HocuspocusProvider;
  ydoc: HocuspocusProvider["document"];
  state: SessionState;
  actions: { signalForcedClose: (value: boolean) => void };
};

// Release every resource even if an earlier cleanup fails. Async persistence
// shutdown belongs here too; React cannot await an effect's destructor.
async function releaseSessionResources(release: (() => void | Promise<void>)[]) {
  const failures: unknown[] = [];
  await Promise.all(
    release.map(async (dispose) => {
      try {
        await dispose();
      } catch (error) {
        failures.push(error);
      }
    })
  );
  if (failures.length) console.error("Error releasing collaborative session resources", new AggregateError(failures));
}

const getServerSnapshot = () => null;

// Construction is pure. Only start(), called by the committed effect, opens
// connections. Each start owns its callbacks, including across Strict Mode replay.
export function createYjsSession({ docId, serverUrl, authToken, maxConnectionAttempts }: YjsSessionOptions) {
  let snapshot: SessionSnapshot | null = null;
  const listeners = new Set<() => void>();
  const notify = () => {
    for (const listener of listeners) listener();
  };

  const start = () => {
    let disposed = false;
    let retries = 0;
    let forcedClose = false;
    let lastReconnectTime = -Infinity;
    let stage: CollabStage = { kind: "initial" };
    let isCacheReady = false;
    let hasCachedContent = false;
    let provider: HocuspocusProvider | undefined;
    const actions = {
      signalForcedClose: (value: boolean) => {
        if (!disposed) forcedClose = value;
      },
    };
    const publish = () => {
      if (disposed || !provider) return;
      const isServerSynced = stage.kind === "synced";
      const isServerDisconnected = stage.kind === "disconnected";
      snapshot = {
        provider,
        ydoc: provider.document,
        actions,
        state: {
          stage,
          isCacheReady,
          hasCachedContent,
          isServerSynced,
          isServerDisconnected,
          isDocReady: isServerSynced || isServerDisconnected || (isCacheReady && hasCachedContent),
        },
      };
      notify();
    };
    const transition = (next: CollabStage) => {
      if (!disposed) {
        stage = next;
        publish();
      }
    };
    const connected = () => {
      if (!disposed) {
        retries = 0;
        transition({ kind: "awaiting-sync" });
      }
    };
    provider = new HocuspocusProvider({
      name: docId,
      token: authToken,
      url: serverUrl,
      onAuthenticationFailed: () =>
        transition({ kind: "disconnected", error: { type: "auth-failed", message: "Authentication failed" } }),
      onConnect: connected,
      onStatus: ({ status }) => {
        if (status === WebSocketStatus.Connecting)
          transition(retries > 0 ? { kind: "reconnecting", attempt: retries } : { kind: "connecting" });
        else if (status === WebSocketStatus.Connected) connected();
      },
      onSynced: () => {
        if (!disposed) {
          retries = 0;
          transition({ kind: "synced" });
        }
      },
    });
    const activeProvider = provider;
    const websocket = activeProvider.configuration.websocketProvider;
    const pause = () => {
      websocket.shouldConnect = false;
      websocket.disconnect();
    };
    const networkFailure = () =>
      transition({ kind: "disconnected", error: { type: "network-error", message: "Unable to reconnect" } });
    const reconnect = async () => {
      if (disposed) return;
      retries = 0;
      transition({ kind: "connecting" });
      try {
        websocket.shouldConnect = true;
        websocket.disconnect();
        await websocket.connect();
      } catch {
        networkFailure();
      }
    };
    const onClose = ({ event }: { event?: { code?: number } }) => {
      if (disposed) return;
      const code = event?.code ?? 0;
      const manual = !websocket.shouldConnect;
      if ((code >= 4000 && code <= 4003) || forcedClose || manual) {
        retries = 0;
        forcedClose = false;
        transition({
          kind: "disconnected",
          error: {
            type: "forced-close",
            code,
            message: manual ? "Manually disconnected" : "Server forced connection close",
          },
        });
        if (!manual) pause();
      } else if (++retries >= maxConnectionAttempts) {
        transition({
          kind: "disconnected",
          error: { type: "max-retries", message: `Failed to connect after ${maxConnectionAttempts} attempts` },
        });
        pause();
      } else transition({ kind: "reconnecting", attempt: retries });
    };
    const onVisibility = (event: Event) => {
      if (disposed || (document.visibilityState !== "visible" && event.type !== "focus")) return;
      const now = Date.now();
      if (now - lastReconnectTime < 1000) return;
      const socket = websocket.webSocket;
      if (
        socket?.readyState === WebSocket.CLOSED ||
        socket?.readyState === WebSocket.CLOSING ||
        stage.kind === "disconnected"
      ) {
        lastReconnectTime = now;
        void reconnect();
      }
    };
    const onOnline = () => {
      void reconnect();
    };
    const persistence = new IndexeddbPersistence(docId, activeProvider.document);
    const fragment = activeProvider.document.getXmlFragment("default");
    const onContent = () => {
      if (disposed || !isCacheReady) return;
      const hasContent = fragment.length > 0;
      if (hasContent !== hasCachedContent) {
        hasCachedContent = hasContent;
        publish();
      }
    };
    const onCacheReady = () => {
      if (disposed) return;
      isCacheReady = true;
      hasCachedContent = fragment.length > 0;
      publish();
    };
    persistence.on("synced", onCacheReady);
    fragment.observeDeep(onContent);
    activeProvider.on("close", onClose);
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("focus", onVisibility);
    window.addEventListener("online", onOnline);
    publish();

    return () => {
      if (disposed) return;
      disposed = true;
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("focus", onVisibility);
      window.removeEventListener("online", onOnline);
      snapshot = null;
      notify();
      void releaseSessionResources([
        () => {
          activeProvider.off("close", onClose);
        },
        () => {
          persistence.off("synced", onCacheReady);
        },
        () => {
          fragment.unobserveDeep(onContent);
        },
        pause,
        () => {
          websocket.destroy();
        },
        () => {
          activeProvider.destroy();
        },
        () => persistence.destroy(),
      ]);
    };
  };
  return {
    start,
    getSnapshot: () => snapshot,
    getServerSnapshot,
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
  };
}
