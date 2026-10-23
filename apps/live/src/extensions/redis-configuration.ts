/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AppError } from "@/lib/errors";

type RedisEnvironment = {
  url?: string;
  host?: string;
  port: number;
};

const defaultOptions = {
  lazyConnect: false,
  keepAlive: 30000,
  connectTimeout: 10000,
  maxRetriesPerRequest: 3,
  enableOfflineQueue: true,
  retryStrategy: (times: number) => Math.min(times * 50, 2000),
};

const parseQuery = (search: string) => {
  const options: Record<string, string | number> = {};
  new URLSearchParams(search).forEach((value, key) => {
    options[key] = key === "family" && !Number.isNaN(Number.parseInt(value, 10)) ? Number.parseInt(value, 10) : value;
  });
  return options;
};

const parseUnixSocketUrl = (url: string) => {
  const queryIndex = url.indexOf("?");
  const path = queryIndex === -1 ? url : url.slice(0, queryIndex);
  const query = queryIndex === -1 ? "" : url.slice(queryIndex + 1);
  return { host: "localhost", port: 6379, options: { ...defaultOptions, ...parseQuery(query), path } };
};

const parseNetworkUrl = (url: string) => {
  const hasScheme = /^rediss?:\/\//i.test(url);
  const isProtocolRelative = url.startsWith("//");
  const normalizedUrl = hasScheme ? url : isProtocolRelative ? `redis:${url}` : `redis://${url}`;
  const parsedUrl = new URL(normalizedUrl);
  if (parsedUrl.protocol !== "redis:" && parsedUrl.protocol !== "rediss:") {
    throw new AppError("Redis URL must use redis:// or rediss://");
  }

  const queryOptions = parseQuery(parsedUrl.search);
  const urlOptions = {
    ...(parsedUrl.username || parsedUrl.password
      ? {
          username: decodeURIComponent(parsedUrl.username),
          password: decodeURIComponent(parsedUrl.password),
        }
      : {}),
    ...getPathOption(parsedUrl.pathname, hasScheme || isProtocolRelative),
  };
  const options = {
    ...defaultOptions,
    ...queryOptions,
    ...urlOptions,
    ...(parsedUrl.protocol === "rediss:" && !("tls" in queryOptions) ? { tls: true } : {}),
  };

  return {
    host: parsedUrl.hostname.replace(/^\[|\]$/g, "") || getQueryHost(queryOptions),
    port: getPort(parsedUrl.port, queryOptions.port),
    options,
  };
};

const getPathOption = (pathname: string, isDatabaseUrl: boolean) => {
  if (pathname.length <= 1) return {};
  return isDatabaseUrl ? { db: pathname.slice(1) } : { path: pathname };
};

const getQueryHost = (options: Record<string, string | number>) => {
  return typeof options.host === "string" ? options.host : "localhost";
};

const getPort = (urlPort: string, queryPort: string | number | undefined) => {
  if (urlPort) return Number(urlPort);
  if (queryPort !== undefined) return Number(queryPort);
  return 6379;
};

const parseRedisUrl = (url: string) => {
  return url.startsWith("/") && !url.startsWith("//") ? parseUnixSocketUrl(url) : parseNetworkUrl(url);
};

/** Build Hocuspocus Redis options with the same URL precedence as ioredis. */
export const getRedisConfiguration = ({ url, host, port }: RedisEnvironment) => {
  if (url) return parseRedisUrl(url);
  if (!host) throw new AppError("Redis client not initialized");
  return { host, port, options: defaultOptions };
};
