import { describe, expect, it } from "vitest";
import { getRedisConfiguration } from "@/extensions/redis-configuration";

describe("Hocuspocus Redis connection options", () => {
  it("maps Redis URL credentials, database, TLS, and retry settings", () => {
    const config = getRedisConfiguration({
      url: "rediss://user%20name:p%40ss@redis.example.test:6380/4",
      port: 6379,
    });

    expect(config.host).toBe("redis.example.test");
    expect(config.port).toBe(6380);
    expect(config.options).toMatchObject({
      username: "user name",
      password: "p@ss",
      db: "4",
      tls: true,
      lazyConnect: false,
      keepAlive: 30000,
      connectTimeout: 10000,
      maxRetriesPerRequest: 3,
      enableOfflineQueue: true,
    });
    expect(config.options.retryStrategy(2)).toBe(100);
  });

  it("uses URL defaults and query options without applying the environment port", () => {
    const config = getRedisConfiguration({
      url: "redis://redis.example.test/?family=6&connectTimeout=2500&port=6382",
      port: 6399,
    });

    expect(config.host).toBe("redis.example.test");
    expect(config.port).toBe(6382);
    expect(config.options).toMatchObject({ family: 6, connectTimeout: "2500" });
  });

  it("uses localhost for an empty URL authority and preserves query overrides", () => {
    const config = getRedisConfiguration({ url: "rediss:///?tls=preferred", port: 6388 });

    expect(config.host).toBe("localhost");
    expect(config.port).toBe(6379);
    expect(config.options).toMatchObject({ tls: "preferred" });
  });

  it("uses query host and port when the URL authority omits them", () => {
    const config = getRedisConfiguration({ url: "redis:///?host=cache.internal&port=6383", port: 6399 });

    expect(config.host).toBe("cache.internal");
    expect(config.port).toBe(6383);
  });

  it("maps a raw socket URL without query options", () => {
    const config = getRedisConfiguration({ url: "/var/run/redis.sock", port: 6399 });

    expect(config.options).toMatchObject({ path: "/var/run/redis.sock" });
  });

  it("supports protocol-relative and unqualified socket URLs", () => {
    const protocolRelative = getRedisConfiguration({ url: "//cache.internal:6384", port: 6399 });
    const unqualified = getRedisConfiguration({ url: "cache.internal:6385/socket/redis", port: 6399 });

    expect(protocolRelative).toMatchObject({ host: "cache.internal", port: 6384 });
    expect(unqualified).toMatchObject({ host: "cache.internal", port: 6385, options: { path: "/socket/redis" } });
  });

  it("maps Unix socket paths and their query options", () => {
    const config = getRedisConfiguration({ url: "/var/run/redis.sock?db=2", port: 6399 });

    expect(config.host).toBe("localhost");
    expect(config.port).toBe(6379);
    expect(config.options).toMatchObject({ path: "/var/run/redis.sock", db: "2" });
  });

  it("uses host and port settings when no Redis URL is configured", () => {
    expect(getRedisConfiguration({ host: "redis.internal", port: 6381 })).toMatchObject({
      host: "redis.internal",
      port: 6381,
      options: { lazyConnect: false, enableOfflineQueue: true },
    });
  });

  it("rejects an unset Redis connection target", () => {
    expect(() => getRedisConfiguration({ port: 6379 })).toThrow("Redis client not initialized");
  });
});
