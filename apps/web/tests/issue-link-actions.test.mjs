import assert from "node:assert/strict";
import test from "node:test";
import { mockClipboard } from "./helpers/work-item-copy-contract.mjs";
import {
  React,
  act,
  clickAndSettle as click,
  createPermissionUiFixture,
  settleQuery,
  registerPermissionUiStubs,
} from "../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../tests/frontend/helpers/web-test-modules.mjs";
globalThis.React = React;
globalThis.linkToasts = [];
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=value=>globalThis.linkToasts.push(value);",
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@plane/propel/icons":
    "const React=globalThis.React;export const CopyIcon=()=>React.createElement('i',{'data-icon':'copy'});export const LinkIcon=()=>null;export const EditIcon=()=>null;export const TrashIcon=()=>null;",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState=()=>({issueLinkData:globalThis.linkModalData,toggleIssueLinkModal:()=>{},setIssueLinkData:value=>{globalThis.linkModalData=value;}});",
  "@plane/propel/button": `const React=globalThis.React;export const Button=({children,loading,disabled,variant,size,...props})=>React.createElement('button',{...props,disabled:disabled||loading},children);`,
});
registerPermissionUiStubs("/issues/issue-detail/links/link-item.tsx", {
  "@plane/ui": `const React=globalThis.React;
export const CustomMenu=({children,disabled})=>React.createElement('div',null,React.Children.map(children,child=>React.cloneElement(child,{disabled:disabled||child.props.disabled})));
CustomMenu.MenuItem=({children,...props})=>React.createElement('button',{type:'button',...props},children);`,
});
registerPermissionUiStubs("/components/common/link-modal.tsx", {
  "@plane/ui": `const React=globalThis.React;export const ModalCore=({children})=>React.createElement('div',{role:'dialog'},children);export const Input=({hasError,...props})=>React.createElement('input',props);`,
});
const { IssueLinkCreateUpdateModal } =
  await import("../core/components/issues/issue-detail/links/create-update-link-modal.tsx");
const { LinkList } = await import("../core/components/issues/issue-detail/links/link-list.tsx");
const { useLinkOperations } = await import("../core/hooks/use-issue-link-operations.ts");
const { IssueService } = await import("../core/services/issue/issue.service.ts");
const identity = await import("../core/queries/identity.ts");
async function setup(t, { readOnly = false, modal = false, serviceType = "issues" } = {}) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: 20 },
  });
  const original = {
    id: "link",
    issue_id: "item",
    title: "Reference",
    url: "https://example.com/reference",
    created_at: "2026-10-01T00:00:00Z",
  };
  let links = [original];
  let operations;
  let closed = 0;
  let setModalOpen;
  const writes = [];
  globalThis.linkModalData = original;
  const transport = { write: async () => {} };
  globalThis.linkToasts = [];
  t.mock.method(IssueService.prototype, "get", async (path) => {
    assert.equal(path, `/api/workspaces/acme/projects/p1/${serviceType}/item/`);
    return { data: { id: "item", project_id: "p1", issue_link: links } };
  });
  const collectionPath = `/api/workspaces/acme/projects/p1/${serviceType}/item/${serviceType === "epics" ? "links" : "issue-links"}/`;
  t.mock.method(IssueService.prototype, "post", async (path, data) => {
    writes.push(["create", path, data]);
    assert.equal(path, collectionPath);
    await transport.write();
    const saved = { ...original, ...data, id: "created" };
    links.push(saved);
    return { status: 201, data: saved };
  });
  t.mock.method(IssueService.prototype, "patch", async (path, data) => {
    writes.push(["update", path, data]);
    assert.equal(path, `${collectionPath}link/`);
    await transport.write();
    links = links.map((row) => (row.id === "link" ? { ...row, ...data } : row));
    return { status: 200, data: links.find((row) => row.id === "link") };
  });
  t.mock.method(IssueService.prototype, "delete", async (path) => {
    writes.push(["remove", path]);
    assert.equal(path, `${collectionPath}link/`);
    await transport.write();
    links = links.filter((row) => row.id !== "link");
    return { status: 204 };
  });
  function Probe({ onOperations, onClosed, onModalControl }) {
    const current = useLinkOperations("acme", "p1", "item", serviceType);
    onOperations(current);
    const [open, setOpen] = React.useState(true);
    onModalControl(setOpen);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(LinkList, {
        workspaceSlug: "acme",
        projectId: "p1",
        issueId: "item",
        issueServiceType: serviceType,
        linkOperations: current,
        disabled: readOnly,
      }),
      modal
        ? React.createElement(IssueLinkCreateUpdateModal, {
            isModalOpen: open,
            handleOnClose: () => {
              onClosed();
              setOpen(false);
            },
            linkOperations: current,
            issueServiceType: serviceType,
          })
        : null
    );
  }
  await fixture.render(
    React.createElement(Probe, {
      onOperations: (value) => {
        operations = value;
      },
      onClosed: () => {
        closed++;
      },
      onModalControl: (value) => {
        setModalOpen = value;
      },
    })
  );
  await settleQuery();
  const copy = () =>
    fixture.environment.container.querySelector('[aria-label="common.actions.copy_link"]') ??
    fixture.environment.container.querySelector('[data-icon="copy"]').parentElement;
  const remove = () =>
    [...fixture.environment.container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.actions.delete"
    );
  return {
    ...fixture,
    writes,
    transport,
    collectionPath,
    copy,
    remove,
    click,
    operations: () => operations,
    closed: () => closed,
    open: (value) => setModalOpen(value),
    submit: async () => {
      await act(() =>
        fixture.environment.container
          .querySelector("form")
          .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
      );
      await settleQuery();
    },
  };
}
void test("link copying reports success only after the clipboard accepts it", async (t) => {
  const v = await setup(t);
  const pending = Promise.withResolvers();
  mockClipboard(t, () => pending.promise);
  await v.click(v.copy());
  const toastsBefore = globalThis.linkToasts.length;
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(toastsBefore, 0);
  assert.deepEqual(
    globalThis.linkToasts.map((x) => x.type),
    ["success"]
  );
});
void test("link copy prevents duplicate pending writes", async (t) => {
  const v = await setup(t);
  const pending = Promise.withResolvers();
  const calls = mockClipboard(t, () => pending.promise);
  const control = v.copy();
  await act(() => {
    control.click();
    control.click();
  });
  await settleQuery();
  const count = calls.length;
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(count, 1);
});
void test("link delete preserves the primary rejection for its caller", async (t) => {
  const v = await setup(t);
  const expected = { message: "delete failed" };
  v.transport.write = async () => {
    throw { response: { data: expected } };
  };
  await act(async () => {
    await assert.rejects(v.operations().remove("link"), (error) => error === expected);
  });
});
void test("link delete prevents duplicate pending requests", async (t) => {
  const v = await setup(t);
  const pending = Promise.withResolvers();
  v.transport.write = () => pending.promise;
  const control = v.remove();
  await act(() => {
    control.click();
    control.click();
  });
  await settleQuery();
  const disabled = v.remove().disabled;
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(v.writes.length, 1);
  assert.equal(disabled, true);
  assert.deepEqual(
    globalThis.linkToasts.map((toast) => toast.type),
    ["success"]
  );
});
for (const kind of ["create", "update", "remove"])
  for (const failed of [false, true])
    void test(`old-session link ${kind} completion is silent (${failed ? "failure" : "success"})`, async (t) => {
      const v = await setup(t);
      const pending = Promise.withResolvers();
      v.transport.write = () => pending.promise;
      let completion;
      await act(() => {
        const ops = v.operations();
        completion = (
          kind === "create"
            ? ops.create({ url: "https://example.com/new" })
            : kind === "update"
              ? ops.update("link", { title: "Updated" })
              : ops.remove("link")
        ).catch(() => {});
      });
      await settleQuery();
      await act(() => identity.advanceIdentitySession(v.client));
      await settleQuery();
      await act(async () => {
        if (failed) pending.reject(new Error("old write"));
        else pending.resolve();
        await completion;
      });
      await settleQuery();
      assert.deepEqual(globalThis.linkToasts, []);
    });
void test("read-only link rows still copy the exact external URL", async (t) => {
  const v = await setup(t, { readOnly: true });
  const calls = mockClipboard(t, async () => {});
  await v.click(v.copy());
  assert.deepEqual(calls, ["https://example.com/reference"]);
  assert.equal(v.remove().disabled, true);
  assert.deepEqual(v.writes, []);
});

for (const retiredBy of ["session", "unmount"])
  void test(`accepted link modal completion cannot close UI after ${retiredBy}`, async (t) => {
    const v = await setup(t, { modal: true });
    const pending = Promise.withResolvers();
    v.transport.write = () => pending.promise;
    await v.submit();
    assert.equal(v.writes.length, 1);
    if (retiredBy === "unmount") await v.render(null);
    else await act(() => identity.advanceIdentitySession(v.client));
    await settleQuery();
    await act(() => pending.resolve());
    await settleQuery();
    assert.equal(v.closed(), 0);
  });

void test("an old accepted link save cannot close a reopened modal", async (t) => {
  const v = await setup(t, { modal: true });
  const pending = Promise.withResolvers();
  v.transport.write = () => pending.promise;
  await v.submit();
  await act(() => v.open(false));
  await act(() => v.open(true));
  await settleQuery();
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(v.closed(), 0);
  assert.ok(v.environment.container.querySelector('[role="dialog"]'));
});
void test("a rejected link modal save preserves its values, reports the server error once, and retries", async (t) => {
  const v = await setup(t, { modal: true });
  v.transport.write = async () => {
    throw { response: { data: { error: "Invalid reference" } } };
  };
  await v.submit();
  assert.equal(v.closed(), 0);
  assert.deepEqual(globalThis.linkToasts, [
    { type: "error", title: "links.toasts.not_updated.title", message: "Invalid reference" },
  ]);
  assert.equal(v.environment.container.querySelector('[name="url"]').value, "https://example.com/reference");
  v.transport.write = async () => {};
  await v.submit();
  assert.equal(v.closed(), 1);
  assert.equal(v.writes.length, 2);
  assert.deepEqual(
    globalThis.linkToasts.map((x) => x.type),
    ["error", "success"]
  );
});
void test("clipboard failure reports one error and allows retry", async (t) => {
  const v = await setup(t);
  let reject = true;
  const calls = mockClipboard(t, async () => {
    if (reject) throw new Error("clipboard denied");
  });
  await v.click(v.copy());
  assert.deepEqual(
    globalThis.linkToasts.map((x) => x.type),
    ["error"]
  );
  assert.equal(v.copy().disabled, false);
  reject = false;
  await v.click(v.copy());
  assert.equal(calls.length, 2);
  assert.deepEqual(
    globalThis.linkToasts.map((x) => x.type),
    ["error", "success"]
  );
});

for (const serviceType of ["issues", "epics"])
  for (const kind of ["create", "update", "remove"])
    void test(`link ${kind} uses the ${serviceType} endpoint and updates its visible collection`, async (t) => {
      const v = await setup(t, { serviceType });
      const data = { title: "Updated reference", url: "https://example.com/new" };
      await act(async () => {
        const ops = v.operations();
        if (kind === "create") await ops.create(data);
        else if (kind === "update") await ops.update("link", data);
        else await ops.remove("link");
      });
      await settleQuery();
      const path = kind === "create" ? v.collectionPath : `${v.collectionPath}link/`;
      assert.deepEqual(v.writes, [kind === "remove" ? [kind, path] : [kind, path, data]]);
      const anchors = [...v.environment.container.querySelectorAll("a")];
      assert.equal(anchors.length, kind === "create" ? 2 : kind === "update" ? 1 : 0);
      if (kind !== "remove") assert.ok(anchors.some((anchor) => anchor.href === data.url));
      assert.deepEqual(
        globalThis.linkToasts.map((x) => x.type),
        ["success"]
      );
    });

void test("a link modal reports an unavailable save after access is revoked", async (t) => {
  const v = await setup(t, { modal: true });
  const { forgetItemReads } = await import("../core/queries/work-item-access.ts");
  await act(() =>
    forgetItemReads(v.client, {
      accountId: "alice",
      generation: identity.getIdentitySessionGeneration(v.client),
      workspaceSlug: "acme",
      projectId: "p1",
      issueId: "item",
      serviceType: "issues",
    })
  );
  await settleQuery();
  await v.submit();
  assert.deepEqual(v.writes, []);
  assert.equal(v.closed(), 0);
  assert.deepEqual(
    globalThis.linkToasts.map((toast) => toast.type),
    ["error"]
  );
});
