import { createRequire } from "node:module";

const requireFromWeb = createRequire(new URL("../../../apps/web/package.json", import.meta.url));
export const StoreContext = requireFromWeb("react").createContext(undefined);
