import { act, StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, expect, it, vi } from "vitest";
import * as Y from "yjs";
import { useYjsSetup } from "./use-yjs-setup";

const fixture = vi.hoisted(() => ({
  providers: [] as ProviderDouble[],
  caches: [] as CacheDouble[],
}));
type ProviderCallbacks = {
  name: string;
  onSynced: () => void;
  onAuthenticationFailed: () => void;
  onConnect: () => void;
  onStatus: (event: { status: string }) => void;
};
class ProviderDouble {
  document = new Y.Doc();
  configuration = {
    name: "",
    websocketProvider: {
      shouldConnect: true,
      webSocket: { readyState: WebSocket.CLOSED },
      disconnect: vi.fn(),
      connect: vi.fn(async () => {}),
      destroy: vi.fn(),
    },
  };
  close: ((event: { event: { code: number } }) => void) | undefined;
  destroy = vi.fn();
  constructor(readonly callbacks: ProviderCallbacks) {
    this.configuration.name = callbacks.name;
    fixture.providers.push(this);
  }
  on(_name: string, callback: NonNullable<ProviderDouble["close"]>) {
    this.close = callback;
  }
  off = vi.fn();
}
class CacheDouble {
  synced: (() => void) | undefined;
  destroy = vi.fn(async () => {});
  off = vi.fn();
  constructor(
    readonly name: string,
    readonly doc: Y.Doc
  ) {
    fixture.caches.push(this);
  }
  on(_name: string, callback: () => void) {
    this.synced = callback;
  }
}
vi.mock("@hocuspocus/provider", () => ({
  WebSocketStatus: { Connecting: "connecting", Connected: "connected" },
  HocuspocusProvider: vi.fn(function (callbacks: ProviderCallbacks) {
    return new ProviderDouble(callbacks);
  }),
}));
vi.mock("y-indexeddb", () => ({
  IndexeddbPersistence: vi.fn(function (name: string, doc: Y.Doc) {
    return new CacheDouble(name, doc);
  }),
}));

function Harness({
  docId = "first",
  authToken = "token",
  onStateChange,
  maxConnectionAttempts,
}: {
  docId?: string;
  authToken?: string;
  maxConnectionAttempts?: number;
  onStateChange?: Parameters<typeof useYjsSetup>[0]["onStateChange"];
}) {
  const session = useYjsSetup({
    docId,
    authToken,
    serverUrl: "ws://localhost",
    onStateChange,
    options: { maxConnectionAttempts },
  });
  return (
    <button type="button" onClick={() => session?.actions.signalForcedClose(true)}>
      {session
        ? `${session.provider.configuration.name}:${session.state.stage.kind}:${session.state.hasCachedContent}:${session.state.isCacheReady}`
        : "initial"}
    </button>
  );
}

async function setup() {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const host = document.createElement("div");
  const root = createRoot(host);
  const render = async (props = {}, strict = false) => {
    await act(async () =>
      root.render(
        strict ? (
          <StrictMode>
            <Harness {...props} />
          </StrictMode>
        ) : (
          <Harness {...props} />
        )
      )
    );
  };
  return {
    host,
    render,
    unmount: async () => {
      await act(async () => root.unmount());
    },
  };
}
afterEach(() => {
  for (const provider of fixture.providers) provider.document.destroy();
  fixture.providers.length = 0;
  fixture.caches.length = 0;
  vi.restoreAllMocks();
});

it("document and credential changes never attach the old document to the new cache", async () => {
  const view = await setup();
  try {
    await view.render();
    const first = fixture.providers[0];
    const cache = fixture.caches[0];
    await act(async () => {
      first.document.getXmlFragment("default").insert(0, [new Y.XmlText("cached")]);
      cache.synced?.();
      first.callbacks.onSynced();
    });
    expect(view.host.textContent).toContain("synced:true:true");
    await view.render({ docId: "second" });
    const second = fixture.providers[1];
    expect(fixture.caches.map((entry) => [entry.name, entry.doc])).toEqual([
      ["first", first.document],
      ["second", second.document],
    ]);
    expect(view.host.textContent).toContain("initial:false:false");
    await act(async () => {
      first.callbacks.onSynced();
      cache.synced?.();
      first.close?.({ event: { code: 4000 } });
    });
    expect(view.host.textContent).toContain("initial:false:false");
    await view.render({ docId: "second", authToken: "replacement" });
    expect(fixture.providers).toHaveLength(3);
    expect(fixture.caches[2]?.doc).toBe(fixture.providers[2]?.document);
    expect(first.destroy).toHaveBeenCalledOnce();
    expect(second.destroy).toHaveBeenCalledOnce();
  } finally {
    await view.unmount();
  }
});

it("cache and connection readiness follow the active document's content and retry budget", async () => {
  const view = await setup();
  try {
    await view.render();
    const provider = fixture.providers[0];
    await act(async () => {
      fixture.caches[0]?.synced?.();
    });
    expect(view.host.textContent).toContain("initial:false:true");
    await act(async () => {
      provider.document.getXmlFragment("default").insert(0, [new Y.XmlText("text")]);
    });
    expect(view.host.textContent).toContain("initial:true:true");
    await act(async () => {
      provider.document.getXmlFragment("default").delete(0, 1);
      provider.close?.({ event: { code: 1006 } });
    });
    expect(view.host.textContent).toContain("reconnecting:false:true");
    await act(async () => {
      provider.close?.({ event: { code: 1006 } });
      provider.close?.({ event: { code: 1006 } });
    });
    expect(view.host.textContent).toContain("disconnected:false:true");
    expect(provider.configuration.websocketProvider.shouldConnect).toBe(false);
    await act(async () => {
      provider.callbacks.onConnect();
      provider.callbacks.onSynced();
    });
    expect(view.host.textContent).toContain("synced:false:true");
  } finally {
    await view.unmount();
  }
});

it("rejected reconnects become state and late failures cannot disconnect a replacement session", async () => {
  const view = await setup();
  try {
    await view.render();
    const old = fixture.providers[0];
    old.configuration.websocketProvider.connect.mockRejectedValueOnce(new Error("network"));
    await act(async () => {
      window.dispatchEvent(new Event("online"));
    });
    expect(view.host.textContent).toContain("disconnected");
    let reject!: (error: Error) => void;
    old.configuration.websocketProvider.connect.mockReturnValueOnce(
      new Promise<void>((_resolve, fail) => {
        reject = fail;
      })
    );
    await act(async () => {
      window.dispatchEvent(new Event("online"));
    });
    await view.render({ docId: "replacement" });
    await act(async () => {
      reject(new Error("late network failure"));
    });
    expect(view.host.textContent).toContain("initial:false:false");
  } finally {
    await view.unmount();
  }
});

it("strict replay retires the old session and cleanup reports async failures without skipping resources", async () => {
  const report = vi.spyOn(console, "error").mockImplementation(() => {});
  const view = await setup();
  await view.render({}, true);
  const [old, current] = fixture.providers;
  expect(old?.destroy).toHaveBeenCalledOnce();
  await act(async () => {
    old?.callbacks.onSynced();
    fixture.caches[0]?.synced?.();
  });
  expect(view.host.textContent).toContain("initial:false:false");
  current?.configuration.websocketProvider.disconnect.mockImplementation(() => {
    throw new Error("disconnect");
  });
  fixture.caches[1]?.destroy.mockRejectedValueOnce(new Error("database close"));
  await view.unmount();
  expect(current?.destroy).toHaveBeenCalledOnce();
  expect(current?.configuration.websocketProvider.destroy).toHaveBeenCalledOnce();
  expect(fixture.caches[1]?.off).toHaveBeenCalledOnce();
  expect(report).toHaveBeenCalledOnce();
  expect(report.mock.calls[0]?.[1]).toBeInstanceOf(AggregateError);
  const calls = current?.configuration.websocketProvider.connect.mock.calls.length;
  window.dispatchEvent(new Event("online"));
  expect(current?.configuration.websocketProvider.connect.mock.calls.length).toBe(calls);
});

it("focus and visibility reconnect once, online retries, and provider status preserves sync stages", async () => {
  const visibility = vi.spyOn(document, "visibilityState", "get").mockReturnValue("hidden");
  const view = await setup();
  try {
    await view.render();
    const provider = fixture.providers[0];
    const socket = provider.configuration.websocketProvider;
    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"));
    });
    expect(socket.connect).not.toHaveBeenCalled();
    await act(async () => {
      window.dispatchEvent(new Event("focus"));
    });
    expect(socket.connect).toHaveBeenCalledOnce();
    expect(view.host.textContent).toContain("connecting");
    visibility.mockReturnValue("visible");
    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"));
      window.dispatchEvent(new Event("focus"));
    });
    expect(socket.connect).toHaveBeenCalledOnce();
    await act(async () => {
      window.dispatchEvent(new Event("online"));
      provider.callbacks.onStatus({ status: "connecting" });
    });
    expect(socket.connect).toHaveBeenCalledTimes(2);
    await act(async () => {
      provider.callbacks.onStatus({ status: "connected" });
    });
    expect(view.host.textContent).toContain("awaiting-sync");
    await act(async () => {
      provider.callbacks.onSynced();
      provider.callbacks.onStatus({ status: "disconnected" });
    });
    expect(view.host.textContent).toContain("synced");
    await view.render({ docId: "new" });
    await act(async () => {
      provider.callbacks.onConnect();
      provider.callbacks.onStatus({ status: "connecting" });
      provider.callbacks.onAuthenticationFailed();
    });
    expect(view.host.textContent).toContain("initial");
  } finally {
    await view.unmount();
  }
});

for (const mode of ["server", "manual", "signal"] as const)
  it(`a ${mode} forced close stops automatic retries until online resumes`, async () => {
    const view = await setup();
    try {
      await view.render();
      const provider = fixture.providers[0];
      const socket = provider.configuration.websocketProvider;
      if (mode === "manual") socket.shouldConnect = false;
      await act(async () => {
        if (mode === "signal") view.host.querySelector("button")?.click();
        provider.close?.({ event: { code: mode === "server" ? 4001 : 1000 } });
      });
      expect(view.host.textContent).toContain("disconnected");
      expect(socket.shouldConnect).toBe(false);
      expect(socket.disconnect).toHaveBeenCalledTimes(mode === "manual" ? 0 : 1);
      await act(async () => {
        window.dispatchEvent(new Event("online"));
      });
      expect(socket.shouldConnect).toBe(true);
      expect(socket.connect).toHaveBeenCalledOnce();
    } finally {
      await view.unmount();
    }
  });

it("custom retry budget is respected and an authentication failure is exposed", async () => {
  const view = await setup();
  const changed = vi.fn();
  try {
    await view.render({ maxConnectionAttempts: 2, onStateChange: changed });
    const provider = fixture.providers[0];
    await act(async () => {
      provider.close?.({ event: { code: 1006 } });
      provider.callbacks.onStatus({ status: "connecting" });
    });
    expect(changed).toHaveBeenLastCalledWith(expect.objectContaining({ stage: { kind: "reconnecting", attempt: 1 } }));
    await act(async () => {
      provider.close?.({ event: { code: 1006 } });
    });
    expect(changed).toHaveBeenLastCalledWith(
      expect.objectContaining({
        stage: { kind: "disconnected", error: { type: "max-retries", message: "Failed to connect after 2 attempts" } },
      })
    );
    await act(async () => {
      provider.callbacks.onAuthenticationFailed();
    });
    expect(changed).toHaveBeenLastCalledWith(
      expect.objectContaining({
        stage: { kind: "disconnected", error: { type: "auth-failed", message: "Authentication failed" } },
      })
    );
  } finally {
    await view.unmount();
  }
});
