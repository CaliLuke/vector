import { isAxiosError } from "axios";
import { API_BASE_URL } from "@plane/constants";
import { APIService } from "@/services/api.service";
import { authService } from "@/services/auth.service";

export type MCPLinkStatus = {
  issuer: string;
  connected: boolean;
  subject: string | null;
  pending: { id: string; subject: string; expires_at: string } | null;
};

class MCPLinkService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async status(signal?: AbortSignal): Promise<MCPLinkStatus | null> {
    try {
      const response = await this.get("/api/users/me/mcp-link/", {}, { signal });
      return response.data;
    } catch (error) {
      if (isAxiosError(error) && error.response?.status === 404) return null;
    }
    // Discard transport details before exposing a user-facing failure.
    throw new Error("Unable to load your MCP connection. Try again.");
  }

  async start(assertCurrent: () => void): Promise<string> {
    try {
      const headers = await this.csrfHeaders();
      assertCurrent();
      const response = await this.post("/api/users/me/mcp-link/start/", {}, { headers });
      const authorization = new URL(response.data.authorization_url);
      if (authorization.protocol !== "https:" || authorization.username || authorization.password)
        throw new Error("Invalid destination");
      return authorization.href;
    } catch {
      throw new Error("Unable to start the connection. Check its status and try again.");
    }
  }

  async confirm(id: string, assertCurrent: () => void): Promise<void> {
    try {
      const headers = await this.csrfHeaders();
      assertCurrent();
      await this.post("/api/users/me/mcp-link/confirm/", { id }, { headers });
    } catch {
      throw new Error("Could not confirm the connection. Reload its status before trying again.");
    }
  }

  async disconnect(assertCurrent: () => void): Promise<void> {
    try {
      const headers = await this.csrfHeaders();
      assertCurrent();
      await this.delete("/api/users/me/mcp-link/", undefined, { headers });
    } catch {
      throw new Error("Could not disconnect. Reload its status before trying again.");
    }
  }

  private async csrfHeaders() {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Refresh the page and try again.");
    return { "X-CSRFToken": csrf_token };
  }
}

export const mcpLinkService = new MCPLinkService();
