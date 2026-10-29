import assert from "node:assert/strict";
import test from "node:test";
import { act, settleQuery } from "../../../../tests/frontend/helpers/permission-ui-fixture.mjs";

function clipboard(t, write) {
  const target = navigator;
  const original = Object.getOwnPropertyDescriptor(target, "clipboard");
  const calls = [];
  Object.defineProperty(target, "clipboard", {
    configurable: true,
    value: {
      writeText: async (text) => {
        calls.push(text);
        await write();
      },
    },
  });
  t.after(() => {
    if (original) Object.defineProperty(target, "clipboard", original);
    else delete target.clipboard;
  });
  return calls;
}

export function registerWorkItemCopyTests(name, setup, identity, { readOnly = false } = {}) {
  void test(`${name} copies an absolute canonical work-item URL once`, async (t) => {
    const view = await setup(t);
    const calls = clipboard(t, async () => {});
    await view.click(view.copyButton());
    assert.deepEqual(calls, [new URL(view.path, window.location.origin).href]);
    assert.deepEqual(
      view.toasts().map((toast) => toast.type),
      ["success"]
    );
  });
  void test(`${name} prevents duplicate pending clipboard writes`, async (t) => {
    const view = await setup(t);
    const pending = Promise.withResolvers();
    const calls = clipboard(t, () => pending.promise);
    const button = view.copyButton();
    await act(() => {
      button.click();
      button.click();
    });
    await settleQuery();
    const count = calls.length;
    const disabled = view.copyButton().disabled;
    await act(() => pending.resolve());
    await settleQuery();
    assert.equal(count, 1);
    assert.equal(disabled, true);
    assert.equal(view.copyButton().disabled, false);
    assert.equal(view.toasts().length, 1);
  });
  void test(`${name} handles clipboard rejection once and allows retry`, async (t) => {
    const view = await setup(t);
    let reject = true;
    const calls = clipboard(t, async () => {
      if (reject) throw new Error("clipboard denied");
    });
    await view.click(view.copyButton());
    assert.deepEqual(
      view.toasts().map((toast) => toast.type),
      ["error"]
    );
    assert.equal(view.copyButton().disabled, false);
    reject = false;
    await view.click(view.copyButton());
    assert.equal(calls.length, 2);
    assert.deepEqual(
      view.toasts().map((toast) => toast.type),
      ["error", "success"]
    );
  });
  for (const retiredBy of ["session replacement", "unmount"]) {
    for (const rejects of [false, true]) {
      void test(`${name} suppresses late ${rejects ? "failure" : "success"} after ${retiredBy}`, async (t) => {
        const view = await setup(t);
        const pending = Promise.withResolvers();
        clipboard(t, () => pending.promise);
        await view.click(view.copyButton());
        if (retiredBy === "unmount") await view.unmount();
        else await act(() => identity.advanceIdentitySession(view.client));
        await settleQuery();
        await act(() => {
          if (rejects) pending.reject(new Error("late clipboard failure"));
          else pending.resolve();
        });
        await settleQuery();
        assert.deepEqual(view.toasts(), []);
      });
    }
  }
  if (readOnly)
    void test(`${name} permits copying a read-only work item`, async (t) => {
      const view = await setup(t);
      await view.renderReadOnly();
      const calls = clipboard(t, async () => {});
      assert.equal(view.copyButton().disabled, false);
      await view.click(view.copyButton());
      assert.deepEqual(calls, [new URL(view.path, window.location.origin).href]);
      assert.deepEqual(
        view.toasts().map((toast) => toast.type),
        ["success"]
      );
    });
}
