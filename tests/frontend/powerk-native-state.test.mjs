import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { act, React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { NativeCommandStateProvider, CommandPaletteContext, PowerKRegistryContext } =
  await import("../../apps/web/core/providers/native-command-state.tsx");

function click(element) {
  element.dispatchEvent(new MouseEvent("click", { bubbles: true }));
}

function CommandPaletteProbe() {
  const state = React.useContext(CommandPaletteContext);
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("button", {
      id: "open",
      onClick: () => state.toggleCreateIssueModal(true, "PROJECT", ["p1"]),
    }),
    React.createElement("button", { id: "toggle", onClick: () => state.toggleCreateIssueModal() }),
    React.createElement("button", {
      id: "profile",
      onClick: () => state.toggleProfileSettingsModal({ activeTab: "security", isOpen: true }),
    }),
    React.createElement(
      "output",
      { id: "state" },
      JSON.stringify({
        issueOpen: state.isCreateIssueModalOpen,
        storeType: state.createIssueStoreType,
        allowed: state.createWorkItemAllowedProjectIds ?? null,
        profile: state.profileSettingsModal,
      })
    )
  );
}

function Registration({ commands }) {
  const { registerCommands } = React.useContext(PowerKRegistryContext);
  React.useEffect(() => registerCommands(commands), [commands, registerCommands]);
  return null;
}

await test("command palette state preserves modal payload and reset semantics", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.render(null);
    await environment.unmount();
  });
  await environment.render(
    React.createElement(NativeCommandStateProvider, null, React.createElement(CommandPaletteProbe))
  );
  const clickAndRead = async (buttonId) => {
    await act(async () => click(environment.container.querySelector(`#${buttonId}`)));
    return JSON.parse(environment.container.querySelector("#state").textContent);
  };

  assert.deepEqual(await clickAndRead("open"), {
    issueOpen: true,
    storeType: "PROJECT",
    allowed: ["p1"],
    profile: { activeTab: "general", isOpen: false },
  });
  assert.deepEqual(await clickAndRead("toggle"), {
    issueOpen: false,
    storeType: "PROJECT",
    allowed: null,
    profile: { activeTab: "general", isOpen: false },
  });
  assert.deepEqual(await clickAndRead("profile"), {
    issueOpen: false,
    storeType: "PROJECT",
    allowed: null,
    profile: { activeTab: "security", isOpen: true },
  });
});

await test("Power K registrations are visible to readers and disposed by their owner", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const command = {
    id: "open-projects",
    i18n_title: "Open projects",
    group: "navigation",
    type: "action",
    closeOnSelect: true,
    isVisible: () => true,
    isEnabled: () => true,
    action: () => undefined,
  };
  const commands = [command];

  function Probe() {
    const [isRegistered, setRegistered] = React.useState(true);
    const { commandRegistry } = React.useContext(PowerKRegistryContext);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement("button", { id: "toggle-registration", onClick: () => setRegistered((value) => !value) }),
      React.createElement(
        "output",
        { id: "commands" },
        commandRegistry
          .getAllCommands()
          .map(({ id }) => id)
          .join(",")
      ),
      isRegistered && React.createElement(Registration, { commands })
    );
  }

  t.after(async () => {
    await environment.render(null);
    await environment.unmount();
  });
  await environment.render(React.createElement(NativeCommandStateProvider, null, React.createElement(Probe)));
  assert.equal(environment.container.querySelector("#commands").textContent, "open-projects");
  await act(async () => click(environment.container.querySelector("#toggle-registration")));
  assert.equal(environment.container.querySelector("#commands").textContent, "");
});
