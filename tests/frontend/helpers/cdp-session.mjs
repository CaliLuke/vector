// CDP failures invalidate the whole session: neither command promises nor event
// waiters may survive a broken transport and make a browser check appear hung.
export class CdpSession {
  constructor(webSocketUrl) {
    this.webSocketUrl = webSocketUrl;
    this.nextId = 1;
    this.pending = new Map();
    this.listeners = new Map();
    this.waiters = new Set();
  }

  async connect() {
    this.socket = new WebSocket(this.webSocketUrl);
    this.socket.addEventListener("message", (event) => {
      void this.onMessage(event).catch((error) => this.fail(error));
    });
    this.socket.addEventListener("close", () => this.fail(new Error("DevTools connection closed")));
    this.socket.addEventListener("error", () => this.fail(new Error("DevTools connection failed")));
    await new Promise((resolve, reject) => {
      const failed = (error) => {
        this.socket.removeEventListener("open", opened);
        reject(error);
      };
      const opened = () => {
        this.waiters.delete(failed);
        resolve();
      };
      this.waiters.add(failed);
      this.socket.addEventListener("open", opened, { once: true });
    });
  }

  on(method, listener) {
    const listeners = this.listeners.get(method) ?? new Set();
    listeners.add(listener);
    this.listeners.set(method, listeners);
    return () => {
      listeners.delete(listener);
      if (!listeners.size) this.listeners.delete(method);
    };
  }

  async onMessage(event) {
    if (this.failure) return;
    const data = typeof event.data === "string" ? event.data : await event.data.text();
    const message = JSON.parse(data);
    if (!message || typeof message !== "object") throw new Error("Invalid DevTools message");
    if (message.id !== undefined) {
      const pending = this.pending.get(message.id);
      if (!pending) return;
      this.pending.delete(message.id);
      if (message.error) pending.reject(new Error(`${message.error.message} (${message.error.code})`));
      else pending.resolve(message.result);
      return;
    }
    await Promise.all(Array.from(this.listeners.get(message.method) ?? [], (listener) => listener(message.params)));
  }

  send(method, params = {}) {
    if (this.failure) return Promise.reject(this.failure);
    const id = this.nextId++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      try {
        this.socket.send(JSON.stringify({ id, method, params }));
      } catch (error) {
        this.fail(error);
      }
    });
  }

  waitFor(method, timeoutMilliseconds) {
    if (this.failure) return Promise.reject(this.failure);
    return new Promise((resolve, reject) => {
      const cleanup = () => {
        clearTimeout(timeout);
        off();
        this.waiters.delete(failed);
      };
      const failed = (error) => {
        cleanup();
        reject(error);
      };
      const off = this.on(method, (params) => {
        cleanup();
        resolve(params);
      });
      const timeout = setTimeout(() => failed(new Error(`timed out waiting for ${method}`)), timeoutMilliseconds);
      this.waiters.add(failed);
    });
  }

  fail(error) {
    if (this.failure) return;
    this.failure = error;
    for (const pending of this.pending.values()) pending.reject(error);
    this.pending.clear();
    for (const reject of this.waiters) reject(error);
    this.waiters.clear();
    this.listeners.clear();
    this.socket?.close();
  }

  close() {
    this.fail(new Error("DevTools session closed"));
  }
}
