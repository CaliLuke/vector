import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  settleQuery,
  registerPermissionUiStubs,
} from "./helpers/permission-ui-fixture.mjs";

const copyToasts = [];
globalThis.markdownCopyToasts = copyToasts;
registerPermissionUiStubs("/use-workspace-markdown-copy.ts", {
  "@plane/propel/toast":
    'export const TOAST_TYPE={ERROR:"error"};export const setToast=value=>globalThis.markdownCopyToasts.push(value);',
});

const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { useWorkspaceMarkdownCopy } = await import("../../apps/web/core/hooks/use-workspace-markdown-copy.ts");
const { EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((done, fail) => {
    resolve = done;
    reject = fail;
  });
  return { promise, resolve, reject };
}

function waitFor(predicate, description) {
  if (predicate()) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const observer = new MutationObserver(() => {
      if (!predicate()) return;
      observer.disconnect();
      clearTimeout(timeout);
      resolve();
    });
    observer.observe(globalThis.document.body, { childList: true, subtree: true, attributes: true });
    const timeout = setTimeout(() => {
      observer.disconnect();
      reject(new Error(`Timed out waiting for ${description}`));
    }, 3000);
  });
}

const ignoreCopy = () => {};

function MarkdownCopyProbe({ editor, onCopied = ignoreCopy }) {
  const { actionLabel, actionDisabled, statusRole, statusMessage, handleAction } = useWorkspaceMarkdownCopy("acme");
  return React.createElement(
    "div",
    null,
    statusMessage ? React.createElement("p", { role: statusRole }, statusMessage) : null,
    React.createElement(
      "button",
      {
        type: "button",
        disabled: actionDisabled,
        onClick: () => handleAction(() => editor, onCopied),
        "data-testid": "copy",
      },
      actionLabel
    )
  );
}

await test("markdown copy waits for the native roster query and retries its failed read", async (t) => {
  const failedRead = deferred();
  const retriedRead = deferred();
  let fetchCount = 0;
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", () => {
    fetchCount++;
    return fetchCount === 1 ? failedRead.promise : retriedRead.promise;
  });

  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/pages/page-1/",
    userId: "viewer",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
  });
  let copyCount = 0;
  const editor = { copyMarkdownToClipboard: () => copyCount++ };
  await fixture.render(React.createElement(MarkdownCopyProbe, { editor }));

  const copyButton = fixture.environment.container.querySelector('[data-testid="copy"]');
  assert.equal(fetchCount, 1);
  assert.equal(copyButton.disabled, true);
  assert.equal(copyCount, 0);

  await act(async () => {
    failedRead.reject(new Error("Roster request failed"));
    await new Promise((resolve) => setTimeout(resolve, 30));
  });
  await settleQuery();
  assert.equal(
    fixture.environment.container.querySelector('[role="alert"]')?.textContent,
    "Workspace members could not be loaded. Retry before copying markdown."
  );
  assert.equal(copyButton.disabled, false);
  assert.equal(copyButton.textContent, "Retry loading workspace members");
  assert.equal(copyCount, 0);

  await act(async () => {
    copyButton.click();
  });
  await waitFor(() => fetchCount === 2, "the roster retry request");
  assert.equal(copyCount, 0);

  await act(async () => {
    retriedRead.resolve([
      {
        id: "membership-1",
        is_active: true,
        role: EUserWorkspaceRoles.MEMBER,
        member: { id: "member-1", display_name: "Member One" },
      },
    ]);
    await new Promise((resolve) => setTimeout(resolve, 30));
  });
  await settleQuery();
  assert.equal(fixture.environment.container.querySelector('[role="alert"]'), null);
  assert.equal(copyButton.textContent, "Copy markdown");

  await act(async () => {
    copyButton.click();
  });
  assert.equal(copyCount, 1);
});

await test("markdown copy failure keeps the action available without announcing success", async (t) => {
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", async () => []);
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/pages/page-1/",
    userId: "viewer",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
  });
  copyToasts.length = 0;
  let denied = true;
  const editor = {
    copyMarkdownToClipboard: () => {
      if (denied) throw new Error("clipboard denied");
    },
  };
  let successes = 0;
  await fixture.render(
    React.createElement(MarkdownCopyProbe, {
      editor,
      onCopied: () => {
        successes++;
      },
    })
  );
  await settleQuery();
  const button = fixture.environment.container.querySelector('[data-testid="copy"]');
  assert.equal(button.disabled, false);
  await act(async () => button.click());
  assert.equal(successes, 0);
  assert.equal(copyToasts.length, 1);
  assert.equal(copyToasts[0].type, "error");
  assert.equal(button.disabled, false);
  denied = false;
  await act(async () => button.click());
  assert.equal(successes, 1);
  assert.equal(copyToasts.length, 1);
});
