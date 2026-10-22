import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";

registerWebModuleHooks();

const [{ CopyIcon }, { InfoIcon }, { LockIcon }, { PriorityIcon }, { Tooltip }] = await Promise.all([
  import("../../packages/propel/src/icons/actions/copy-icon.tsx"),
  import("../../packages/propel/src/icons/misc/info-icon.tsx"),
  import("../../packages/propel/src/icons/actions/lock-icon.tsx"),
  import("../../packages/propel/src/icons/priority-icon.tsx"),
  import("../../packages/propel/src/tooltip/root.tsx"),
]);

const iconCases = [
  { name: "CopyIcon", Icon: CopyIcon, props: { className: "h-4 w-4" }, nodeName: "svg" },
  { name: "InfoIcon", Icon: InfoIcon, props: { className: "h-4 w-4" }, nodeName: "svg" },
  { name: "LockIcon", Icon: LockIcon, props: { className: "h-4 w-4" }, nodeName: "svg" },
  {
    name: "PriorityIcon",
    Icon: PriorityIcon,
    props: { priority: "high", withContainer: true, className: "h-3 w-3" },
    nodeName: "div",
  },
  {
    name: "PriorityIcon without container",
    Icon: PriorityIcon,
    props: { priority: "high", withContainer: false, className: "h-3 w-3" },
    nodeName: "svg",
  },
];

for (const { name, Icon, props, nodeName } of iconCases) {
  test(`${name} forwards Tooltip refs and consumer events to its rendered trigger`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());

    const errors = [];
    const originalError = console.error;
    console.error = (...args) => errors.push(args.join(" "));
    t.after(() => {
      console.error = originalError;
    });

    const ref = React.createRef();
    let clickCount = 0;
    await environment.render(
      React.createElement(
        Tooltip,
        { tooltipContent: `${name} tooltip`, openDelay: 0 },
        React.createElement(Icon, { ...props, onClick: () => clickCount++, ref })
      )
    );

    assert.equal(
      errors.some((message) => message.includes("Function components cannot be given refs")),
      false,
      `${name} must accept the ref attached by Tooltip.Trigger`
    );
    assert.equal(ref.current?.nodeName.toLowerCase(), nodeName);

    await React.act(async () => ref.current.dispatchEvent(new MouseEvent("click", { bubbles: true })));
    assert.equal(clickCount, 1, "the icon preserves caller event handlers while serving as a Tooltip trigger");
  });
}
