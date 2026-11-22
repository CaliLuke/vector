type DiagnosticPath = "/api/diagnostics/project-access/" | "/api/diagnostics/browser-action/";
type BrowserAction = "webhook_secret_csv_download" | "theme_config_download";

export async function postBrowserDiagnostic(path: DiagnosticPath, event: object) {
  if (typeof window === "undefined" || typeof window.fetch !== "function") return;
  try {
    await window.fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(event),
      credentials: "omit",
      mode: "same-origin",
      referrerPolicy: "no-referrer",
      keepalive: true,
    });
  } catch {
    // Diagnostic delivery must never affect the operation being reported.
  }
}

export function reportBrowserActionFailure(action: BrowserAction) {
  void postBrowserDiagnostic("/api/diagnostics/browser-action/", { action });
}
