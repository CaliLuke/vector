import { useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import { currentUserOptions } from "@/queries/identity";
import { mcpLinkChangeOptions, mcpLinkOptions } from "@/queries/mcp-link";
import { mcpLinkService } from "@/services/mcp-link.service";
import type { MCPLinkStatus } from "@/services/mcp-link.service";

async function startConnection(signal: AbortSignal, assertCurrent: () => void) {
  const guard = () => {
    assertCurrent();
    signal.throwIfAborted();
  };
  const destination = await mcpLinkService.start(guard);
  guard();
  window.location.assign(destination);
}

export function MCPConnection() {
  const client = useQueryClient();
  const identity = useQuery(currentUserOptions());
  const userId = identity.isSuccess ? identity.data?.id : undefined;
  const status = useQuery(mcpLinkOptions(userId));
  const navigation = useRef<AbortController | null>(null);
  useEffect(
    () => () => {
      navigation.current?.abort();
    },
    []
  );
  const start = useMutation(mcpLinkChangeOptions(client, userId, startConnection));
  const confirm = useMutation(
    mcpLinkChangeOptions(client, userId, (id: string, guard) => mcpLinkService.confirm(id, guard))
  );
  const disconnect = useMutation(
    mcpLinkChangeOptions(client, userId, (_: void, guard) => mcpLinkService.disconnect(guard))
  );
  const busy = [start, confirm, disconnect].some((mutation) => mutation.isPending);
  const disabled = busy || status.isFetching || !status.isSuccess || !userId;
  const error = [start.error, confirm.error, disconnect.error].find(Boolean);
  const data = status.isSuccess ? status.data : undefined;
  const reload = () => {
    start.reset();
    confirm.reset();
    disconnect.reset();
    void status.refetch();
  };
  const begin = () => {
    navigation.current?.abort();
    const controller = new AbortController();
    navigation.current = controller;
    start.mutate(controller.signal);
  };
  if (data === null) return null;
  return (
    <section aria-labelledby="mcp-connection-heading" className="mt-8 border-t border-subtle py-6">
      <h2 id="mcp-connection-heading" className="text-16 font-semibold text-primary">
        MCP connection
      </h2>
      <p className="mt-2 text-13 text-secondary">
        Connect an identity to let MCP clients use your existing Vector permissions.
      </p>
      <ConnectionFeedback
        identityError={identity.isError}
        statusError={status.isError}
        loading={status.isPending}
        error={error?.message}
        disabled={busy || status.isFetching}
        reload={reload}
      />
      {data && (
        <ConnectionActions
          data={data}
          disabled={disabled}
          starting={start.isPending}
          confirming={confirm.isPending}
          disconnecting={disconnect.isPending}
          begin={begin}
          confirm={(id) => confirm.mutate(id)}
          disconnect={() => disconnect.mutate()}
        />
      )}
    </section>
  );
}

function ConnectionFeedback({
  identityError,
  statusError,
  loading,
  error,
  disabled,
  reload,
}: {
  identityError: boolean;
  statusError: boolean;
  loading: boolean;
  error?: string;
  disabled: boolean;
  reload: () => void;
}) {
  if (identityError)
    return (
      <p role="alert" className="mt-4 text-13 text-danger-primary">
        Sign in again to manage MCP access.
      </p>
    );
  const message = statusError ? "Unable to load your connection." : error;
  if (message)
    return (
      <div className="mt-4 flex items-center gap-3" role="alert">
        <p className="text-13 text-danger-primary">{message}</p>
        <Button variant="secondary" disabled={disabled} onClick={reload}>
          Reload status
        </Button>
      </div>
    );
  if (loading) return <output className="mt-4 block text-13 text-secondary">Loading connection…</output>;
  return null;
}

function ConnectionActions({
  data,
  disabled,
  starting,
  confirming,
  disconnecting,
  begin,
  confirm,
  disconnect,
}: {
  data: MCPLinkStatus;
  disabled: boolean;
  starting: boolean;
  confirming: boolean;
  disconnecting: boolean;
  begin: () => void;
  confirm: (id: string) => void;
  disconnect: () => void;
}) {
  const pending = data.pending;
  return (
    <div className="mt-4 space-y-3">
      <p className="text-13 break-all text-secondary">Identity provider: {data.issuer}</p>
      {data.connected ? (
        <>
          <p className="text-13 break-all text-primary">Connected as {data.subject}</p>
          <p className="text-13 text-secondary">
            Disconnecting stops this identity from accessing Vector. It does not revoke grants at your identity
            provider.
          </p>
          <Button variant="secondary" disabled={disabled} loading={disconnecting} onClick={disconnect}>
            Disconnect
          </Button>
        </>
      ) : pending ? (
        <>
          <p className="text-13 break-all text-primary">Connect identity {pending.subject} to your Vector account?</p>
          <p className="text-13 text-secondary">
            Continue only if this is your identity. Confirmation expires at{" "}
            {new Date(pending.expires_at).toLocaleTimeString()}.
          </p>
          <div className="flex gap-3">
            <Button variant="primary" disabled={disabled} loading={confirming} onClick={() => confirm(pending.id)}>
              Connect account
            </Button>
            <Button variant="secondary" disabled={disabled} loading={disconnecting} onClick={disconnect}>
              Cancel
            </Button>
          </div>
        </>
      ) : (
        <Button variant="primary" disabled={disabled} loading={starting} onClick={begin}>
          Continue to identity provider
        </Button>
      )}
    </div>
  );
}
