import { registerWebModuleHooks } from "./web-module-hooks.mjs";

registerWebModuleHooks();

const [identity, permissions, favorites, favoriteQueries, favoriteService, workspaceService, types] = await Promise.all(
  [
    import("../../../apps/web/core/queries/identity.ts"),
    import("../../../apps/web/core/queries/permissions.ts"),
    import("../../../apps/web/core/hooks/use-favorites.ts"),
    import("../../../apps/web/core/queries/favorites.ts"),
    import("../../../apps/web/core/services/favorite/favorite.service.ts"),
    import("../../../apps/web/core/services/workspace.service.ts"),
    import("../../../packages/types/dist/index.mjs"),
  ]
);

export const favoriteQueryTestContext = {
  ...identity,
  ...permissions,
  ...favorites,
  ...favoriteQueries,
  ...favoriteService,
  ...workspaceService,
  ...types,
};
