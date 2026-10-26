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
import { InstanceImageConfigForm } from "./form";

const InstanceImagePage = function InstanceImagePage(_props: Route.ComponentProps) {
  const { data: formattedConfig } = useQuery(instanceConfigurationsOptions);

  return (
    <PageWrapper
      header={{
        title: "Third-party image libraries",
        description: "Let your users search and choose images from third-party libraries",
      }}
    >
      {formattedConfig ? (
        <InstanceImageConfigForm config={formattedConfig} />
      ) : (
        <Loader className="space-y-8">
          <Loader.Item height="50px" width="50%" />
          <Loader.Item height="50px" width="20%" />
        </Loader>
      )}
    </PageWrapper>
  );
};

export const meta: Route.MetaFunction = () => [{ title: "Images Settings - God Mode" }];

export default InstanceImagePage;
