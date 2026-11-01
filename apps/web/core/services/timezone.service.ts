/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { TTimezones } from "@plane/types";
// helpers
// api services
import { APIService } from "@/services/api.service";

class TimezoneService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async fetch(): Promise<TTimezones> {
    return this.responseData(this.get(`/api/timezones/`));
  }
}

const timezoneService = new TimezoneService();

export default timezoneService;
