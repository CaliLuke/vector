import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const load = (file) => import(`../../packages/propel/src/icons/${file}.tsx`);

for (const [file, name] of [
  ["create-icon", "CreateIcon"],
  ["photo-filter-icon", "PhotoFilterIcon"],
  ["dice-icon", "DiceIcon"],
  ["full-screen-panel-icon", "FullScreenPanelIcon"],
  ["side-panel-icon", "SidePanelIcon"],
]) {
  void test(`${name} retains its frame, caller sizing, styling, and events`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const { [name]: Icon } = await load(file);
    await env.render(React.createElement(Icon));
    let svg = env.container.querySelector("svg");
    assert.equal(svg.getAttribute("viewBox"), "0 0 24 24");
    assert.equal(svg.getAttribute("width"), null);
    assert.equal(svg.getAttribute("class"), "text-current stroke-2");
    assert.ok(svg.querySelectorAll("path").length >= 2);
    let clicks = 0;
    await env.render(
      React.createElement(Icon, {
        width: 32,
        height: 28,
        className: "toolbar",
        stroke: "red",
        "aria-label": "Action",
        onClick: () => clicks++,
      })
    );
    svg = env.container.querySelector("svg");
    assert.equal(svg.getAttribute("width"), "32");
    assert.equal(svg.getAttribute("height"), "28");
    assert.equal(svg.getAttribute("class"), "toolbar stroke-2");
    assert.equal(svg.getAttribute("stroke"), "red");
    assert.equal(svg.getAttribute("aria-label"), "Action");
    await act(async () => svg.dispatchEvent(new MouseEvent("click", { bubbles: true })));
    assert.equal(clicks, 1);
  });
}

await test("clipped icons use stable per-instance definitions without cross-icon collisions", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { AddIcon } = await load("actions/add-icon");
  const { AddCircleIcon } = await load("actions/add-circle-icon");
  const { ModuleCancelledIcon } = await load("module/cancelled");
  const { ModulePausedIcon } = await load("module/paused");
  const { GithubIcon } = await load("github-icon");
  const icons = [
    AddIcon,
    AddCircleIcon,
    ModuleCancelledIcon,
    ModuleCancelledIcon,
    ModulePausedIcon,
    ModulePausedIcon,
    GithubIcon,
    GithubIcon,
  ];
  const render = () =>
    env.render(
      React.createElement(
        React.Fragment,
        null,
        ...icons.map((Icon, index) => React.createElement(Icon, { key: index }))
      )
    );
  await render();
  const definitions = [...env.container.querySelectorAll("clipPath")];
  const ids = definitions.map((node) => node.id);
  assert.equal(new Set(ids).size, icons.length);
  for (const svg of env.container.querySelectorAll("svg")) {
    assert.equal(svg.querySelector("g").getAttribute("clip-path"), `url(#${svg.querySelector("clipPath").id})`);
  }
  const paths = [...env.container.querySelectorAll("svg")]
    .slice(0, 2)
    .map((svg) => svg.querySelector("path").getAttribute("d"));
  assert.equal(paths[0], paths[1], "both add exports retain the same symbol");
  await render();
  assert.deepEqual(
    [...env.container.querySelectorAll("clipPath")].map((node) => node.id),
    ids
  );
});

for (const [file, name, size, color] of [
  ["blocked-icon", "BlockedIcon", "24", "#F76659"],
  ["blocker-icon", "BlockerIcon", "24", "#F7AE59"],
  ["module/cancelled", "ModuleCancelledIcon", "20", "#ef4444"],
  ["module/paused", "ModulePausedIcon", "20", "#525252"],
  ["module/completed", "ModuleCompletedIcon", "20", "#16a34a"],
  ["module/planned", "ModulePlannedIcon", "20", "#3f76ff"],
  ["state/cancelled-group-icon", "CancelledGroupIcon", "20", "#9AA4BC"],
  ["state/completed-group-icon", "CompletedGroupIcon", "20", "#46A758"],
]) {
  void test(`${name} retains its status color and independent size overrides`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const { [name]: Icon } = await load(file);
    await env.render(React.createElement(Icon));
    let svg = env.container.querySelector("svg");
    assert.equal(svg.getAttribute("width"), size);
    assert.equal(svg.getAttribute("height"), size);
    assert.equal(svg.querySelector("path").getAttribute("fill"), color);
    await env.render(React.createElement(Icon, { width: 36, height: 30, className: "status" }));
    svg = env.container.querySelector("svg");
    assert.equal(svg.getAttribute("width"), "36");
    assert.equal(svg.getAttribute("height"), "30");
    assert.equal(svg.getAttribute("class"), "status");
    assert.equal(svg.querySelector("path").getAttribute("fill"), color);
  });
}

await test("filled state icons honor custom colors and forward accessible labels and events", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { CancelledGroupIcon } = await load("state/cancelled-group-icon");
  const { CompletedGroupIcon } = await load("state/completed-group-icon");
  let clicks = 0;
  await env.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(CancelledGroupIcon, { color: "red", "aria-label": "Cancelled", onClick: () => clicks++ }),
      React.createElement(CompletedGroupIcon, { color: "blue", "aria-label": "Completed", onClick: () => clicks++ })
    )
  );
  const icons = [...env.container.querySelectorAll("svg")];
  assert.deepEqual(
    icons.map((svg) => svg.querySelector("path").getAttribute("fill")),
    ["red", "blue"]
  );
  assert.deepEqual(
    icons.map((svg) => svg.getAttribute("aria-label")),
    ["Cancelled", "Completed"]
  );
  await act(async () => {
    icons[0].dispatchEvent(new MouseEvent("click", { bubbles: true }));
    icons[1].dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
  assert.equal(clicks, 2);
});

await test("Vector brand exports retain their distinct default dimensions and custom color", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { PlaneLockup } = await load("brand/plane-lockup");
  const { PlaneWordmark } = await load("brand/plane-wordmark");
  await env.render(
    React.createElement(React.Fragment, null, React.createElement(PlaneLockup), React.createElement(PlaneWordmark))
  );
  const icons = [...env.container.querySelectorAll("svg")];
  assert.deepEqual(
    icons.map((svg) => [svg.getAttribute("width"), svg.getAttribute("height")]),
    [
      ["253", "53"],
      ["146", "44"],
    ]
  );
  assert.deepEqual(
    icons.map((svg) => svg.getAttribute("aria-label")),
    ["Vector", "Vector"]
  );
  assert.deepEqual(
    icons.map((svg) => svg.textContent),
    ["Vector", "Vector"]
  );
  await env.render(React.createElement(PlaneLockup, { width: 95, height: 20, color: "red", className: "brand" }));
  const svg = env.container.querySelector("svg");
  assert.equal(svg.getAttribute("width"), "95");
  assert.equal(svg.getAttribute("height"), "20");
  assert.equal(svg.getAttribute("fill"), "red");
  assert.equal(svg.getAttribute("class"), "brand");
});

await test("GitLab renders the existing GitLab asset rather than the GitHub symbol", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { GitlabIcon } = await load("gitlab-icon");
  const asset = new window.DOMParser().parseFromString(
    readFileSync(new URL("../../apps/web/app/assets/logos/gitlab-logo.svg", import.meta.url), "utf8"),
    "image/svg+xml"
  );
  await env.render(React.createElement(GitlabIcon));
  assert.deepEqual(
    [...env.container.querySelectorAll("path")].map((path) => path.getAttribute("d")),
    [...asset.querySelectorAll("path")].map((path) => path.getAttribute("d"))
  );
  assert.equal(env.container.querySelector("svg").getAttribute("width"), "24");
  assert.equal(env.container.querySelector("path").getAttribute("fill"), "#e24329");
  await env.render(React.createElement(GitlabIcon, { color: "red", width: 40, height: 36, className: "provider" }));
  assert.deepEqual(
    [...env.container.querySelectorAll("path")].map((path) => path.getAttribute("fill")),
    ["red", "red", "red", "red"]
  );
  assert.equal(env.container.querySelector("svg").getAttribute("height"), "36");
  assert.equal(env.container.querySelector("svg").getAttribute("class"), "provider");
});

for (const [file, name, centerCount] of [
  ["state/started-group-icon", "StartedGroupIcon", 1],
  ["state/unstarted-group-icon", "UnstartedGroupIcon", 0],
]) {
  for (const [percentage, fraction, segments] of [
    [undefined, 1, 0],
    [0, 0, 15],
    [0.5, 0.5, 7],
    [50, 0.5, 7],
    [1, 1, 0],
    [-3, 0, 15],
    [150, 1, 0],
    [NaN, 1, 0],
    [Infinity, 1, 0],
    [-Infinity, 1, 0],
    ["bad", 1, 0],
  ]) {
    void test(`${name} renders finite progress for ${String(percentage)}`, async (t) => {
      const env = await createReactDomTestEnvironment();
      t.after(() => env.unmount());
      const { [name]: Icon } = await load(file);
      await env.render(React.createElement(Icon, { percentage }));
      const svg = env.container.querySelector("svg");
      const progress = svg.querySelector('circle[cx="8"]');
      assert.ok(Math.abs(Number(progress.getAttribute("stroke-dashoffset")) - 12 * Math.PI * (1 - fraction)) < 1e-9);
      assert.equal(svg.querySelectorAll("line").length, segments);
      assert.equal(svg.querySelectorAll('circle[cx="6"]').length, centerCount);
      assert.equal(svg.getAttribute("width"), "20");
      assert.equal(progress.getAttribute("stroke"), "#F59E0B");
    });
  }
  void test(`${name} honors caller dimensions, styling, and color`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const { [name]: Icon } = await load(file);
    await env.render(
      React.createElement(Icon, { percentage: 50, width: 36, height: 30, className: "progress", color: "red" })
    );
    assert.equal(env.container.querySelector("svg").getAttribute("width"), "36");
    assert.equal(env.container.querySelector("svg").getAttribute("height"), "30");
    assert.equal(env.container.querySelector("svg").getAttribute("class"), "progress");
    assert.equal(env.container.querySelector('circle[cx="8"]').getAttribute("stroke"), "red");
  });
}
