/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { instanceConfigurationsOptions } from "@/queries/instance";
import { Loader } from "@plane/ui";
// components
import { PageWrapper } from "@/components/common/page-wrapper";
// hooks
// types
import type { Route } from "./+types/page";
// local
import { InstanceAIForm } from "./form";

const InstanceAIPage = function InstanceAIPage(_props: Route.ComponentProps) {
  const { data: formattedConfig } = useQuery(instanceConfigurationsOptions);

  return (
    <PageWrapper
      header={{
        title: "AI features for all your workspaces",
        description: "Configure your AI API credentials so Vector AI features are turned on for all your workspaces.",
      }}
    >
      {formattedConfig ? (
        <InstanceAIForm config={formattedConfig} />
      ) : (
        <Loader className="space-y-8">
          <Loader.Item height="50px" width="40%" />
          <div className="grid w-2/3 grid-cols-2 gap-x-8 gap-y-4">
            <Loader.Item height="50px" />
            <Loader.Item height="50px" />
          </div>
          <Loader.Item height="50px" width="20%" />
        </Loader>
      )}
    </PageWrapper>
  );
};

export const meta: Route.MetaFunction = () => [{ title: "Artificial Intelligence Settings - God Mode" }];

export default InstanceAIPage;
