import { useCallback, useEffect, useEffectEvent, useMemo, useState } from "react";
import { createStore, useSelector } from "@tanstack/react-store";
import { useIsMutating, useQueryClient } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { TNameDescriptionLoader } from "@plane/types";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { captureIdentityScope, isIdentityScopeCurrent, isIdentityScopeVerified } from "@/queries/identity";

/** Only mounted editor status and ownership live here. Text drafts stay in Form/editor,
 * accepted items stay in their native cache, and writes stay in the mutation cache. */
type TextOwner = { id: symbol; status: TNameDescriptionLoader; closed: boolean };
class ItemEditors {
  readonly state = createStore<{ owners: ReadonlyMap<symbol, TNameDescriptionLoader>; lease: symbol | undefined }>({
    owners: new Map(),
    lease: undefined,
  });
  get lease() {
    return this.state.state.lease;
  }
  newOwner(): TextOwner {
    return { id: Symbol("text-editor"), status: "saved", closed: false };
  }
  register(owner: TextOwner) {
    owner.closed = false;
    this.report(owner, owner.status);
  }
  unregister(owner: TextOwner) {
    owner.closed = true;
    this.state.setState((state) => {
      const owners = new Map(state.owners);
      owners.delete(owner.id);
      return { ...state, owners };
    });
  }
  report(owner: TextOwner, value: TNameDescriptionLoader) {
    if (owner.closed) return false;
    owner.status = value;
    if (this.state.state.owners.get(owner.id) !== value) {
      this.state.setState((state) => ({ ...state, owners: new Map(state.owners).set(owner.id, value) }));
    }
    return true;
  }
  acquire(owner: symbol) {
    if (this.lease || this.unsaved()) return false;
    this.state.setState((state) => ({ ...state, lease: owner }));
    return true;
  }
  release(owner: symbol) {
    if (this.lease !== owner) return;
    this.state.setState((state) => ({ ...state, lease: undefined }));
  }
  unsaved() {
    return [...this.state.state.owners.values()].some((status) => status === "submitting" || status === "error");
  }
}
const editors = new WeakMap<QueryClient, Map<string, ItemEditors>>();
function itemEditors(client: QueryClient, key: string) {
  let scopes = editors.get(client);
  if (!scopes) {
    scopes = new Map();
    editors.set(client, scopes);
  }
  let current = scopes.get(key);
  if (!current) {
    current = new ItemEditors();
    scopes.set(key, current);
  }
  return current;
}
export function useItemTextSession() {
  const client = useQueryClient();
  const account = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const identity = useMemo(() => {
    const captured = captureIdentityScope(client, account);
    return captured?.generation === generation ? captured : undefined;
  }, [client, account, generation]);
  const current = useCallback(() => isIdentityScopeCurrent(client, identity), [client, identity]);
  const verified = useCallback(() => isIdentityScopeVerified(client, identity), [client, identity]);
  return { client, key: JSON.stringify([account, generation]), current, verified };
}
function useItemEditors(scope: string) {
  const { client, key, current, verified } = useItemTextSession();
  const entry = useMemo(() => itemEditors(client, JSON.stringify([key, scope])), [client, key, scope]);
  useSelector(entry.state, (state) => state);
  return { client, entry, current, verified };
}
export function useItemTextEditor(scope: string, onStatus: (status: TNameDescriptionLoader) => void) {
  const { entry, current } = useItemEditors(scope);
  const owner = useMemo(() => entry.newOwner(), [entry]);
  const notifyInitialStatus = useEffectEvent((status: TNameDescriptionLoader) => onStatus(status));
  useEffect(() => {
    entry.register(owner);
    notifyInitialStatus(owner.status);
    return () => entry.unregister(owner);
  }, [entry, owner]);
  const report = useCallback(
    (status: TNameDescriptionLoader) => {
      if (entry.report(owner, status)) onStatus(status);
    },
    [entry, owner, onStatus]
  );
  return { report, current, locked: !!entry.lease, canEdit: () => current() && !entry.lease };
}
export function useItemEditLease(scope: string) {
  const { client, entry, verified } = useItemEditors(scope);
  const [owner] = useState(() => Symbol("item-editor"));
  const filters = {
    predicate: (mutation: { options: { scope?: { id: string } } }) =>
      mutation.options.scope?.id === `issue-text/${scope}`,
  };
  const pending = useIsMutating(filters);
  useEffect(() => () => entry.release(owner), [entry, owner]);
  const acquire = useCallback(() => {
    if (
      !verified() ||
      entry.lease ||
      entry.unsaved() ||
      client.isMutating({ predicate: (mutation) => mutation.options.scope?.id === `issue-text/${scope}` })
    )
      return false;
    return entry.acquire(owner);
  }, [verified, entry, client, owner, scope]);
  return {
    held: entry.lease === owner,
    blocked: !!entry.lease || entry.unsaved() || pending > 0,
    acquire,
    release: () => entry.release(owner),
  };
}
