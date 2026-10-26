import { createRequire } from "node:module";

export const webRequire = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
export const React = webRequire("react");
export const act = React.act;
