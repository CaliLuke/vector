/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { MAX_FILE_SIZE } from "@plane/constants";
// hooks
import { useQuery } from "@tanstack/react-query";
import { instanceConfigOptions } from "@/queries/instance";

type TReturnProps = {
  maxFileSize: number;
};

export const useFileSize = (): TReturnProps => {
  // store hooks
  const { data: config } = useQuery(instanceConfigOptions);

  return {
    maxFileSize: config?.file_size_limit ?? MAX_FILE_SIZE,
  };
};
