import { AxiosError, AxiosHeaders } from "axios";
import { describe, expect, it } from "vitest";
import { AppError } from "@/lib/errors";

describe("AppError", () => {
  it("retains an existing application error and its metadata", () => {
    const original = new AppError("Conflict", { statusCode: 409, context: { operation: "save" } });
    expect(new AppError(original)).toBe(original);
    expect(original).toMatchObject({ name: "AppError", message: "Conflict", statusCode: 409 });
    expect(original.context).toEqual({ operation: "save" });
  });

  it("extracts HTTP diagnostics without retaining request credentials or the response body", () => {
    const config = {
      method: "post",
      url: "https://example.com/items",
      headers: new AxiosHeaders({ Authorization: "private-token" }),
    };
    const error = new AxiosError("Request failed", "HTTP_ERROR", config, undefined, {
      status: 503,
      statusText: "Unavailable",
      config,
      headers: new AxiosHeaders({ "Set-Cookie": "private-cookie" }),
      data: { message: "Try again", secret: "private-response" },
    });
    const result = new AppError(error);
    expect(result).toMatchObject({
      name: "AppError",
      message: "Try again",
      code: "HTTP_ERROR",
      statusCode: 503,
      method: "POST",
      url: config.url,
    });
    expect(JSON.stringify(result)).not.toMatch(/private-|Authorization|Set-Cookie/);
    expect(result).not.toHaveProperty("config");
    expect(result).not.toHaveProperty("response");
  });

  it("uses the transport message when a response supplies a non-string message", () => {
    const error = new AxiosError("Request failed");
    error.response = {
      status: 500,
      statusText: "Failure",
      headers: new AxiosHeaders(),
      config: { headers: new AxiosHeaders() },
      data: { message: { secret: "private-response" } },
    };
    expect(new AppError(error).message).toBe("Request failed");
  });

  it("normalizes cancellation, standard errors and unknown values", () => {
    expect(new AppError(new DOMException("Cancelled", "AbortError"))).toMatchObject({
      message: "Cancelled",
      code: "ABORT_ERROR",
    });
    expect(new AppError(new TypeError("Invalid input"))).toMatchObject({
      message: "Invalid input",
      code: "TypeError",
    });
    expect(new AppError(null).message).toBe("Unknown error occurred");
  });
});
