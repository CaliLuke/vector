/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "react-router";
import { BrandHeader } from "@plane/ui";

export function AuthHeader() {
  return <BrandHeader homeLink={<Link to="/" />} />;
}
