/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks

// layouts
import { useRouter } from "@tanstack/react-router";
import { DevErrorComponent } from "./dev";
import { ProdErrorComponent } from "./prod";

export function CustomErrorComponent({ error }: { error: unknown }) {
  // router
  const router = useRouter();

  const handleGoHome = () => void router.navigate({ href: "/" });
  const handleReload = () => window.location.reload();

  if (import.meta.env.DEV) {
    return <DevErrorComponent error={error} onGoHome={handleGoHome} onReload={handleReload} />;
  }

  return <ProdErrorComponent onGoHome={handleGoHome} />;
}
