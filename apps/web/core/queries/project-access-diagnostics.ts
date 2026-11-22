import { isAxiosError } from "axios";
import { postBrowserDiagnostic } from "@/helpers/browser-diagnostics";

type AccessRead =
  | "identity"
  | "workspace_membership"
  | "project_roles"
  | "project_catalogue"
  | "project_detail"
  | "project_probe"
  | "project_preferences";
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

const retryableStatuses = new Set([502, 503, 504]);

function isTransientFailure(failure: ReturnType<typeof accessFailure>) {
  return failure.reason === "network" || retryableStatuses.has(failure.http_status ?? 0);
}

export function isTransientProjectAccessError(error: unknown) {
  return isTransientFailure(accessFailure(error));
}

export function isDefinitiveProjectAccessDenial(error: unknown) {
  const failure = accessFailure(error);
  return failure.reason === "response_mismatch" || [401, 403, 404, 409, 410].includes(failure.http_status ?? 0);
}

export function projectAccessReadRetry(stage: AccessRead) {
  return {
    retryDelay: (attempt: number) => 500 * 2 ** attempt,
    retry(failureCount: number, error: unknown) {
      const failure = accessFailure(error);
      const willRetry = failureCount < 3 && isTransientFailure(failure);
      const projectRead = stage === "project_detail" || stage === "project_probe";
      const expectedDenial =
        failure.http_status === 401 || (projectRead && [403, 404, 409].includes(failure.http_status ?? 0));
      if (!expectedDenial && failure.reason !== "canceled") {
        void postBrowserDiagnostic("/api/diagnostics/project-access/", { stage, ...failure, will_retry: willRetry });
      }
      return willRetry;
    },
  };
}
