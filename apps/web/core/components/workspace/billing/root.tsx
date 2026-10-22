/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { SettingsHeading } from "@/components/settings/heading";

export function BillingRoot() {
  return (
    <section>
      <SettingsHeading
        title="Self-hosted build"
        description="This installation does not use subscriptions or paid plans."
      />
    </section>
  );
}
