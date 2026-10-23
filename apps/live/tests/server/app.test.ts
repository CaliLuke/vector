import type { Request, Response } from "express";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { startLiveTestServer } from "../helpers/http-server";

describe("Live Express application", () => {
  let closeServer: () => Promise<void>;
  let baseUrl: string;

  beforeAll(async () => {
    const server = await startLiveTestServer(({ app }) => {
      app.get("/__security_headers", (_request, response) => response.sendStatus(204));
      app.post("/__body", (request: Request, response: Response) => response.json(request.body));
    }, "http://test-origin");
    closeServer = server.close;
    baseUrl = server.baseUrl;
  });

  afterAll(async () => {
    await closeServer?.();
  });

  it("sends Helmet's one-year HSTS policy and configured CORS headers", async () => {
    const response = await fetch(`${baseUrl}/__security_headers`, {
      headers: { Origin: "http://test-origin" },
    });

    expect(response.status).toBe(204);
    expect(response.headers.get("strict-transport-security")).toBe("max-age=31536000; includeSubDomains");
    expect(response.headers.get("access-control-allow-origin")).toBe("http://test-origin");
  });

  it("parses JSON and extended URL-encoded request bodies", async () => {
    const jsonResponse = await fetch(`${baseUrl}/__body`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ page: { title: "JSON" } }),
    });
    expect(jsonResponse.status).toBe(200);
    expect(await jsonResponse.json()).toEqual({ page: { title: "JSON" } });

    const formResponse = await fetch(`${baseUrl}/__body`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: "page%5Btitle%5D=Form",
    });
    expect(formResponse.status).toBe(200);
    expect(await formResponse.json()).toEqual({ page: { title: "Form" } });
  });
});
