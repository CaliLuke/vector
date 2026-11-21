import { isAxiosError } from "axios";

type AccessRead = "workspace_membership" | "project_roles" | "project_detail" | "project_probe";
type AccessFailure = "scope_changed" | "response_mismatch";

export class ProjectAccessReadError extends Error {
  constructor(
    readonly reason: AccessFailure,
    message: string
  ) {
    super(message);
  }
}

function accessFailure(error: unknown) {
  if (error instanceof ProjectAccessReadError) return { reason: error.reason };

  const response = isAxiosError(error) ? error.response : error;
  if (response && typeof response === "object" && "status" in response && typeof response.status === "number") {
    const requestId = "headers" in response ? readRequestId(response.headers) : undefined;
    return { reason: "http", http_status: response.status, request_id: requestId };
  }

  if (isAxiosError(error)) {
    if (error.code === "ERR_CANCELED") return { reason: "canceled" };
    if (error.code === "ERR_NETWORK" || error.code === "ECONNABORTED" || error.code === "ETIMEDOUT")
      return { reason: "network" };
  }
  return { reason: "unexpected" };
}

function readRequestId(headers: unknown) {
  if (!headers || typeof headers !== "object" || !("x-request-id" in headers)) return undefined;
  const id = headers["x-request-id"];
  if (typeof id === "string" && /^[\da-f]{8}-(?:[\da-f]{4}-){3}[\da-f]{12}$/i.test(id)) return id;
  return undefined;
}

export function projectAccessReadRetry(stage: AccessRead) {
  return {
    retryDelay: 500,
    retry(failureCount: number, error: unknown) {
      const failure = accessFailure(error);
      const willRetry = failureCount === 0 && failure.reason === "network";
      const projectRead = stage === "project_detail" || stage === "project_probe";
      const expectedDenial =
        failure.http_status === 401 || (projectRead && [403, 404, 409].includes(failure.http_status ?? 0));
      if (
        typeof window !== "undefined" &&
        typeof window.fetch === "function" &&
        !expectedDenial &&
        failure.reason !== "canceled"
      ) {
        void window
          .fetch("/api/diagnostics/project-access/", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ stage, ...failure, will_retry: willRetry }),
            credentials: "omit",
            keepalive: true,
          })
          .catch(() => {});
      }
      return willRetry;
    },
  };
}
