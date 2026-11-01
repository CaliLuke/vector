import { describe, expect, it, vi } from "vitest";
import { ServerLifecycle } from "../src/server-lifecycle";

function pendingOperation() {
  let resolve!: () => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<void>((accept, fail) => {
    resolve = accept;
    reject = fail;
  });
  return { promise, resolve, reject };
}

function fixture() {
  const initialize = pendingOperation();
  const destroy = pendingOperation();
  const server = {
    initialize: vi.fn(() => initialize.promise),
    listen: vi.fn(),
    destroy: vi.fn(() => destroy.promise),
  };
  const factory = vi.fn(() => server);
  return { lifecycle: new ServerLifecycle(factory), initialize, destroy, server, factory };
}

describe("server lifecycle", () => {
  it("owns synchronous construction failures as startup rejections", async () => {
    const failure = new Error("invalid server configuration");
    const lifecycle = new ServerLifecycle(() => {
      throw failure;
    });
    await expect(lifecycle.start()).rejects.toBe(failure);
    await expect(lifecycle.stop()).rejects.toBe(failure);
  });

  it("initializes once, listens after initialization, and shares shutdown completion", async () => {
    const f = fixture();
    const started = f.lifecycle.start();
    expect(f.lifecycle.start()).toBe(started);
    expect(f.factory).toHaveBeenCalledTimes(1);
    expect(f.server.listen).not.toHaveBeenCalled();
    f.initialize.resolve();
    await started;
    expect(f.server.listen).toHaveBeenCalledTimes(1);
    const stopped = f.lifecycle.stop();
    expect(f.lifecycle.stop()).toBe(stopped);
    await Promise.resolve();
    expect(f.server.destroy).toHaveBeenCalledTimes(1);
    f.destroy.resolve();
    await stopped;
  });

  it("waits for startup during shutdown without opening a listener afterward", async () => {
    const f = fixture();
    const started = f.lifecycle.start();
    const stopped = f.lifecycle.stop();
    expect(f.server.destroy).not.toHaveBeenCalled();
    f.initialize.resolve();
    await started;
    expect(f.server.listen).not.toHaveBeenCalled();
    await Promise.resolve();
    expect(f.server.destroy).toHaveBeenCalledTimes(1);
    f.destroy.resolve();
    await stopped;
  });

  it("cleans up partial initialization and preserves its failure", async () => {
    const f = fixture();
    const failure = new Error("dependency unavailable");
    const started = f.lifecycle.start();
    const rejected = expect(started).rejects.toBe(failure);
    f.initialize.reject(failure);
    await rejected;
    const stopped = f.lifecycle.stop();
    const failedShutdown = expect(stopped).rejects.toBe(failure);
    f.destroy.resolve();
    await failedShutdown;
    expect(f.server.destroy).toHaveBeenCalledTimes(1);
  });

  it("propagates cleanup failures and cannot start after it was stopped", async () => {
    const f = fixture();
    f.initialize.resolve();
    await f.lifecycle.start();
    const failure = new Error("cleanup failed");
    const rejected = expect(f.lifecycle.stop()).rejects.toBe(failure);
    f.destroy.reject(failure);
    await rejected;
    const factory = vi.fn(() => f.server);
    const stopped = new ServerLifecycle(factory);
    await stopped.stop();
    await stopped.start();
    expect(factory).not.toHaveBeenCalled();
  });

  it("preserves both initialization and cleanup failures", async () => {
    const f = fixture();
    const startupError = new Error("startup failed");
    const cleanupError = new Error("cleanup failed");
    const startup = expect(f.lifecycle.start()).rejects.toBe(startupError);
    f.initialize.reject(startupError);
    await startup;
    const shutdown = expect(f.lifecycle.stop()).rejects.toMatchObject({ errors: [startupError, cleanupError] });
    f.destroy.reject(cleanupError);
    await shutdown;
  });
});
