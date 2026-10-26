/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { RANDOM_EMOJI_CODES } from "@plane/constants";
import type { IProject } from "@plane/types";
import { getRandomCoverImage } from "@/helpers/cover-image.helper";

export const getProjectFormValues = (data: Partial<IProject> = {}) => ({
  cover_image_url: data.cover_image_url ?? getRandomCoverImage(),
  description: data.description ?? "",
  logo_props: data.logo_props ?? {
    in_use: "emoji",
    emoji: {
      value: RANDOM_EMOJI_CODES[Math.floor(Math.random() * RANDOM_EMOJI_CODES.length)],
    },
  },
  identifier: data.identifier ?? "",
  name: data.name ?? "",
  network: data.network ?? 2,
  project_lead: typeof data.project_lead === "string" ? data.project_lead : (data.project_lead?.id ?? null),
});
