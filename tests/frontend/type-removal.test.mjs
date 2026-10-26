import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
const { useTypeRemoval } = await import("../../apps/web/core/components/work-item-types/use-type-removal.ts");
const { identityKeys, advanceIdentitySession, currentUserOptions } =
  await import("../../apps/web/core/queries/identity.ts");
const { default: userService } = await import("../../apps/web/core/services/user.service.ts");

async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(identityKeys.currentUser(), { id: "owner" });
  const commits = [];
  let hook;
  const operations = {
    preview: async (_slug, _command, beforeSend) => {
      beforeSend();
      return request().impact;
    },
    commit: async (_slug, command, beforeSend) => {
      beforeSend();
      commits.push(structuredClone(command));
    },
  };
  function Subject({ workspaceSlug, typeId }) {
    const result = useTypeRemoval(workspaceSlug, typeId, operations);
    React.useLayoutEffect(() => {
      hook = result;
    });
    return React.createElement("output", {}, result.retained ? "locked" : "editable");
  }
  const render = (show, workspaceSlug = "acme", typeId = "bug") =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        show ? React.createElement(Subject, { workspaceSlug, typeId }) : null
      )
    );
  t.after(async () => {
    await env.unmount();
    client.clear();
  });
  await render(true);
  return {
    env,
    client,
    operations,
    commits,
    render,
    get hook() {
      return hook;
    },
  };
}
function request() {
  return {
    command: { type_id: "bug", replacement_id: "task", revision: 3, review_id: "receipt" },
    impact: { id: "receipt", items: 3, drafts: 2, projects: 1, preserved: 4, invalid_items: 0, blocked_reason: "" },
  };
}

await test("lost deletion response retains exact review through reopen and access denial", async (t) => {
  const f = await fixture(t);
  let attempt = 0;
  f.operations.commit = async (_slug, command, beforeSend) => {
    beforeSend();
    f.commits.push(structuredClone(command));
    if (++attempt === 1) throw new Error("Response lost");
    if (attempt === 2) throw { response: { status: 403 } };
  };
  const original = request();
  await act(async () => {
    await assert.rejects(f.hook.commit(original));
  });
  original.command.replacement_id = "story";
  await f.render(false);
  await f.render(true);
  assert.equal(f.env.container.textContent, "locked");
  assert.equal(f.hook.retained.impact.preserved, 4);
  await act(async () => {
    await assert.rejects(f.hook.commit(original));
  });
  assert.equal(f.hook.retained.command.replacement_id, "task");
  await act(async () => {
    await f.hook.commit(original);
  });
  await f.render(false);
  await f.render(true);
  await act(async () => {
    await f.hook.commit(original);
  });
  assert.deepEqual(
    f.commits.map((command) => command.replacement_id),
    ["task", "task", "task"]
  );
  await act(async () => {
    f.hook.acknowledge();
  });
  assert.equal(f.hook.retained, undefined);
});

await test("confirmed uncommitted rejection permits a new reviewed replacement", async (t) => {
  const f = await fixture(t);
  f.operations.commit = async () => {
    throw { response: { status: 409, data: { review_again: true } } };
  };
  await act(async () => {
    await assert.rejects(f.hook.commit(request()));
  });
  assert.equal(f.hook.retained, undefined);
  f.operations.commit = async (_slug, command) => {
    f.commits.push(command);
  };
  const repaired = request();
  repaired.command.review_id = "new-receipt";
  repaired.command.replacement_id = "story";
  await act(async () => {
    await f.hook.commit(repaired);
  });
  assert.equal(f.commits[0].review_id, "new-receipt");
  assert.equal(f.commits[0].replacement_id, "story");
});

await test("unresolved deletion does not cross workspace or session boundaries", async (t) => {
  const f = await fixture(t);
  f.operations.commit = async () => {
    throw new Error("Response lost");
  };
  await act(async () => {
    await assert.rejects(f.hook.commit(request()));
  });
  await f.render(true, "other");
  assert.equal(f.hook.retained, undefined);
  await f.render(true);
  assert.equal(f.hook.retained.command.review_id, "receipt");
  await act(async () => {
    advanceIdentitySession(f.client);
  });
  assert.equal(f.hook.retained, undefined);
});

await test("closing during session-token acquisition prevents deletion", async (t) => {
  const f = await fixture(t);
  let release;
  const token = new Promise((resolve) => {
    release = resolve;
  });
  f.operations.commit = async (_slug, command, beforeSend) => {
    await token;
    beforeSend();
    f.commits.push(command);
  };
  let pending;
  await act(async () => {
    pending = f.hook.commit(request());
  });
  const rejected = assert.rejects(pending, /closed|session/);
  await f.render(false);
  await act(async () => {
    release();
    await rejected;
  });
  assert.equal(f.commits.length, 0);
});

await test("preview result is discarded when identity verification fails in flight", async (t) => {
  const f = await fixture(t);
  const pending = Promise.withResolvers();
  const started = Promise.withResolvers();
  f.operations.preview = async (_slug, _command, beforeSend) => {
    beforeSend();
    started.resolve();
    return pending.promise;
  };
  let result;
  await act(async () => {
    result = f.hook.preview(request().command);
  });
  await started.promise;
  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(f.client.fetchQuery({ ...currentUserOptions(), staleTime: 0 }));
  pending.resolve(request().impact);
  await act(async () => {
    await assert.rejects(result, /unverified/);
  });
  assert.equal(f.commits.length, 0);
});

await test("commit rechecks verified identity after token acquisition before transport", async (t) => {
  const f = await fixture(t);
  const token = Promise.withResolvers();
  const tokenStarted = Promise.withResolvers();
  f.operations.commit = async (_slug, command, beforeSend) => {
    tokenStarted.resolve();
    await token.promise;
    beforeSend();
    f.commits.push(structuredClone(command));
  };
  let result;
  await act(async () => {
    result = f.hook.commit(request());
  });
  await tokenStarted.promise;
  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(f.client.fetchQuery({ ...currentUserOptions(), staleTime: 0 }));
  token.resolve();
  await act(async () => {
    await assert.rejects(result, /unverified/);
  });
  assert.equal(f.commits.length, 0);
});

await test("late accepted deletion cannot invalidate a replacement session query", async (t) => {
  const f = await fixture(t);
  const key = ["workspace", "acme", "work-item-types"];
  f.client.setQueryData(key, { account: "owner" });
  let release;
  const response = new Promise((resolve) => {
    release = resolve;
  });
  f.operations.commit = async () => {
    await response;
  };
  let pending;
  await act(async () => {
    pending = f.hook.commit(request());
  });
  const rejected = assert.rejects(pending, /closed|session/);
  await act(async () => {
    advanceIdentitySession(f.client);
    f.client.setQueryData(identityKeys.currentUser(), { id: "other" });
    f.client.removeQueries({ queryKey: key });
    f.client.setQueryData(key, { account: "other" });
  });
  await act(async () => {
    release();
    await rejected;
  });
  assert.deepEqual(f.client.getQueryData(key), { account: "other" });
  assert.equal(f.client.getQueryState(key).isInvalidated, false);
});

await test("successful migration invalidates captured item and draft caches without touching other scopes", async (t) => {
  const f = await fixture(t);
  const peek = ["peek-issue", "acme", "project", "item"];
  const drafts = ["workspace-drafts", "owner", 0, "acme", 20, "membership-v1", "list"];
  const otherPeek = ["peek-issue", "other", "project", "item"];
  const otherDrafts = ["workspace-drafts", "other-owner", 0, "acme", 20, "membership-v1", "list"];
  f.client.setQueryData(peek, { type_id: "bug" });
  f.client.setQueryData(drafts, [{ type_id: "bug" }]);
  f.client.setQueryData(otherPeek, { type_id: "bug" });
  f.client.setQueryData(otherDrafts, [{ type_id: "bug" }]);
  await act(async () => {
    await f.hook.commit(request());
  });
  assert.equal(f.client.getQueryState(peek).isInvalidated, true);
  assert.equal(f.client.getQueryState(drafts).isInvalidated, true);
  assert.equal(f.client.getQueryState(otherPeek).isInvalidated, false);
  assert.equal(f.client.getQueryState(otherDrafts).isInvalidated, false);
});

await test("fresh preview clears safely rejected error after reopening", async (t) => {
  const f = await fixture(t);
  f.operations.commit = async () => {
    throw { response: { status: 409, data: { review_again: true } } };
  };
  await act(async () => {
    await assert.rejects(f.hook.commit(request()));
  });
  await f.render(false);
  await f.render(true);
  assert.equal(f.hook.rejected.command.replacement_id, "task");
  assert.ok(f.hook.error);
  await act(async () => {
    await f.hook.preview(request().command);
  });
  assert.equal(f.hook.error, undefined);
  assert.equal(f.hook.rejected, undefined);
});

await test("a new preview cannot replace an uncertain deletion", async (t) => {
  const f = await fixture(t);
  f.operations.commit = async () => {
    throw new Error("Response lost");
  };
  await act(async () => {
    await assert.rejects(f.hook.commit(request()));
  });
  await act(async () => {
    await assert.rejects(f.hook.preview(request().command), /original deletion/);
  });
  assert.equal(f.hook.retained.command.review_id, "receipt");
});

await test("a retained dialog cannot commit a different type's review", async (t) => {
  const f = await fixture(t);
  await f.render(true, "acme", "story");
  await act(async () => {
    await assert.rejects(f.hook.commit(request()), /selected type changed/);
  });
  assert.equal(f.commits.length, 0);
});
