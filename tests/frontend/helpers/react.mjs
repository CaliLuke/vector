import { createRequire } from "./module-hooks.mjs";

export const webRequire = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
export const React = webRequire("react");
export const act = React.act;
