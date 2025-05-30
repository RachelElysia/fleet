/** software/titles/:id */

import React, { useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import paths from "router/paths";
import useTeamIdParam from "hooks/useTeamIdParam";
import { AppContext } from "context/app";
import { ignoreAxiosError } from "interfaces/errors";
import {
  ISoftwareTitleDetails,
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
} from "interfaces/software";
import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";
import softwareAPI, {
  ISoftwareTitleResponse,
  IGetSoftwareTitleQueryKey,
} from "services/entities/software";

import { getPathWithQueryParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import Card from "components/Card";

import SoftwareDetailsSummary from "../components/cards/SoftwareDetailsSummary";
import SoftwareTitleDetailsTable from "./SoftwareTitleDetailsTable";
import DetailsNoHosts from "../components/cards/DetailsNoHosts";
import SoftwareInstallerCard from "./SoftwareInstallerCard";
import { getInstallerCardInfo } from "./helpers";

const baseClass = "software-title-details-page";

interface ISoftwareTitleDetailsRouteParams {
  id: string;
  team_id?: string;
}

type ISoftwareTitleDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareTitleDetailsRouteParams
>;

const SoftwareTitleDetailsPage = ({
  router,
  routeParams,
  location,
}: ISoftwareTitleDetailsPageProps) => {
  const {
    isPremiumTier,
    isOnGlobalTeam,
    isTeamAdmin,
    isTeamMaintainer,
    isTeamObserver,
  } = useContext(AppContext);
  const handlePageError = useErrorHandler();

  // TODO: handle non integer values
  const softwareId = parseInt(routeParams.id, 10);

  const {
    currentTeamId,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
  });

  const {
    data: softwareTitle,
    isLoading: isSoftwareTitleLoading,
    isError: isSoftwareTitleError,
    refetch: refetchSoftwareTitle,
  } = useQuery<
    ISoftwareTitleResponse,
    AxiosError,
    ISoftwareTitleDetails,
    IGetSoftwareTitleQueryKey[]
  >(
    [{ scope: "softwareById", softwareId, teamId: teamIdForApi }],
    ({ queryKey }) => softwareAPI.getSoftwareTitle(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      select: (data) => data.software_title,
      onError: (error) => {
        if (!ignoreAxiosError(error, [403, 404])) {
          handlePageError(error);
        }
      },
    }
  );

  const isAvailableForInstall =
    !!softwareTitle?.software_package || !!softwareTitle?.app_store_app;

  const onDeleteInstaller = useCallback(() => {
    if (softwareTitle?.versions?.length) {
      refetchSoftwareTitle();
      return;
    }

    // redirect to software titles page if no versions are available
    router.push(
      getPathWithQueryParams(paths.SOFTWARE_TITLES, {
        team_id: teamIdForApi,
      })
    );
  }, [refetchSoftwareTitle, router, softwareTitle, teamIdForApi]);

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderSoftwareInstallerCard = (title: ISoftwareTitleDetails) => {
    const hasPermission = Boolean(
      isOnGlobalTeam || isTeamAdmin || isTeamMaintainer || isTeamObserver
    );

    const showInstallerCard =
      currentTeamId !== APP_CONTEXT_ALL_TEAMS_ID &&
      hasPermission &&
      isAvailableForInstall;

    if (!showInstallerCard) {
      return null;
    }

    const {
      softwarePackage,
      name,
      version,
      addedTimestamp,
      status,
      isSelfService,
    } = getInstallerCardInfo(title);

    return (
      <SoftwareInstallerCard
        softwareInstaller={softwarePackage}
        name={name}
        version={version}
        addedTimestamp={addedTimestamp}
        status={status}
        isSelfService={isSelfService}
        softwareId={softwareId}
        teamId={currentTeamId ?? APP_CONTEXT_NO_TEAM_ID}
        onDelete={onDeleteInstaller}
        refetchSoftwareTitle={refetchSoftwareTitle}
      />
    );
  };

  const renderSoftwareVersionsCard = (title: ISoftwareTitleDetails) => {
    // Hide versions card for tgz_packages only
    if (title.source === "tgz_packages") return null;

    return (
      <Card
        borderRadiusSize="xxlarge"
        includeShadow
        className={`${baseClass}__versions-section`}
      >
        <h2>Versions</h2>
        <SoftwareTitleDetailsTable
          router={router}
          data={title.versions ?? []}
          isLoading={isSoftwareTitleLoading}
          teamIdForApi={teamIdForApi}
          isIPadOSOrIOSApp={isIpadOrIphoneSoftwareSource(title.source)}
          isAvailableForInstall={isAvailableForInstall}
          countsUpdatedAt={title.counts_updated_at}
        />
      </Card>
    );
  };

  const renderContent = () => {
    if (isSoftwareTitleLoading) {
      return <Spinner />;
    }

    if (isSoftwareTitleError) {
      return (
        <DetailsNoHosts
          header="Software not detected"
          details="Expecting to see software? Check back later."
        />
      );
    }

    if (softwareTitle) {
      return (
        <>
          <SoftwareDetailsSummary
            title={softwareTitle.name}
            type={formatSoftwareType(softwareTitle)}
            versions={softwareTitle.versions?.length ?? 0}
            hosts={softwareTitle.hosts_count}
            countsUpdatedAt={softwareTitle.counts_updated_at}
            queryParams={{
              software_title_id: softwareId,
              team_id: teamIdForApi,
            }}
            name={softwareTitle.name}
            source={softwareTitle.source}
            iconUrl={
              softwareTitle.app_store_app
                ? softwareTitle.app_store_app.icon_url
                : undefined
            }
          />
          {renderSoftwareInstallerCard(softwareTitle)}
          {renderSoftwareVersionsCard(softwareTitle)}
        </>
      );
    }

    return null;
  };

  return (
    <MainContent className={baseClass}>
      {isPremiumTier && (
        <TeamsHeader
          isOnGlobalTeam={isOnGlobalTeam}
          currentTeamId={currentTeamId}
          userTeams={userTeams}
          onTeamChange={onTeamChange}
        />
      )}
      <>{renderContent()}</>
    </MainContent>
  );
};

export default SoftwareTitleDetailsPage;
