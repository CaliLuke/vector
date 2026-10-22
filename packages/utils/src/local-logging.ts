const clean = (text: string) =>
  text
    .replace(/(https?:\/\/[^\s?#]+)[?#][^\s]*/g, "$1")
    .replace(
      /((?:proxy-)?authorization|(?:set-)?cookie|password|token|secret)([\w-]*["']?\s*[:=]\s*)(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\r\n]*)/gi,
      "$1$2[redacted]"
    )
    .slice(0, 8000);

// Do not serialize arbitrary console objects: Axios errors can contain auth headers and bodies.
const consoleMessage = (args: unknown[]) =>
  args
    .map((arg) =>
      arg instanceof Error ? arg.stack || arg.message : typeof arg === "string" ? arg : "[object omitted]"
    )
    .join(" ");

const truncateJsonString = (value: string, maxBytes: number) => {
  const characters = Array.from(value);
  const encodedLength = (length: number) =>
    new TextEncoder().encode(JSON.stringify(characters.slice(0, length).join(""))).byteLength;

  let low = 0;
  let high = characters.length;
  while (low < high) {
    const middle = Math.ceil((low + high) / 2);
    if (encodedLength(middle) <= maxBytes) low = middle;
    else high = middle - 1;
  }
  return characters.slice(0, low).join("");
};

/** Local development diagnostics, sent only to the local proxy's OTLP receiver. */
export function installLocalLogging(service: string): () => void {
  if (!["localhost", "127.0.0.1"].includes(window.location.hostname)) return () => {};

  const endpoint = "/__local_telemetry/v1/logs";
  const originalFetch = window.fetch;
  const originalSend = XMLHttpRequest.prototype.send;
  const originalError = console.error;
  const originalWarn = console.warn;
  const queue: object[] = [];
  let sending = false;

  const report = (message: string, attributes: Record<string, string | number> = {}, level = "ERROR") => {
    if (queue.length >= 100) return;
    const record = {
      timeUnixNano: String(BigInt(Date.now()) * 1000000n),
      severityNumber: level === "WARN" ? 13 : 17,
      severityText: level,
      body: { stringValue: truncateJsonString(clean(message), 12000) },
      attributes: Object.entries({ "url.path": window.location.pathname, ...attributes })
        .slice(0, 8)
        .map(([key, value]) => ({
          key,
          value:
            typeof value === "number"
              ? { intValue: String(value) }
              : {
                  stringValue: truncateJsonString(clean(value), key === "code.stacktrace" ? 8000 : 1000),
                },
        })),
    };
    queue.push(record);
  };

  const flush = async () => {
    if (sending || !queue.length) return;
    sending = true;
    // Keep each export below the browser's 64 KiB keepalive body budget.
    const batch: object[] = [];
    let bytes = 0;
    while (queue.length && batch.length < 20) {
      const size = new TextEncoder().encode(JSON.stringify(queue[0])).length;
      if (batch.length && bytes + size > 48000) break;
      bytes += size;
      batch.push(queue.shift()!);
    }
    try {
      const response = await originalFetch.call(window, endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "omit",
        keepalive: true,
        signal: AbortSignal.timeout(3000),
        body: JSON.stringify({
          resourceLogs: [
            {
              resource: {
                attributes: [{ key: "service.name", value: { stringValue: truncateJsonString(service, 1000) } }],
              },
              scopeLogs: [{ scope: { name: "plane.local.browser" }, logRecords: batch }],
            },
          ],
        }),
      });
      if (!response.ok) queue.unshift(...batch.slice(0, 100 - queue.length));
    } catch {
      queue.unshift(...batch.slice(0, 100 - queue.length));
    } finally {
      sending = false;
    }
  };

  const failedRequest = (url: string, status: number) => {
    const parsed = new URL(url, window.location.href);
    if (parsed.origin !== window.location.origin || parsed.pathname === endpoint) return;
    if (status === 0 || status >= 400) {
      report("HTTP request failed", { "http.response.status_code": status, "http.request.path": parsed.pathname });
    }
  };
  window.fetch = async (...args: Parameters<typeof window.fetch>) => {
    const input = args[0];
    const url = input instanceof Request ? input.url : String(input);
    try {
      const response = await originalFetch.apply(window, args);
      failedRequest(url, response.status);
      return response;
    } catch (error) {
      if (!(error instanceof DOMException && error.name === "AbortError")) failedRequest(url, 0);
      throw error;
    }
  };
  // oxlint-disable-next-line no-extend-native -- Development instrumentation covers all Axios instances and is restored on disposal.
  XMLHttpRequest.prototype.send = function (body) {
    let aborted = false;
    const markAborted = () => {
      aborted = true;
    };
    const onLoadEnd = () => {
      if (!aborted) failedRequest(this.responseURL, this.status);
      this.removeEventListener("abort", markAborted);
    };
    this.addEventListener("abort", markAborted, { once: true });
    this.addEventListener("loadend", onLoadEnd, { once: true });
    originalSend.call(this, body);
  };

  console.error = (...args: unknown[]) => {
    originalError.apply(console, args);
    report(consoleMessage(args), { "code.stacktrace": new Error("Console error call site").stack || "" });
  };
  console.warn = (...args: unknown[]) => {
    originalWarn.apply(console, args);
    report(consoleMessage(args), { "code.stacktrace": new Error("Console warning call site").stack || "" }, "WARN");
  };
  const onError = (event: ErrorEvent) =>
    report(event.error instanceof Error ? event.error.stack || event.message : event.message);
  const onRejection = (event: PromiseRejectionEvent) => report(consoleMessage([event.reason]));
  const onPageHide = () => {
    void flush();
  };
  window.addEventListener("error", onError);
  window.addEventListener("unhandledrejection", onRejection);
  window.addEventListener("pagehide", onPageHide);
  const timer = window.setInterval(() => {
    void flush();
  }, 1000);

  return () => {
    window.clearInterval(timer);
    window.removeEventListener("error", onError);
    window.removeEventListener("unhandledrejection", onRejection);
    window.removeEventListener("pagehide", onPageHide);
    window.fetch = originalFetch;
    // oxlint-disable-next-line no-extend-native -- Restore the original development instrumentation hook.
    XMLHttpRequest.prototype.send = originalSend;
    console.error = originalError;
    console.warn = originalWarn;
    void flush();
  };
}
