import assert from "node:assert/strict";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { EUserWorkspaceRoles } = await fromRoot("packages/types/dist/index.mjs");
const { React, act, createPermissionUiFixture, registerPermissionUiStubs, settleQuery } = await fromRoot(
  "tests/frontend/helpers/permission-ui-fixture.mjs"
);

const modalUrl = pathToFileURL(path.resolve("packages/ui/src/modals/modal-core.tsx")).href;
const constantsUrl = pathToFileURL(path.resolve("packages/ui/src/modals/constants.ts")).href;
const buttonUrl = pathToFileURL(path.resolve("packages/propel/src/button/button.tsx")).href;
registerPermissionUiStubs("/confirm-project-member-remove.tsx", {
  "@plane/ui": `export {ModalCore} from ${JSON.stringify(modalUrl)}; export {EModalPosition,EModalWidth} from ${JSON.stringify(constantsUrl)};`,
  "@plane/propel/button": `export {Button} from ${JSON.stringify(buttonUrl)};`,
});

const button = (label) => [...document.body.querySelectorAll("button")].find((item) => item.textContent === label);

async function testRemovalCase(memberId, action, pending, failure) {
  await test(`${action} remains open on failure, prevents pending dismissal, and permits retry`, async (t) => {
    const fixture = await createPermissionUiFixture(t, {
      path: "/acme/projects/project-1/settings/members/",
      workspaceRole: EUserWorkspaceRoles.ADMIN,
    });
    const { ConfirmProjectMemberRemove } = await fromRoot(
      "apps/web/core/components/project/confirm-project-member-remove.tsx"
    );
    let attempts = 0;
    let rejectFirst;
    const first = new Promise((_, reject) => {
      rejectFirst = reject;
    });
    const submit = async () => {
      attempts += 1;
      if (attempts === 1) await first;
    };
    function Subject() {
      const [open, setOpen] = React.useState(true);
      return React.createElement(ConfirmProjectMemberRemove, {
        data: { id: memberId, display_name: "Member" },
        isOpen: open,
        onSubmit: submit,
        onClose: () => setOpen(false),
      });
    }
    await fixture.render(React.createElement(Subject));
    await act(async () => new Promise((resolve) => setTimeout(resolve, 350)));

    await act(async () => button(action).click());
    await settleQuery();
    assert.equal(attempts, 1);
    assert.equal(button(pending).disabled, true);
    assert.equal(button("Cancel").disabled, true);
    await act(async () => {
      button("Cancel").click();
      document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    });
    assert.ok(document.body.querySelector('[role="dialog"]'));

    await act(async () => rejectFirst(new Error("Transport failed")));
    await settleQuery();
    assert.match(document.body.querySelector('[role="alert"]')?.textContent ?? "", new RegExp(failure));
    assert.equal(button(action).disabled, false);
    await act(async () => button(action).click());
    await settleQuery();
    assert.equal(attempts, 2);
    await act(async () => new Promise((resolve) => setTimeout(resolve, 350)));
    assert.equal(Boolean(document.body.querySelector('[role="dialog"]')), false);
  });
}

await testRemovalCase("another-user", "Remove", "Removing...", "Unable to remove the member");
await testRemovalCase("alice", "Leave", "Leaving...", "Unable to leave the project");
