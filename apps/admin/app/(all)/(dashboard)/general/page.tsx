/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { Button } from "@plane/propel/button";
import { Loader } from "@plane/ui";
import { useQuery } from "@tanstack/react-query";
import { instanceAdminsOptions, instanceInfoOptions } from "@/queries/instance";
// components
import { PageWrapper } from "@/components/common/page-wrapper";
// hooks
// local imports
import { GeneralConfigurationForm } from "./form";
// types
import type { Route } from "./+types/page";

function GeneralPage() {
  const info = useQuery(instanceInfoOptions);
  const admins = useRecordCollection(instanceAdminsOptions);

  return (
    <PageWrapper
      header={{
        title: "General settings",
        description:
          "Change the name of your instance and instance admin e-mail addresses. Enable or disable telemetry in your instance.",
      }}
    >
      {info.data && admins.data ? (
        <GeneralConfigurationForm instance={info.data.instance} instanceAdmins={admins.data} />
      ) : info.isError || admins.isError ? (
        <div role="alert">
          Unable to load instance settings.{" "}
          <Button
            onClick={() => {
              void info.refetch();
              void admins.refetch();
            }}
          >
            Retry
          </Button>
        </div>
      ) : (
        <Loader>
          <Loader.Item height="50px" />
          <Loader.Item height="50px" />
        </Loader>
      )}
    </PageWrapper>
  );
}

export const meta: Route.MetaFunction = () => [{ title: "General Settings - God Mode" }];

export default GeneralPage;
