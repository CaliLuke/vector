/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { existsSync } from "node:fs";
import { createServer as createTcpServer, createConnection } from "node:net";
import { execFile, spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";
import { setTimeout as delay } from "node:timers/promises";
import { Hocuspocus } from "@hocuspocus/server";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { AdminCommand } from "../../src/types/admin-commands";
import type { HealthCheckCommandData } from "../../src/types/admin-commands";
import type { Redis } from "../../src/extensions/redis";
import type { RedisManager } from "../../src/redis";

const run = promisify(execFile);

type ProcessExit = { code: number | null; signal: NodeJS.Signals | null; error?: Error };

const reserveEphemeralPort = async () => {
  const listener = createTcpServer();
  await new Promise<void>((resolve, reject) => {
    listener.once("error", reject);
    listener.listen(0, "127.0.0.1", resolve);
  });

  const address = listener.address();
  if (!address || typeof address === "string")
    throw new Error("Could not reserve an HTTP port for the native Live process");

  await new Promise<void>((resolve, reject) => {
    listener.close((error) => (error ? reject(error) : resolve()));
  });
  return address.port;
};

const observeProcessExit = (process: ReturnType<typeof spawn>) =>
  new Promise<ProcessExit>((resolve) => {
    process.once("error", (error) => resolve({ code: null, signal: null, error }));
    process.once("exit", (code, signal) => resolve({ code, signal }));
  });

const waitForHealth = async (
  healthUrl: string,
  exit: Promise<ProcessExit>,
  readOutput: () => string,
  timeoutMs: number
) => {
  const deadline = Date.now() + timeoutMs;
  const poll = async (): Promise<unknown> => {
    if (Date.now() >= deadline) {
      throw new Error(`Built Live process did not return health status OK within ${timeoutMs}ms.\n${readOutput()}`);
    }

    const exited = await Promise.race([exit, delay(0).then(() => undefined)]);
    if (exited)
      throw new Error(`Built Live process exited before becoming healthy: ${JSON.stringify(exited)}\n${readOutput()}`);

    try {
      const response = await fetch(healthUrl, { signal: AbortSignal.timeout(1000) });
      if (response.status === 200) {
        const body: unknown = await response.json();
        if (body && typeof body === "object" && "status" in body && body.status === "OK") return body;
      }
    } catch {
      // The listener may not be ready yet; retry until the bounded startup deadline.
    }

    return delay(100).then(poll);
  };

  return poll();
};

const stopProcess = async (process: ReturnType<typeof spawn>, exit: Promise<ProcessExit>) => {
  if (process.exitCode !== null || process.signalCode !== null) return await exit;
  process.kill("SIGTERM");
  const outcome = await Promise.race([exit, delay(10000, undefined, { ref: false }).then(() => undefined)]);
  if (outcome) return outcome;

  process.kill("SIGKILL");
  const forcedExit = await Promise.race([exit, delay(5000, undefined, { ref: false }).then(() => undefined)]);
  if (!forcedExit) throw new Error("Built Live process did not exit after SIGKILL");
  return forcedExit;
};

describe("Redis manager and collaboration transport", () => {
  let containerId: string | undefined;
  let redisUrl = "";
  let manager: RedisManager | undefined;
  const extensions: Redis[] = [];

  beforeAll(async () => {
    const container = await run("docker", [
      "run",
      "--detach",
      "--rm",
      "--publish",
      "127.0.0.1::6379",
      "valkey/valkey:9.1.2-alpine",
    ]);
    containerId = container.stdout.trim();
    const binding = await run("docker", ["port", containerId, "6379/tcp"]);
    const address = binding.stdout.trim();
    redisUrl = `redis://${address}`;
    // A published TCP port can accept connections before Valkey is ready.
    // Require a protocol response before spending the application's retry budget.
    const endpoint = new URL(redisUrl);
    await vi.waitFor(
      () =>
        new Promise<void>((resolve, reject) => {
          const socket = createConnection({ host: endpoint.hostname, port: Number(endpoint.port) });
          socket.setTimeout(500, () => socket.destroy(new Error("Redis readiness connection timed out")));
          socket.once("connect", () => socket.write("*1\r\n$4\r\nPING\r\n"));
          let response = "";
          socket.on("data", (data) => {
            response += data.toString();
            if (!response.includes("\r\n")) return;
            socket.destroy();
            if (response === "+PONG\r\n") resolve();
            else reject(new Error(`Unexpected Redis readiness response: ${response}`));
          });
          socket.once("error", reject);
          socket.once("close", () => reject(new Error("Redis readiness connection closed")));
        }),
      { timeout: 10000, interval: 100 }
    );
    vi.stubEnv("REDIS_URL", redisUrl);
    vi.stubEnv("API_BASE_URL", "http://api.test");
    vi.stubEnv("LIVE_SERVER_SECRET_KEY", "redis-integration-test");

    const { redisManager } = await import("../../src/redis");
    manager = redisManager;
    await manager.initialize();
    const { Redis: CollaborationRedis } = await import("../../src/extensions/redis");
    await Promise.all(
      Array.from({ length: 2 }, async () => {
        const extension = new CollaborationRedis();
        extensions.push(extension);
        const instance = new Hocuspocus();
        await extension.onConfigure({ instance, configuration: instance.configuration, version: "test" });
      })
    );
  }, 60000);

  afterAll(async () => {
    try {
      await Promise.all(extensions.map((extension) => extension.onDestroy()));
      await manager?.disconnect();
    } finally {
      vi.unstubAllEnvs();
      if (containerId) await run("docker", ["rm", "--force", containerId]);
    }
  }, 15000);

  it("stores, reads, expires and deletes keys through the application manager", async () => {
    if (!manager) throw new Error("Redis manager was not initialized");
    expect(manager.isClientConnected()).toBe(true);
    const key = `quality:${crypto.randomUUID()}`;
    expect(await manager.set(key, "first", 60)).toBe(true);
    expect(await manager.get(key)).toBe("first");
    expect(await manager.exists(key)).toBe(true);
    expect(await manager.expire(key, 0)).toBe(true);
    expect(await manager.get(key)).toBeNull();
    expect(await manager.set(key, "second")).toBe(true);
    expect(await manager.del(key)).toBe(true);
    expect(await manager.exists(key)).toBe(false);
  });

  it("delivers an admin command between independent collaboration instances", async () => {
    const [publisher, subscriber] = extensions;
    if (!publisher || !subscriber) throw new Error("Collaboration Redis instances were not initialized");
    const received: HealthCheckCommandData[] = [];
    subscriber.onAdminCommand<HealthCheckCommandData>(AdminCommand.HEALTH_CHECK, (command) => {
      received.push(command);
    });
    const command: HealthCheckCommandData = {
      command: AdminCommand.HEALTH_CHECK,
      originServer: "quality-publisher",
      timestamp: new Date().toISOString(),
    };
    expect(await publisher.publishAdminCommand(command)).toBeGreaterThanOrEqual(2);
    await expect.poll(() => received).toEqual([command]);
  });

  it("starts the built server, serves health, and exits cleanly on SIGTERM", async () => {
    if (!redisUrl) throw new Error("Disposable Redis URL was not initialized");
    const entrypoint = fileURLToPath(new URL("../../dist/start.mjs", import.meta.url));
    if (!existsSync(entrypoint)) {
      throw new Error(
        `Built Live entrypoint is missing at ${entrypoint}; run the Live build before this integration test`
      );
    }

    const port = await reserveEphemeralPort();
    const healthUrl = `http://127.0.0.1:${port}/live/health/`;
    let output = "";
    const child = spawn(process.execPath, [entrypoint], {
      cwd: fileURLToPath(new URL("../../", import.meta.url)),
      env: {
        ...process.env,
        NODE_OPTIONS: "",
        API_BASE_URL: "http://127.0.0.1:1",
        LIVE_SERVER_SECRET_KEY: "native-server-smoke-test-secret",
        LIVE_BASE_PATH: "/live",
        PORT: String(port),
        REDIS_URL: redisUrl,
      },
      stdio: ["ignore", "pipe", "pipe"],
    });
    child.stdout?.on("data", (chunk: Buffer) => {
      output = `${output}${chunk.toString()}`.slice(-12000);
    });
    child.stderr?.on("data", (chunk: Buffer) => {
      output = `${output}${chunk.toString()}`.slice(-12000);
    });
    const exit = observeProcessExit(child);

    try {
      const health = await waitForHealth(healthUrl, exit, () => output, 20000);
      expect(health).toMatchObject({ status: "OK" });

      const result = await stopProcess(child, exit);
      expect(result.error).toBeUndefined();
      expect(result.signal).toBeNull();
      expect(result.code).toBe(0);
    } finally {
      await stopProcess(child, exit);
    }
  }, 40000);
});
