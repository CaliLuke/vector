import { registerController } from "@plane/decorators";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { DocumentController } from "@/controllers/document.controller";
import { startLiveTestServer } from "../helpers/http-server";

describe("document conversion validation", () => {
  let closeServer: () => Promise<void>;
  let baseUrl: string;

  beforeAll(async () => {
    const server = await startLiveTestServer(({ router }) => {
      registerController(router, DocumentController);
    });
    baseUrl = server.baseUrl;
    closeServer = server.close;
  });

  afterAll(async () => closeServer());

  it("returns issue paths and messages for invalid input", async () => {
    const response = await fetch(`${baseUrl}/convert-document/`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ description_html: "<p>Valid markup</p>", variant: "unknown" }),
    });

    expect(response.status).toBe(400);
    expect(await response.json()).toMatchObject({
      message: "Validation error",
      context: {
        validationErrors: [expect.objectContaining({ path: "variant", message: expect.any(String) })],
      },
    });
  });
});
