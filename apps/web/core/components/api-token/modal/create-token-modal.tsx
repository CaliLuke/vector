/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
// plane imports
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IApiToken } from "@plane/types";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { renderFormattedDate } from "@plane/utils/datetime";
import { csvDownload } from "@plane/utils/file";
// constants
import { apiTokensOptions, createApiTokenOptions } from "@/queries/api-tokens";
// local imports
import { CreateApiTokenForm } from "./form";
import { GeneratedTokenDetails } from "./generated-token-details";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function CreateApiTokenModal(props: Props) {
  return props.isOpen ? <CreateApiTokenDialog onClose={props.onClose} /> : null;
}

const downloadSecretKey = (data: IApiToken) => {
  const csvData = {
    Title: data.label,
    Description: data.description,
    Expiry: data.expired_at ? (renderFormattedDate(data.expired_at)?.replace(",", " ") ?? "") : "Never expires",
    "Secret key": data.token ?? "",
  };

  csvDownload(csvData, `secret-key-${Date.now()}`);
};

function CreateApiTokenDialog({ onClose }: Pick<Props, "onClose">) {
  const queryClient = useQueryClient();
  const createToken = useMutation(createApiTokenOptions(queryClient));
  // states
  const [generatedToken, setGeneratedToken] = useState<IApiToken | null | undefined>(null);

  const handleCreateToken = async (data: Partial<IApiToken>) => {
    await createToken.mutateAsync(data, {
      onSuccess: (token, _input, query) => {
        if (!query || queryClient.getQueryCache().find({ queryKey: apiTokensOptions.queryKey, exact: true }) !== query)
          return;
        setGeneratedToken(token);
        try {
          downloadSecretKey(token);
        } catch {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Token created",
            message: "The CSV download failed. Copy the token from this dialog.",
          });
        }
      },
    });
  };

  return (
    <ModalCore isOpen handleClose={() => {}} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      {generatedToken ? (
        <GeneratedTokenDetails handleClose={onClose} tokenDetails={generatedToken} />
      ) : (
        <CreateApiTokenForm handleClose={onClose} onSubmit={handleCreateToken} />
      )}
    </ModalCore>
  );
}
