import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
const { useReviewedItemEdit } =
  await import("../../apps/web/core/components/issues/issue-modal/use-reviewed-item-edit.ts");
const { identityKeys, advanceIdentitySession, currentUserOptions } =
  await import("../../apps/web/core/queries/identity.ts");
const { default: userService } = await import("../../apps/web/core/services/user.service.ts");

async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(identityKeys.currentUser(), { id: "owner" });
  const commits = [],
    previews = [];
  let hook;
  const operations = {
    preview: async (request) => {
      previews.push(structuredClone(request));
      return { id: "review", preserved_properties: 0, description_html: "" };
    },
    commit: async (payload) => {
      commits.push(structuredClone(payload));
      return { id: "published", project_id: "project" };
    },
  };
  function Subject() {
    const save = useReviewedItemEdit("acme", "draft", operations);
    React.useLayoutEffect(() => {
      hook = save;
    });
    return React.createElement("output", {}, save.recoveryCommand ? "locked" : "editable");
  }
  const render = (show) =>
    env.render(React.createElement(QueryClientProvider, { client }, show ? React.createElement(Subject) : null));
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  await render(true);
  return {
    env,
    client,
    operations,
    commits,
    previews,
    render,
    get hook() {
      return hook;
    },
  };
}
const request = () => ({
  changes: { name: "Original", project_id: "project", custom_properties: { rich: "<p>Retained</p>" } },
  mode: "publish",
  project_id: "project",
  item_id: "draft",
  revision: 3,
});

async function submitWithFailedIdentityRefresh(t, f, started) {
  let submission;
  await act(async () => {
    submission = f.hook.submit(request());
  });
  await started;
  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(f.client.fetchQuery({ ...currentUserOptions(), staleTime: 0 }));
  return { submission };
}

await test("uncertain save survives reopen and recovery denial keeps the original payload", async (t) => {
  const f = await fixture(t);
  let attempt = 0;
  f.operations.commit = async (payload) => {
    f.commits.push(structuredClone(payload));
    if (++attempt === 1) throw new Error("Response lost");
    if (attempt === 2) throw { response: { status: 403 } };
    return { id: "published" };
  };
  const original = request();
  await act(async () => {
    await assert.rejects(f.hook.submit(original));
  });
  assert.equal(!!f.hook.recoveryCommand, true);
  original.changes.name = "Changed later";
  await f.render(false);
  await f.render(true);
  assert.equal(f.env.container.textContent, "locked");
  await act(async () => {
    await assert.rejects(f.hook.submit(request()));
  });
  assert.equal(!!f.hook.recoveryCommand, true);
  await act(async () => {
    assert.equal((await f.hook.submit(original)).id, "published");
  });
  assert.equal(f.previews.length, 1);
  assert.deepEqual(
    f.commits.map((p) => p.changes.name),
    ["Original", "Original", "Original"]
  );
  // Recovery replays the same receipt to refresh authorization and the saved projection.
  await f.render(false);
  await f.render(true);
  await act(async () => {
    assert.equal((await f.hook.submit(original)).id, "published");
  });
  assert.equal(f.commits.length, 4);
  assert.deepEqual(f.commits[3], f.commits[2]);
});

await test("preview result is discarded when identity verification fails in flight", async (t) => {
  const f = await fixture(t);
  const preview = Promise.withResolvers();
  const started = Promise.withResolvers();
  f.operations.preview = async () => {
    started.resolve();
    return preview.promise;
  };
  const { submission } = await submitWithFailedIdentityRefresh(t, f, started.promise);
  preview.resolve({ id: "review", preserved_properties: 1, description_html: "<p>Private</p>" });
  await act(async () => {
    await assert.rejects(submission, /unverified/);
  });
  assert.equal(f.commits.length, 0);
});

await test("commit rechecks verified identity at the transport boundary", async (t) => {
  const f = await fixture(t);
  const token = Promise.withResolvers();
  const started = Promise.withResolvers();
  f.operations.commit = async (payload, beforeSend) => {
    started.resolve();
    await token.promise;
    beforeSend?.();
    f.commits.push(structuredClone(payload));
    return { id: "published" };
  };
  const { submission } = await submitWithFailedIdentityRefresh(t, f, started.promise);
  token.resolve();
  await act(async () => {
    await assert.rejects(submission, /unverified/);
  });
  assert.equal(f.commits.length, 0);
});

await test("initial validation failure permits a repaired request and preview retry", async (t) => {
  const f = await fixture(t);
  let first = true;
  f.operations.commit = async (payload) => {
    f.commits.push(structuredClone(payload));
    if (first) {
      first = false;
      throw { response: { status: 400, data: { name: "Required" } } };
    }
    return { id: "published" };
  };
  await act(async () => {
    await assert.rejects(f.hook.submit(request()));
  });
  assert.equal(!!f.hook.recoveryCommand, false);
  const repaired = request();
  repaired.changes.name = "Repaired";
  await act(async () => {
    await f.hook.submit(repaired);
  });
  assert.equal(f.previews.length, 2);
  assert.equal(f.commits[1].changes.name, "Repaired");
});

await test("concurrent submission posts once and account change during previews prevents save", async (t) => {
  const f = await fixture(t);
  const ready = Promise.withResolvers();
  f.operations.preview = () => ready.promise;
  let pending;
  await act(async () => {
    pending = f.hook.submit(request());
  });
  await act(async () => {
    await assert.rejects(f.hook.submit(request()), /already in progress/);
  });
  const failure = assert.rejects(pending, /account session has changed/);
  await act(async () => {
    advanceIdentitySession(f.client);
    f.client.clear();
    f.client.setQueryData(identityKeys.currentUser(), { id: "other" });
    ready.resolve({ id: "review", preserved_properties: 0, description_html: "" });
    await failure;
  });
  assert.equal(f.commits.length, 0);
  assert.equal(!!f.hook.recoveryCommand, false);
});

await test("preview transport failure permits retry without claiming a save was attempted", async (t) => {
  const f = await fixture(t);
  f.operations.preview = async () => {
    throw new Error("Upload failed");
  };
  await act(async () => {
    await assert.rejects(f.hook.submit(request()), /Upload failed/);
  });
  assert.equal(!!f.hook.recoveryCommand, false);
  assert.equal(f.commits.length, 0);
});

await test("reopening an initial pending save waits on the existing attempt", async (t) => {
  const f = await fixture(t);
  const response = Promise.withResolvers();
  f.operations.commit = async (payload) => {
    f.commits.push(payload);
    return response.promise;
  };
  let pending;
  await act(async () => {
    pending = f.hook.submit(request());
  });
  assert.equal(f.hook.isPending, true);
  assert.equal(f.hook.recoveryCommand, undefined, "the initiating form stays mounted during its own submission");
  await f.render(false);
  await f.render(true);
  assert.equal(f.hook.isPending, true);
  assert.equal(f.hook.recoveryCommand.changes.name, "Original");
  await act(async () => {
    await assert.rejects(f.hook.submit(request()), /already in progress/);
  });
  const closed = assert.rejects(pending, /editor was closed/);
  await act(async () => {
    response.resolve({ id: "published" });
    await closed;
  });
  assert.equal(!!f.hook.recoveryCommand, true);
  await act(async () => {
    assert.equal((await f.hook.submit(request())).id, "published");
  });
  assert.equal(f.commits.length, 2);
  assert.deepEqual(f.commits[1], f.commits[0]);
});

await test("an account change after POST suppresses acceptance while retaining no other account's payload", async (t) => {
  const f = await fixture(t);
  const response = Promise.withResolvers();
  f.operations.commit = () => response.promise;
  let accepted = 0,
    pending;
  await act(async () => {
    pending = f.hook.submit(request()).then(() => accepted++);
  });
  const rejection = assert.rejects(pending, /account session has changed/);
  await act(async () => {
    advanceIdentitySession(f.client);
    f.client.clear();
    f.client.setQueryData(identityKeys.currentUser(), { id: "other" });
    response.resolve({ id: "published" });
    await rejection;
  });
  assert.equal(accepted, 0);
  assert.equal(f.hook.recoveryCommand, undefined);
});

await test("preservation requires confirmation, cancellation retains input and a later save uses a fresh review", async (t) => {
  const f = await fixture(t);
  let reviews = 0;
  f.operations.preview = async (command) => {
    f.previews.push(structuredClone(command));
    return { id: `review-${++reviews}`, preserved_properties: 1, description_html: "<p>Preserved false</p>" };
  };
  const original = request();
  let pending;
  await act(async () => {
    pending = f.hook.submit(original);
  });
  assert.equal(f.hook.review.value.description_html, "<p>Preserved false</p>");
  assert.equal(f.commits.length, 0);
  const cancelled = assert.rejects(pending, /Changes were not saved/);
  await act(async () => {
    f.hook.decide(false);
    await cancelled;
  });
  assert.equal(f.hook.recoveryCommand, undefined);
  assert.equal(original.changes.name, "Original");
  await act(async () => {
    pending = f.hook.submit(original);
  });
  original.changes.name = "Changed after review";
  await act(async () => {
    f.hook.decide(true);
    await pending;
  });
  assert.equal(f.commits.length, 1);
  assert.equal(f.commits[0].review_id, "review-2");
  assert.equal(f.commits[0].changes.name, "Original");
});

await test("closing during preview or confirmation cannot commit or reopen the confirmation", async (t) => {
  const f = await fixture(t);
  const preview = Promise.withResolvers();
  f.operations.preview = () => preview.promise;
  let pending;
  await act(async () => {
    pending = f.hook.submit(request());
  });
  const closed = assert.rejects(pending, /editor was closed/);
  await f.render(false);
  await f.render(true);
  await act(async () => {
    preview.resolve({ id: "old", preserved_properties: 1, description_html: "Old review" });
    await closed;
  });
  assert.equal(f.hook.review, undefined);
  assert.equal(f.commits.length, 0);
  await act(async () => {
    pending = f.hook.submit(request());
  });
  assert.equal(f.hook.review.value.id, "old");
  const cancelled = assert.rejects(pending, /editor was closed/);
  await f.render(false);
  await cancelled;
  assert.equal(f.commits.length, 0);
});

await test("acknowledging a saved result permits a new edit after reopening", async (t) => {
  const f = await fixture(t);
  await act(async () => {
    await f.hook.submit(request());
  });
  await act(async () => {
    f.hook.acknowledge();
  });
  await f.render(false);
  await f.render(true);
  assert.equal(f.hook.recoveryCommand, undefined);
  const next = request();
  next.changes.name = "Next edit";
  await act(async () => {
    await f.hook.submit(next);
  });
  assert.equal(f.commits.length, 2);
  assert.equal(f.commits[1].changes.name, "Next edit");
});

await test("a proven uncommitted review returns the original editable draft and allows fresh review", async (t) => {
  const f = await fixture(t);
  let attempt = 0;
  f.operations.commit = async (command) => {
    f.commits.push(structuredClone(command));
    if (++attempt === 1) throw new Error("Response lost");
    if (attempt === 2) throw { response: { status: 409, data: { fields: { review_again: true } } } };
    return { id: "published" };
  };
  const draft = { name: "All draft fields", description_html: "<p>Kept</p>", custom_properties: { notes: "Unsaved" } };
  await act(async () => {
    await assert.rejects(f.hook.submit(request(), draft));
  });
  await f.render(false);
  await f.render(true);
  await act(async () => {
    await assert.rejects(f.hook.submit(request()));
  });
  assert.equal(f.hook.recoveryCommand, undefined);
  assert.deepEqual(f.hook.rejectedAttempt.draft, draft);
  const repaired = request();
  repaired.revision = 4;
  await act(async () => {
    await f.hook.submit(repaired, draft);
  });
  assert.equal(f.previews.length, 2);
  assert.equal(f.commits[2].revision, 4);
});

await test("recovered edits remain available through a subsequent pending save and validation rejection", async (t) => {
  const f = await fixture(t);
  f.operations.commit = async () => {
    throw { response: { status: 409, data: { fields: { review_again: true } } } };
  };
  const draft = { name: "Recovered", custom_properties: { notes: "Unsaved" } };
  await act(async () => {
    await assert.rejects(f.hook.submit(request(), draft));
  });
  await f.render(false);
  await f.render(true);
  const response = Promise.withResolvers();
  f.operations.commit = () => response.promise;
  const repaired = { ...draft, name: "Repaired" };
  let pending;
  await act(async () => {
    pending = f.hook.submit(request(), repaired);
  });
  assert.deepEqual(f.hook.rejectedAttempt.draft, repaired);
  const rejected = assert.rejects(pending);
  await act(async () => {
    response.reject({ response: { status: 400 } });
    await rejected;
  });
  assert.deepEqual(f.hook.rejectedAttempt.draft, repaired);
  assert.equal(f.hook.recoveryCommand, undefined);
  f.operations.commit = async () => ({ id: "saved" });
  await act(async () => {
    await f.hook.submit(request(), f.hook.rejectedAttempt.draft);
    f.hook.acknowledge();
  });
  assert.equal(f.hook.rejectedAttempt, undefined);
});

await test("permission change during preservation preview rejects the late result without committing", async (t) => {
  const f = await fixture(t),
    response = Promise.withResolvers();
  let generation = 1;
  f.operations.captureAccess = () => {
    const captured = generation;
    return () => {
      if (generation !== captured) throw new Error("Access changed");
    };
  };
  f.operations.preview = () => response.promise;
  let saving;
  await act(async () => {
    saving = f.hook.submit(request());
  });
  const rejected = assert.rejects(saving, /Access changed/);
  generation = 2;
  await act(async () => {
    response.resolve({ id: "review", preserved_properties: 1, description_html: "<p>Private</p>" });
    await rejected;
  });
  assert.equal(f.hook.review, undefined);
  assert.equal(f.commits.length, 0);
});
await test("permission loss closes an open preservation confirmation without discarding uncertain receipts", async (t) => {
  const f = await fixture(t);
  let allowed = true;
  f.operations.captureAccess = () => () => {
    if (!allowed) throw new Error("Access denied");
  };
  f.operations.preview = async () => ({ id: "review", preserved_properties: 1, description_html: "<p>Private</p>" });
  let saving;
  await act(async () => {
    saving = f.hook.submit(request());
  });
  assert.equal(f.hook.isReviewCurrent(), true);
  const rejected = assert.rejects(saving);
  allowed = false;
  await f.render(true);
  await rejected;
  assert.equal(f.hook.review, undefined);
  assert.equal(f.commits.length, 0);
  // A later uncertain commit is still retained when access is subsequently lost.
  allowed = true;
  f.operations.preview = async () => ({ id: "next-review", preserved_properties: 0, description_html: "" });
  f.operations.commit = async () => {
    throw new Error("Response lost");
  };
  await act(async () => {
    await assert.rejects(f.hook.submit(request()));
  });
  allowed = false;
  await f.render(true);
  assert.equal(f.hook.recoveryCommand.review_id, "next-review");
});

for (const change of ["edit", "move", "restrict"]) {
  void test(`successful receipt recovery revalidates an intervening ${change}`, async (t) => {
    const f = await fixture(t);
    let saved = { id: "published", project_id: "project", name: "Restored" };
    let denied = false;
    f.operations.commit = async (command) => {
      f.commits.push(structuredClone(command));
      if (denied) throw { response: { status: 403 } };
      return structuredClone(saved);
    };
    const original = request();
    original.restore = { property_id: "rich", version_id: "revision" };
    await act(async () => {
      await f.hook.submit(original);
    });
    // Simulate failed caller reconciliation, then another user's change.
    if (change === "edit") saved.name = "A newer edit";
    if (change === "move") saved.project_id = "destination";
    if (change === "restrict") denied = true;
    await f.render(false);
    await f.render(true);
    await act(async () => {
      if (denied) await assert.rejects(f.hook.submit(request()));
      else {
        const result = await f.hook.submitWithReceipt(request());
        assert.deepEqual(result.saved, saved);
        assert.deepEqual(result.command.restore, original.restore);
      }
    });
    assert.deepEqual(f.commits[1], f.commits[0]);
    assert.equal(f.previews.length, 1);
    assert.ok(f.hook.recoveryCommand);
  });
}
