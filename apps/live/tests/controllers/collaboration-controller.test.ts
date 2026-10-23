import { HocuspocusProvider, HocuspocusProviderWebsocket } from "@hocuspocus/provider";
import { Hocuspocus } from "@hocuspocus/server";
import { registerController } from "@plane/decorators";
import { describe, expect, it, vi } from "vitest";
import { WebSocket } from "ws";
import { Doc } from "yjs";
import { startLiveTestServer } from "../helpers/http-server";
import { CollaborationController, toHocuspocusRequest } from "@/controllers/collaboration.controller";

describe("collaboration websocket request adaptation", () => {
  it("converts Express upgrade data into a web Request with the original URL and headers", () => {
    const request = {
      headers: { cookie: "session=abc", host: "live.example.test", "x-forwarded-for": ["192.0.2.1", "192.0.2.2"] },
      get: () => "live.example.test",
      method: "GET",
      originalUrl: "/collaboration/?documentName=page-id",
      protocol: "https",
    } satisfies Parameters<typeof toHocuspocusRequest>[0];

    const webRequest = toHocuspocusRequest(request);
    expect(webRequest.url).toBe("https://live.example.test/collaboration/?documentName=page-id");
    expect(webRequest.headers.get("cookie")).toBe("session=abc");
    expect(webRequest.headers.get("x-forwarded-for")).toBe("192.0.2.1, 192.0.2.2");
  });

  it("omits undefined headers and uses localhost when Express has no host", () => {
    const request = {
      headers: { "x-optional": undefined },
      get: () => undefined,
      method: "GET",
      originalUrl: "/collaboration/",
      protocol: "http",
    } satisfies Parameters<typeof toHocuspocusRequest>[0];

    const webRequest = toHocuspocusRequest(request);
    expect(webRequest.url).toBe("http://localhost/collaboration/");
    expect(webRequest.headers.has("x-optional")).toBe(false);
  });
});

describe("collaboration websocket transport", () => {
  it.each(["nodebuffer", "arraybuffer", "fragments"] as const)(
    "authenticates clients, synchronizes %s messages in both directions, and releases closed connections",
    async (binaryType) => {
      const disconnected = vi.fn();
      const hocuspocus = new Hocuspocus({
        quiet: true,
        debounce: 10,
        onAuthenticate: async ({ token }) => {
          if (token !== "test-session") throw new Error("Unauthorized");
        },
        onDisconnect: disconnected,
      });
      const server = await startLiveTestServer(({ router }) => {
        router.ws("/collaboration/", (socket, _request, next) => {
          socket.binaryType = binaryType;
          next();
        });
        registerController(router, CollaborationController, [hocuspocus]);
      });
      const clients: HocuspocusProvider[] = [];
      const documents: Doc[] = [];
      const sockets: HocuspocusProviderWebsocket[] = [];
      const connect = () => {
        const document = new Doc();
        documents.push(document);
        const socket = new HocuspocusProviderWebsocket({
          url: server.baseUrl.replace("http:", "ws:") + "/collaboration/",
          WebSocketPolyfill: WebSocket,
        });
        sockets.push(socket);
        const provider = new HocuspocusProvider({
          websocketProvider: socket,
          name: "transport-regression",
          token: "test-session",
          document,
        });
        provider.attach();
        clients.push(provider);
        return { document, provider };
      };
      try {
        const first = connect();
        const second = connect();
        await expect
          .poll(
            () =>
              clients.map((client, index) => ({
                synced: client.isSynced,
                status: sockets[index].status,
                authenticated: client.isAuthenticated,
              })),
            { timeout: 2000 }
          )
          .toEqual([
            { synced: true, status: "connected", authenticated: true },
            { synced: true, status: "connected", authenticated: true },
          ]);
        first.document.getText("content").insert(0, "Shared draft");
        await expect.poll(() => second.document.getText("content").toJSON()).toBe("Shared draft");
        second.document.getText("content").insert(12, " edited");
        await expect.poll(() => first.document.getText("content").toJSON()).toBe("Shared draft edited");
        second.provider.destroy();
        await expect.poll(() => hocuspocus.getConnectionsCount()).toBe(1);
        expect(disconnected).toHaveBeenCalledOnce();
      } finally {
        clients.forEach((client) => client.destroy());
        sockets.forEach((socket) => socket.destroy());
        documents.forEach((document) => document.destroy());
        hocuspocus.closeConnections();
        hocuspocus.flushPendingStores();
        await server.close();
      }
    }
  );

  it("rejects unauthorized clients before loading a document", async () => {
    const loaded = vi.fn();
    const hocuspocus = new Hocuspocus({
      quiet: true,
      onAuthenticate: async () => {
        throw new Error("Unauthorized");
      },
      onLoadDocument: loaded,
    });
    const server = await startLiveTestServer(({ router }) => {
      registerController(router, CollaborationController, [hocuspocus]);
    });
    const denied = vi.fn();
    const document = new Doc();
    const socket = new HocuspocusProviderWebsocket({
      url: server.baseUrl.replace("http:", "ws:") + "/collaboration/",
      WebSocketPolyfill: WebSocket,
    });
    const provider = new HocuspocusProvider({
      websocketProvider: socket,
      name: "restricted-document",
      token: "invalid-session",
      document,
      onAuthenticationFailed: denied,
    });
    provider.attach();
    try {
      await expect.poll(() => denied.mock.calls.length, { timeout: 2000 }).toBeGreaterThan(0);
      expect(loaded).not.toHaveBeenCalled();
      expect(provider.isSynced).toBe(false);
      expect(hocuspocus.getConnectionsCount()).toBe(0);
    } finally {
      provider.destroy();
      socket.destroy();
      document.destroy();
      hocuspocus.closeConnections();
      hocuspocus.flushPendingStores();
      await server.close();
    }
  });
});
