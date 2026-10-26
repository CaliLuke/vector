/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
// types
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
// ui
import { AlertModalCore } from "@plane/ui";
// query keys
import { apiTokensOptions, deleteApiTokenOptions } from "@/queries/api-tokens";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  tokenId: string;
};

export function DeleteApiTokenModal(props: Props) {
  if (!props.isOpen) return null;
  return <DeleteApiTokenDialog key={props.tokenId} {...props} />;
}

function DeleteApiTokenDialog({ onClose, tokenId }: Props) {
  const queryClient = useQueryClient();
  const deleteToken = useMutation(deleteApiTokenOptions(queryClient));
  // router params
  const { t } = useTranslation();

  const handleClose = () => {
    if (deleteToken.isPending) return;
    onClose();
  };

  const handleDeletion = () => {
    deleteToken.mutate(tokenId, {
      onSuccess: (_deleted, _tokenId, query) => {
        if (!query || queryClient.getQueryCache().find({ queryKey: apiTokensOptions.queryKey, exact: true }) !== query)
          return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("workspace_settings.settings.api_tokens.delete.success.title"),
          message: t("workspace_settings.settings.api_tokens.delete.success.message"),
        });
        onClose();
      },
      onError: (err, _tokenId, query) => {
        if (!query || queryClient.getQueryCache().find({ queryKey: apiTokensOptions.queryKey, exact: true }) !== query)
          return;
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("workspace_settings.settings.api_tokens.delete.error.title"),
          message: err?.message ?? t("workspace_settings.settings.api_tokens.delete.error.message"),
        });
      },
    });
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDeletion}
      isSubmitting={deleteToken.isPending}
      isOpen
      title={t("workspace_settings.settings.api_tokens.delete.title")}
      content={<>{t("workspace_settings.settings.api_tokens.delete.description")} </>}
    />
  );
}
