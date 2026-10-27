/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
// plane imports
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { EmptyStateCompact } from "@plane/propel/empty-state";
// components
import { CreateApiTokenModal } from "@/components/api-token/modal/create-token-modal";
import { ApiTokenListItem } from "@/components/api-token/token-list-item";
import { ProfileSettingsHeading } from "@/components/settings/profile/heading";
import { APITokenSettingsLoader } from "@/components/ui/loader/settings/api-token";
// constants
import { apiTokensOptions } from "@/queries/api-tokens";
import { MCPConnection } from "./mcp-connection";

export function APITokensProfileSettings() {
  return (
    <div className="w-full">
      <APITokensContent />
      <MCPConnection />
    </div>
  );
}

function APITokensContent() {
  // states
  const [isCreateTokenModalOpen, setIsCreateTokenModalOpen] = useState(false);
  const { data: tokens, isError, refetch } = useQuery(apiTokensOptions);
  // translation
  const { t } = useTranslation();

  if (!tokens) {
    if (isError) {
      return (
        <div role="alert" className="flex items-center gap-3 p-5 text-13 text-danger-primary">
          <span>Unable to load API tokens.</span>
          <Button variant="secondary" onClick={() => void refetch()}>
            Try again
          </Button>
        </div>
      );
    }
    return <APITokenSettingsLoader />;
  }

  return (
    <div className="w-full">
      <CreateApiTokenModal isOpen={isCreateTokenModalOpen} onClose={() => setIsCreateTokenModalOpen(false)} />
      <ProfileSettingsHeading
        title={t("account_settings.api_tokens.title")}
        description={t("account_settings.api_tokens.description")}
        control={
          <Button variant="primary" size="lg" onClick={() => setIsCreateTokenModalOpen(true)}>
            {t("workspace_settings.settings.api_tokens.add_token")}
          </Button>
        }
      />
      {isError && (
        <div role="alert" className="mt-4 flex items-center gap-3 text-13 text-danger-primary">
          <span>Could not refresh API tokens.</span>
          <Button variant="secondary" onClick={() => void refetch()}>
            Try again
          </Button>
        </div>
      )}
      <div className="mt-7">
        {tokens.length > 0 ? (
          tokens.map((token) => <ApiTokenListItem key={token.id} token={token} />)
        ) : (
          <EmptyStateCompact
            assetKey="token"
            assetClassName="size-20"
            title={t("settings_empty_state.tokens.title")}
            description={t("settings_empty_state.tokens.description")}
            actions={[
              {
                label: t("settings_empty_state.tokens.cta_primary"),
                onClick: () => {
                  setIsCreateTokenModalOpen(true);
                },
              },
            ]}
            align="start"
            rootClassName="py-20"
          />
        )}
      </div>
    </div>
  );
}
