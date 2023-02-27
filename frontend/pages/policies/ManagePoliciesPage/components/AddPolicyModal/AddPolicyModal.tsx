import React, { useCallback, useContext } from "react";
import PATHS from "router/paths";
import { InjectedRouter } from "react-router/lib/Router";
import ReactTooltip from "react-tooltip";

import { DEFAULT_POLICY, DEFAULT_POLICIES } from "pages/policies/constants";

import { IPolicyNew } from "interfaces/policy";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import CustomLink from "components/CustomLink";

export interface IAddPolicyModalProps {
  onCancel: () => void;
  router: InjectedRouter; // v3
  teamId: number;
  teamName?: string;
}

const baseClass = "add-policy-modal";

const AddPolicyModal = ({
  onCancel,
  router,
  teamId,
  teamName,
}: IAddPolicyModalProps): JSX.Element => {
  const { currentTeam } = useContext(AppContext);
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
    setPolicyTeamId,
    setDefaultPolicy,
  } = useContext(PolicyContext);

  const onAddPolicy = (selectedPolicy: IPolicyNew) => {
    setDefaultPolicy(true);
    teamName
      ? setLastEditedQueryName(`${selectedPolicy.name} (${teamName})`)
      : setLastEditedQueryName(selectedPolicy.name);
    setLastEditedQueryDescription(selectedPolicy.description);
    setLastEditedQueryBody(selectedPolicy.query);
    setLastEditedQueryResolution(selectedPolicy.resolution);
    setLastEditedQueryCritical(selectedPolicy.critical || false);
    setPolicyTeamId(teamId);
    setLastEditedQueryPlatform(selectedPolicy.platform || null);
    router.push(PATHS.NEW_POLICY);
  };

  const onCreateYourOwnPolicyClick = useCallback(() => {
    setPolicyTeamId(currentTeam?.id || 0);
    setLastEditedQueryBody(DEFAULT_POLICY.query);
    router.push(PATHS.NEW_POLICY);
  }, [currentTeam]);

  const policiesAvailable = DEFAULT_POLICIES.map((policy: IPolicyNew) => {
    return (
      <Button
        key={policy.key}
        variant="unstyled-modal-query"
        className="modal-policy-button"
        onClick={() => onAddPolicy(policy)}
      >
        <>
          <div className={`${baseClass}__policy-name`}>
            <span className="info__header">{policy.name}</span>
            {policy.mdm_required && (
              <>
                <span
                  className={`${baseClass}__mdm-policy`}
                  data-tip
                  data-for="mdm-tooltip"
                >
                  MDM
                </span>
                <ReactTooltip
                  className="tooltip"
                  place="top"
                  type="dark"
                  effect="solid"
                  id="mdm-tooltip"
                  backgroundColor="#3e4771"
                  clickable
                  delayHide={200} // need delay set to hover using clickable
                >
                  <>
                    This table requires MDM settings <br />
                    to be enabled.{" "}
                    <CustomLink
                      url="https://fleetdm.com/docs/using-fleet/configuration-files#mobile-device-management-mdm-settings"
                      text="Learn more"
                      newTab
                      iconColor="core-fleet-white"
                    />
                  </>
                </ReactTooltip>
              </>
            )}
          </div>
          <span className="info__data">{policy.description}</span>
        </>
      </Button>
    );
  });

  return (
    <Modal title="Add a policy" onExit={onCancel} className={baseClass}>
      <>
        <div className={`${baseClass}__create-policy`}>
          Choose a policy template to get started or{" "}
          <Button variant="text-link" onClick={onCreateYourOwnPolicyClick}>
            create your own policy
          </Button>
          .
        </div>
        <div className={`${baseClass}__policy-selection`}>
          {policiesAvailable}
        </div>
      </>
    </Modal>
  );
};

export default AddPolicyModal;
