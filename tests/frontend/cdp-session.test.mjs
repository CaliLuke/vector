import assert from "node:assert/strict";
import { once } from "node:events";
import { createRequire } from "node:module";
import test from "node:test";
import { CdpSession } from "./helpers/cdp-session.mjs";

const requireLive = createRequire(new URL("../../apps/live/package.json", import.meta.url));
const { WebSocketServer } = requireLive("ws");
async function fixture(t) {
  const server = new WebSocketServer({ port: 0, host: "127.0.0.1" });
  t.after(async () => {
    for (const socket of server.clients) socket.terminate();
    await new Promise((resolve) => server.close(resolve));
  });
  await once(server, "listening");
  const connected = once(server, "connection");
  const session = new CdpSession(`ws://127.0.0.1:${server.address().port}`);
  t.after(() => session.close());
  await session.connect();
  const [socket] = await connected;
  return { session, socket };
}
void test("CDP correlates out-of-order command results, reports command errors, and delivers events", async (t) => {
  const { session, socket } = await fixture(t);
  const requests = [];
  socket.on("message", (data) => {
    requests.push(JSON.parse(data));
    if (requests.length === 2) {
      socket.send(JSON.stringify({ id: requests[1].id, result: { value: "second" } }));
      socket.send(JSON.stringify({ id: requests[0].id, error: { message: "invalid", code: -1 } }));
      socket.send(JSON.stringify({ method: "Page.loaded", params: { ready: true } }));
    }
  });
  const event = session.waitFor("Page.loaded", 1000);
  const first = assert.rejects(session.send("first"), /invalid \(-1\)/);
  const second = session.send("second");
  await first;
  assert.deepEqual(await second, { value: "second" });
  assert.deepEqual(await event, { ready: true });
});
for (const mode of ["malformed", "close", "listener"])
  void test(`CDP ${mode} failure rejects all pending commands and event waits`, async (t) => {
    const { session, socket } = await fixture(t);
    const command = assert.rejects(session.send("pending"));
    const event = assert.rejects(session.waitFor("never", 1000));
    if (mode === "malformed") socket.send("invalid JSON");
    else if (mode === "close") socket.close();
    else {
      session.on("broken", async () => {
        throw new Error("listener failed");
      });
      socket.send(JSON.stringify({ method: "broken" }));
    }
    await Promise.all([command, event]);
    await assert.rejects(session.send("later"));
    await assert.rejects(session.waitFor("later", 1000));
  });
void test("CDP event timeout does not poison the next event wait", async (t) => {
  const { session, socket } = await fixture(t);
  await assert.rejects(session.waitFor("Page.loaded", 10), /timed out/);
  const next = session.waitFor("Page.loaded", 1000);
  socket.send(JSON.stringify({ method: "Page.loaded", params: "ready" }));
  assert.equal(await next, "ready");
});
