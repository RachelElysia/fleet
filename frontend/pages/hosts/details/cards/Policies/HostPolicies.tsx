import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import { noop } from "lodash";

import paths from "router/paths";
import { isAndroid } from "interfaces/platform";
import { IHostPolicy } from "interfaces/policy";
import { PolicyResponse, SUPPORT_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import Card from "components/Card";
import CardHeader from "components/CardHeader";
import CustomLink from "components/CustomLink";

import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./HostPoliciesTable/HostPoliciesTableConfig";
import PolicyFailingCount from "./HostPoliciesTable/PolicyFailingCount";

const baseClass = "policies-card";

interface IPoliciesProps {
  policies: IHostPolicy[];
  isLoading: boolean;
  deviceUser?: boolean;
  togglePolicyDetailsModal: (policy: IHostPolicy) => void;
  hostPlatform: string;
  router: InjectedRouter;
  currentTeamId?: number;
}

interface IHostPoliciesRowProps extends Row {
  original: {
    id: number;
    response: "pass" | "fail";
  };
}

const Policies = ({
  policies,
  isLoading,
  deviceUser,
  togglePolicyDetailsModal,
  hostPlatform,
  router,
  currentTeamId,
}: IPoliciesProps): JSX.Element => {
  const tableHeaders = generatePolicyTableHeaders(
    togglePolicyDetailsModal,
    currentTeamId
  );
  if (deviceUser) {
    // Remove view all hosts link
    tableHeaders.pop();
  }
  const failingResponses: IHostPolicy[] =
    policies.filter((policy: IHostPolicy) => policy.response === "fail") || [];

  const onClickRow = useCallback(
    (row: IHostPoliciesRowProps) => {
      const { id: policyId, response: policyResponse } = row.original;

      const viewAllHostPath = getPathWithQueryParams(paths.MANAGE_HOSTS, {
        policy_id: policyId,
        policy_response:
          policyResponse === "pass"
            ? PolicyResponse.PASSING
            : PolicyResponse.FAILING,
        team_id: currentTeamId,
      });

      router.push(viewAllHostPath);
    },
    [router]
  );

  const renderHostPolicies = () => {
    if (hostPlatform === "ios" || hostPlatform === "ipados") {
      return (
        <EmptyTable
          header={<>Policies are not supported for this host</>}
          info={
            <>
              Interested in detecting device health issues on{" "}
              {hostPlatform === "ios" ? "iPhones" : "iPads"}?{" "}
              <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
            </>
          }
        />
      );
    }

    if (isAndroid(hostPlatform)) {
      return (
        <EmptyTable
          header={<>Policies are not supported for this host</>}
          info={
            <>
              Interested in detecting device health issues on Android hosts?{" "}
              <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
            </>
          }
        />
      );
    }

    if (policies.length === 0) {
      return (
        <EmptyTable
          header={
            <>
              No policies are checked{" "}
              {deviceUser ? `on your device` : `for this host`}
            </>
          }
          info={
            <>
              Expecting to see policies? Try selecting “Refetch” to ask{" "}
              {deviceUser ? `your device ` : `this host `}
              to report new vitals.
            </>
          }
        />
      );
    }

    return (
      <>
        {failingResponses?.length > 0 && (
          <PolicyFailingCount policyList={policies} deviceUser={deviceUser} />
        )}
        <TableContainer
          columnConfigs={tableHeaders}
          data={generatePolicyDataSet(policies)}
          isLoading={isLoading}
          defaultSortHeader="response"
          defaultSortDirection="asc"
          resultsTitle="policies"
          emptyComponent={() => <></>}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disableCount
          disableMultiRowSelect={!deviceUser} // Removes hover/click state if deviceUser
          isClientSidePagination
          onClickRow={deviceUser ? noop : onClickRow}
        />
      </>
    );
  };

  return (
    <Card
      className={baseClass}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <CardHeader header="Policies" />
      {renderHostPolicies()}
    </Card>
  );
};

export default Policies;
