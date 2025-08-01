package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/automatic_policy"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListPendingSoftwareInstalls(ctx context.Context, hostID uint) ([]string, error) {
	return ds.listUpcomingSoftwareInstalls(ctx, hostID, false)
}

func (ds *Datastore) ListReadyToExecuteSoftwareInstalls(ctx context.Context, hostID uint) ([]string, error) {
	return ds.listUpcomingSoftwareInstalls(ctx, hostID, true)
}

func (ds *Datastore) listUpcomingSoftwareInstalls(ctx context.Context, hostID uint, onlyReadyToExecute bool) ([]string, error) {
	extraWhere := ""
	if onlyReadyToExecute {
		extraWhere = " AND activated_at IS NOT NULL"
	}

	stmt := fmt.Sprintf(`
	SELECT
		execution_id
	FROM (
		SELECT
			execution_id,
			IF(activated_at IS NULL, 0, 1) as topmost,
			priority,
			created_at
		FROM
			upcoming_activities
		WHERE
			host_id = ? AND
			activity_type = 'software_install'
			%s
		ORDER BY topmost DESC, priority ASC, created_at ASC) as t
`, extraWhere)
	var results []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pending software installs")
	}
	return results, nil
}

func (ds *Datastore) GetSoftwareInstallDetails(ctx context.Context, executionId string) (*fleet.SoftwareInstallDetails, error) {
	const stmt = `
  SELECT
    hsi.host_id AS host_id,
    hsi.execution_id AS execution_id,
    hsi.software_installer_id AS installer_id,
    hsi.self_service AS self_service,
    COALESCE(si.pre_install_query, '') AS pre_install_condition,
    inst.contents AS install_script,
    uninst.contents AS uninstall_script,
    COALESCE(pisnt.contents, '') AS post_install_script
  FROM
    host_software_installs hsi
  INNER JOIN
    software_installers si
    ON hsi.software_installer_id = si.id
  LEFT OUTER JOIN
    script_contents inst
    ON inst.id = si.install_script_content_id
  LEFT OUTER JOIN
    script_contents uninst
    ON uninst.id = si.uninstall_script_content_id
  LEFT OUTER JOIN
    script_contents pisnt
    ON pisnt.id = si.post_install_script_content_id
  WHERE
    hsi.execution_id = ? AND
    hsi.canceled = 0

	UNION

  SELECT
    ua.host_id AS host_id,
    ua.execution_id AS execution_id,
    siua.software_installer_id AS installer_id,
		ua.payload->'$.self_service' AS self_service,
    COALESCE(si.pre_install_query, '') AS pre_install_condition,
    inst.contents AS install_script,
    uninst.contents AS uninstall_script,
    COALESCE(pisnt.contents, '') AS post_install_script
  FROM
    upcoming_activities ua
  INNER JOIN
    software_install_upcoming_activities siua
    ON ua.id = siua.upcoming_activity_id
  INNER JOIN
    software_installers si
    ON siua.software_installer_id = si.id
  LEFT OUTER JOIN
    script_contents inst
    ON inst.id = si.install_script_content_id
  LEFT OUTER JOIN
    script_contents uninst
    ON uninst.id = si.uninstall_script_content_id
  LEFT OUTER JOIN
    script_contents pisnt
    ON pisnt.id = si.post_install_script_content_id
  WHERE
    ua.execution_id = ? AND
		ua.activated_at IS NULL -- if already activated, then it is covered by the other SELECT
`

	result := &fleet.SoftwareInstallDetails{}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), result, stmt, executionId, executionId); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SoftwareInstallerDetails").WithName(executionId), "get software installer details")
		}
		return nil, ctxerr.Wrap(ctx, err, "get software install details")
	}

	expandedInstallScript, err := ds.ExpandEmbeddedSecrets(ctx, result.InstallScript)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expanding secrets in install script")
	}
	expandedPostInstallScript, err := ds.ExpandEmbeddedSecrets(ctx, result.PostInstallScript)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expanding secrets in post-install script")
	}
	expandedUninstallScript, err := ds.ExpandEmbeddedSecrets(ctx, result.UninstallScript)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expanding secrets in uninstall script")
	}

	result.InstallScript = expandedInstallScript
	result.PostInstallScript = expandedPostInstallScript
	result.UninstallScript = expandedUninstallScript

	return result, nil
}

func (ds *Datastore) MatchOrCreateSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (installerID, titleID uint, err error) {
	if payload.ValidatedLabels == nil {
		// caller must ensure this is not nil; if caller intends no labels to be created,
		// payload.ValidatedLabels should point to an empty struct.
		return 0, 0, errors.New("validated labels must not be nil")
	}

	titleID, err = ds.getOrGenerateSoftwareInstallerTitleID(ctx, payload)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "get or generate software installer title ID")
	}

	if err := ds.addSoftwareTitleToMatchingSoftware(ctx, titleID, payload); err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "add software title to matching software")
	}

	installScriptID, err := ds.getOrGenerateScriptContentsID(ctx, payload.InstallScript)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "get or generate install script contents ID")
	}

	uninstallScriptID, err := ds.getOrGenerateScriptContentsID(ctx, payload.UninstallScript)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "get or generate uninstall script contents ID")
	}

	var postInstallScriptID *uint
	if payload.PostInstallScript != "" {
		sid, err := ds.getOrGenerateScriptContentsID(ctx, payload.PostInstallScript)
		if err != nil {
			return 0, 0, ctxerr.Wrap(ctx, err, "get or generate post-install script contents ID")
		}
		postInstallScriptID = &sid
	}

	var tid *uint
	var globalOrTeamID uint
	if payload.TeamID != nil {
		globalOrTeamID = *payload.TeamID

		if *payload.TeamID > 0 {
			tid = payload.TeamID
		}
	}

	if err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		stmt := `
INSERT INTO software_installers (
	team_id,
	global_or_team_id,
	title_id,
	storage_id,
	filename,
	extension,
	version,
	package_ids,
	install_script_content_id,
	pre_install_query,
	post_install_script_content_id,
    uninstall_script_content_id,
	platform,
    self_service,
	user_id,
	user_name,
	user_email,
	fleet_maintained_app_id,
 	url,
 	upgrade_code
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, (SELECT name FROM users WHERE id = ?), (SELECT email FROM users WHERE id = ?), ?, ?, ?)`

		args := []interface{}{
			tid,
			globalOrTeamID,
			titleID,
			payload.StorageID,
			payload.Filename,
			payload.Extension,
			payload.Version,
			strings.Join(payload.PackageIDs, ","),
			installScriptID,
			payload.PreInstallQuery,
			postInstallScriptID,
			uninstallScriptID,
			payload.Platform,
			payload.SelfService,
			payload.UserID,
			payload.UserID,
			payload.UserID,
			payload.FleetMaintainedAppID,
			payload.URL,
			payload.UpgradeCode,
		}

		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			if IsDuplicate(err) {
				// already exists for this team/no team
				err = alreadyExists("SoftwareInstaller", payload.Title)
			}
			return err
		}

		id, _ := res.LastInsertId()
		installerID = uint(id) //nolint:gosec // dismiss G115

		if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, installerID, *payload.ValidatedLabels, softwareTypeInstaller); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert software installer labels")
		}

		if payload.CategoryIDs != nil {
			if err := setOrUpdateSoftwareInstallerCategoriesDB(ctx, tx, installerID, payload.CategoryIDs, softwareTypeInstaller); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert software installer categories")
			}
		}

		if payload.AutomaticInstall {
			var installerMetadata automatic_policy.InstallerMetadata
			if payload.AutomaticInstallQuery != "" {
				installerMetadata = automatic_policy.FMAInstallerMetadata{
					Title:    payload.Title,
					Platform: payload.Platform,
					Query:    payload.AutomaticInstallQuery,
				}
			} else {
				installerMetadata = automatic_policy.FullInstallerMetadata{
					Title:            payload.Title,
					Extension:        payload.Extension,
					BundleIdentifier: payload.BundleIdentifier,
					PackageIDs:       payload.PackageIDs,
					UpgradeCode:      payload.UpgradeCode,
				}
			}

			generatedPolicyData, err := automatic_policy.Generate(installerMetadata)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "generate automatic policy query data")
			}

			policy, err := ds.createAutomaticPolicy(ctx, tx, *generatedPolicyData, payload.TeamID, ptr.Uint(installerID), nil)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "create automatic policy")
			}

			payload.AddedAutomaticInstallPolicy = policy
		}

		return nil
	}); err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "insert software installer")
	}

	return installerID, titleID, nil
}

func setOrUpdateSoftwareInstallerCategoriesDB(ctx context.Context, tx sqlx.ExtContext, installerID uint, categoryIDs []uint, swType softwareType) error {
	// remove existing categories
	delArgs := []interface{}{installerID}
	delStmt := fmt.Sprintf(`DELETE FROM %[1]s_software_categories WHERE %[1]s_id = ?`, swType)
	if len(categoryIDs) > 0 {
		inStmt, args, err := sqlx.In(` AND software_category_id NOT IN (?)`, categoryIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete existing software categories query")
		}
		delArgs = append(delArgs, args...)
		delStmt += inStmt
	}
	_, err := tx.ExecContext(ctx, delStmt, delArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete existing software categories")
	}

	if len(categoryIDs) > 0 {

		stmt := `INSERT IGNORE INTO %[1]s_software_categories (%[1]s_id, software_category_id) VALUES %s`
		var placeholders string
		var insertArgs []any
		for _, lid := range categoryIDs {
			placeholders += "(?, ?),"
			insertArgs = append(insertArgs, installerID, lid)
		}
		placeholders = strings.TrimSuffix(placeholders, ",")

		_, err = tx.ExecContext(ctx, fmt.Sprintf(stmt, swType, placeholders), insertArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert software software categories")
		}
	}

	return nil
}

func (ds *Datastore) createAutomaticPolicy(ctx context.Context, tx sqlx.ExtContext, policyData automatic_policy.PolicyData, teamID *uint, softwareInstallerID *uint, vppAppsTeamsID *uint) (*fleet.Policy, error) {
	tmID := fleet.PolicyNoTeamID
	if teamID != nil {
		tmID = *teamID
	}
	availablePolicyName, err := getAvailablePolicyName(ctx, tx, tmID, policyData.Name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get available policy name")
	}
	var userID *uint
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil {
		userID = &ctxUser.ID
	}
	policy, err := newTeamPolicy(ctx, tx, tmID, userID, fleet.PolicyPayload{
		Name:                availablePolicyName,
		Query:               policyData.Query,
		Platform:            policyData.Platform,
		Description:         policyData.Description,
		SoftwareInstallerID: softwareInstallerID,
		VPPAppsTeamsID:      vppAppsTeamsID,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create automatic policy query")
	}

	return policy, nil
}

func getAvailablePolicyName(ctx context.Context, db sqlx.QueryerContext, teamID uint, tentativePolicyName string) (string, error) {
	availableName := tentativePolicyName
	for i := 2; ; i++ {
		var count int
		if err := sqlx.GetContext(ctx, db, &count, `SELECT COUNT(*) FROM policies WHERE team_id = ? AND name = ?`, teamID, availableName); err != nil {
			return "", ctxerr.Wrapf(ctx, err, "get policy by team and name")
		}
		if count == 0 {
			break
		}
		availableName = fmt.Sprintf("%s %d", tentativePolicyName, i)
	}
	return availableName, nil
}

func (ds *Datastore) getOrGenerateSoftwareInstallerTitleID(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
	selectStmt := `SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''`
	selectArgs := []any{payload.Title, payload.Source}
	insertStmt := `INSERT INTO software_titles (name, source, browser) VALUES (?, ?, '')`
	insertArgs := []any{payload.Title, payload.Source}

	if payload.BundleIdentifier != "" {
		// match by bundle identifier first, or standard matching if we don't have a bundle identifier match
		selectStmt = `SELECT id FROM software_titles WHERE bundle_identifier = ? OR (name = ? AND source = ? AND browser = '') ORDER BY bundle_identifier = ? DESC LIMIT 1`
		selectArgs = []any{payload.BundleIdentifier, payload.Title, payload.Source, payload.BundleIdentifier}
		insertStmt = `INSERT INTO software_titles (name, source, bundle_identifier, browser) VALUES (?, ?, ?, '')`
		insertArgs = append(insertArgs, payload.BundleIdentifier)
	}

	titleID, err := ds.optimisticGetOrInsert(ctx,
		&parameterizedStmt{
			Statement: selectStmt,
			Args:      selectArgs,
		},
		&parameterizedStmt{
			Statement: insertStmt,
			Args:      insertArgs,
		},
	)
	if err != nil {
		return 0, err
	}

	return titleID, nil
}

func (ds *Datastore) addSoftwareTitleToMatchingSoftware(ctx context.Context, titleID uint, payload *fleet.UploadSoftwareInstallerPayload) error {
	whereClause := "WHERE (s.name, s.source, s.browser) = (?, ?, '')"
	whereArgs := []any{payload.Title, payload.Source}
	if payload.BundleIdentifier != "" {
		whereClause = "WHERE s.bundle_identifier = ?"
		whereArgs = []any{payload.BundleIdentifier}
	}

	args := make([]any, 0, len(whereArgs))
	args = append(args, titleID)
	args = append(args, whereArgs...)
	updateSoftwareStmt := fmt.Sprintf(`
		    UPDATE software s
		    SET s.title_id = ?
		    %s`, whereClause)
	_, err := ds.writer(ctx).ExecContext(ctx, updateSoftwareStmt, args...)
	return ctxerr.Wrap(ctx, err, "adding fk reference in software to software_titles")
}

type softwareType string

const (
	softwareTypeInstaller softwareType = "software_installer"
	softwareTypeVPP       softwareType = "vpp_app_team"
)

// setOrUpdateSoftwareInstallerLabelsDB sets or updates the label associations for the specified software
// installer. If no labels are provided, it will remove all label associations with the software installer.
func setOrUpdateSoftwareInstallerLabelsDB(ctx context.Context, tx sqlx.ExtContext, installerID uint, labels fleet.LabelIdentsWithScope, softwareType softwareType) error {
	labelIds := make([]uint, 0, len(labels.ByName))
	for _, label := range labels.ByName {
		labelIds = append(labelIds, label.LabelID)
	}

	// remove existing labels
	delArgs := []interface{}{installerID}
	delStmt := fmt.Sprintf(`DELETE FROM %[1]s_labels WHERE %[1]s_id = ?`, softwareType)
	if len(labelIds) > 0 {
		inStmt, args, err := sqlx.In(` AND label_id NOT IN (?)`, labelIds)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete existing software labels query")
		}
		delArgs = append(delArgs, args...)
		delStmt += inStmt
	}
	_, err := tx.ExecContext(ctx, delStmt, delArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete existing software labels")
	}

	// insert new labels
	if len(labelIds) > 0 {
		var exclude bool
		switch labels.LabelScope {
		case fleet.LabelScopeIncludeAny:
			exclude = false
		case fleet.LabelScopeExcludeAny:
			exclude = true
		default:
			// this should never happen
			return ctxerr.New(ctx, "invalid label scope")
		}

		stmt := `INSERT INTO %[1]s_labels (%[1]s_id, label_id, exclude) VALUES %s ON DUPLICATE KEY UPDATE exclude = VALUES(exclude)`
		var placeholders string
		var insertArgs []interface{}
		for _, lid := range labelIds {
			placeholders += "(?, ?, ?),"
			insertArgs = append(insertArgs, installerID, lid, exclude)
		}
		placeholders = strings.TrimSuffix(placeholders, ",")

		_, err = tx.ExecContext(ctx, fmt.Sprintf(stmt, softwareType, placeholders), insertArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert software label")
		}
	}

	return nil
}

func (ds *Datastore) UpdateInstallerSelfServiceFlag(ctx context.Context, selfService bool, id uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE software_installers SET self_service = ? WHERE id = ?`, selfService, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update software installer")
	}

	return nil
}

func (ds *Datastore) SaveInstallerUpdates(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload) error {
	if payload.InstallScript == nil || payload.UninstallScript == nil || payload.PreInstallQuery == nil || payload.SelfService == nil {
		return ctxerr.Wrap(ctx, errors.New("missing installer update payload fields"), "update installer record")
	}

	installScriptID, err := ds.getOrGenerateScriptContentsID(ctx, *payload.InstallScript)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get or generate install script contents ID")
	}

	uninstallScriptID, err := ds.getOrGenerateScriptContentsID(ctx, *payload.UninstallScript)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get or generate uninstall script contents ID")
	}

	var postInstallScriptID *uint
	if payload.PostInstallScript != nil && *payload.PostInstallScript != "" { // pointer because optional
		sid, err := ds.getOrGenerateScriptContentsID(ctx, *payload.PostInstallScript)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get or generate post-install script contents ID")
		}
		postInstallScriptID = &sid
	}

	var touchUploaded string
	var clearFleetMaintainedAppID string // FMA becomes custom package when uploading a new installer file
	if payload.InstallerFile != nil {
		touchUploaded = ", uploaded_at = NOW()"
		clearFleetMaintainedAppID = ", fleet_maintained_app_id = NULL"
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		stmt := fmt.Sprintf(`UPDATE software_installers SET
			storage_id = ?,
			filename = ?,
			version = ?,
			package_ids = ?,
			install_script_content_id = ?,
			pre_install_query = ?,
			post_install_script_content_id = ?,
			uninstall_script_content_id = ?,
			self_service = ?,
			upgrade_code = ?,
			user_id = ?,
			user_name = (SELECT name FROM users WHERE id = ?),
			user_email = (SELECT email FROM users WHERE id = ?)%s%s
			WHERE id = ?`, touchUploaded, clearFleetMaintainedAppID)

		args := []interface{}{
			payload.StorageID,
			payload.Filename,
			payload.Version,
			strings.Join(payload.PackageIDs, ","),
			installScriptID,
			*payload.PreInstallQuery,
			postInstallScriptID,
			uninstallScriptID,
			*payload.SelfService,
			payload.UpgradeCode,
			payload.UserID,
			payload.UserID,
			payload.UserID,
			payload.InstallerID,
		}

		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "update software installer")
		}

		if payload.ValidatedLabels != nil {
			if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, payload.InstallerID, *payload.ValidatedLabels, softwareTypeInstaller); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert software installer labels")
			}
		}

		if payload.CategoryIDs != nil {
			if err := setOrUpdateSoftwareInstallerCategoriesDB(ctx, tx, payload.InstallerID, payload.CategoryIDs, softwareTypeInstaller); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert software installer categories")
			}
		}

		return nil
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update software installer")
	}

	return nil
}

func (ds *Datastore) ValidateOrbitSoftwareInstallerAccess(ctx context.Context, hostID uint, installerID uint) (bool, error) {
	// NOTE: this is ok to only look in host_software_installs (and ignore
	// upcoming_activities), because orbit should not be able to get the
	// installer until it is ready to install.
	query := `
    SELECT 1
    FROM
      host_software_installs
    WHERE
      software_installer_id = ? AND
      host_id = ? AND
      install_script_exit_code IS NULL AND
      canceled = 0
`
	var access bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &access, query, installerID, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "check software installer association to host")
	}
	return true, nil
}

func (ds *Datastore) GetSoftwareInstallerMetadataByID(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
	query := `
SELECT
	si.id,
	si.team_id,
	si.title_id,
	si.storage_id,
	si.package_ids,
	si.filename,
	si.extension,
	si.version,
	si.install_script_content_id,
	si.pre_install_query,
	si.post_install_script_content_id,
	si.uninstall_script_content_id,
	si.uploaded_at,
	COALESCE(st.name, '') AS software_title,
	si.platform,
	si.fleet_maintained_app_id,
	si.upgrade_code
FROM
	software_installers si
	LEFT OUTER JOIN software_titles st ON st.id = si.title_id
WHERE
	si.id = ?`

	var dest fleet.SoftwareInstaller
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SoftwareInstaller").WithID(id), "get software installer metadata")
		}
		return nil, ctxerr.Wrap(ctx, err, "get software installer metadata")
	}

	return &dest, nil
}

func (ds *Datastore) GetSoftwareInstallerMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
	var scriptContentsSelect, scriptContentsFrom string
	if withScriptContents {
		scriptContentsSelect = ` , inst.contents AS install_script, COALESCE(pinst.contents, '') AS post_install_script, uninst.contents AS uninstall_script `
		scriptContentsFrom = ` LEFT OUTER JOIN script_contents inst ON inst.id = si.install_script_content_id
		LEFT OUTER JOIN script_contents pinst ON pinst.id = si.post_install_script_content_id
		LEFT OUTER JOIN script_contents uninst ON uninst.id = si.uninstall_script_content_id`
	}

	query := fmt.Sprintf(`
SELECT
  si.id,
  si.team_id,
  si.title_id,
  si.storage_id,
  si.fleet_maintained_app_id,
  si.package_ids,
  si.upgrade_code,
  si.filename,
  si.extension,
  si.version,
  si.platform,
  si.install_script_content_id,
  si.pre_install_query,
  si.post_install_script_content_id,
  si.uninstall_script_content_id,
  si.uploaded_at,
  si.self_service,
  si.url,
  COALESCE(st.name, '') AS software_title
  %s
FROM
  software_installers si
  JOIN software_titles st ON st.id = si.title_id
  %s
WHERE
  si.title_id = ? AND si.global_or_team_id = ?`,
		scriptContentsSelect, scriptContentsFrom)

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var dest fleet.SoftwareInstaller
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, titleID, tmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SoftwareInstaller"), "get software installer metadata")
		}
		return nil, ctxerr.Wrap(ctx, err, "get software installer metadata")
	}

	// TODO: do we want to include labels on other queries that return software installer metadata
	// (e.g., GetSoftwareInstallerMetadataByID)?
	labels, err := ds.getSoftwareInstallerLabels(ctx, dest.InstallerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installer labels")
	}
	var exclAny, inclAny []fleet.SoftwareScopeLabel
	for _, l := range labels {
		if l.Exclude {
			exclAny = append(exclAny, l)
		} else {
			inclAny = append(inclAny, l)
		}
	}

	if len(inclAny) > 0 && len(exclAny) > 0 {
		// there's a bug somewhere
		level.Warn(ds.logger).Log("msg", "software installer has both include and exclude labels", "installer_id", dest.InstallerID, "include", fmt.Sprintf("%v", inclAny), "exclude", fmt.Sprintf("%v", exclAny))
	}
	dest.LabelsExcludeAny = exclAny
	dest.LabelsIncludeAny = inclAny

	categoryMap, err := ds.GetCategoriesForSoftwareTitles(ctx, []uint{titleID}, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting categories for software installer metadata")
	}

	if categories, ok := categoryMap[titleID]; ok {
		dest.Categories = categories
	}

	policies, err := ds.getPoliciesBySoftwareTitleIDs(ctx, []uint{titleID}, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies by software title ID")
	}
	dest.AutomaticInstallPolicies = policies

	return &dest, nil
}

func (ds *Datastore) getSoftwareInstallerLabels(ctx context.Context, installerID uint) ([]fleet.SoftwareScopeLabel, error) {
	query := `
SELECT
	label_id,
	exclude,
	l.name as label_name,
	si.title_id
FROM
	software_installer_labels sil
	JOIN software_installers si ON si.id = sil.software_installer_id
	JOIN labels l ON l.id = sil.label_id
WHERE
	software_installer_id = ?`

	var labels []fleet.SoftwareScopeLabel
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, query, installerID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installer labels")
	}

	return labels, nil
}

var (
	errDeleteInstallerWithAssociatedPolicy = &fleet.ConflictError{Message: "Couldn't delete. Policy automation uses this software. Please disable policy automation for this software and try again."}
	errDeleteInstallerInstalledDuringSetup = &fleet.ConflictError{Message: "Couldn't delete. This software is installed when new Macs boot. Please remove software in Controls > Setup experience and try again."}
)

func (ds *Datastore) DeleteSoftwareInstaller(ctx context.Context, id uint) error {
	var activateAffectedHostIDs []uint

	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		affectedHostIDs, err := ds.runInstallerUpdateSideEffectsInTransaction(ctx, tx, id, true, true)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "clean up related installs and uninstalls")
		}
		activateAffectedHostIDs = affectedHostIDs

		// allow delete only if install_during_setup is false
		res, err := tx.ExecContext(ctx, `DELETE FROM software_installers WHERE id = ? AND install_during_setup = 0`, id)
		if err != nil {
			if isMySQLForeignKey(err) {
				// Check if the software installer is referenced by a policy automation.
				var count int
				if err := sqlx.GetContext(ctx, tx, &count, `SELECT COUNT(*) FROM policies WHERE software_installer_id = ?`, id); err != nil {
					return ctxerr.Wrapf(ctx, err, "getting reference from policies")
				}
				if count > 0 {
					return errDeleteInstallerWithAssociatedPolicy
				}
			}
			return ctxerr.Wrap(ctx, err, "delete software installer")
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			// could be that the software installer does not exist, or it is installed
			// during setup, do additional check.
			var installDuringSetup bool
			if err := sqlx.GetContext(ctx, tx, &installDuringSetup,
				`SELECT install_during_setup FROM software_installers WHERE id = ?`, id); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, err, "check if software installer is installed during setup")
			}
			if installDuringSetup {
				return errDeleteInstallerInstalledDuringSetup
			}
			return notFound("SoftwareInstaller").WithID(id)
		}

		return nil
	})
	if err != nil {
		return err
	}
	return ds.activateNextUpcomingActivityForBatchOfHosts(ctx, activateAffectedHostIDs)
}

// deletePendingSoftwareInstallsForPolicy should be called after a policy is
// deleted to remove any pending software installs
func (ds *Datastore) deletePendingSoftwareInstallsForPolicy(ctx context.Context, teamID *uint, policyID uint) error {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	// NOTE(mna): I'm adding the deletion for the upcoming_activities too, but I
	// don't think the existing code works as intended anyway as the
	// host_software_installs.policy_id column has a ON DELETE SET NULL foreign
	// key, so the deletion statement will not find any row.
	const deleteStmt = `
		DELETE FROM
			host_software_installs
		WHERE
			policy_id = ? AND
			status = ? AND
			software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ?
			)
	`
	_, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, policyID, fleet.SoftwareInstallPending, globalOrTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete pending software installs for policy")
	}

	const loadAffectedHostsStmt = `
		SELECT
			DISTINCT host_id
		FROM
			upcoming_activities ua
			INNER JOIN software_install_upcoming_activities siua
				ON ua.id = siua.upcoming_activity_id
		WHERE
			ua.activity_type = 'software_install' AND
			ua.activated_at IS NOT NULL AND
			siua.policy_id = ? AND
			siua.software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ?
			)`
	var affectedHosts []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &affectedHosts,
		loadAffectedHostsStmt, policyID, globalOrTeamID); err != nil {
		return ctxerr.Wrap(ctx, err, "load affected hosts for software installs")
	}

	const deleteUAStmt = `
		DELETE FROM
			upcoming_activities
		USING
			upcoming_activities
			INNER JOIN software_install_upcoming_activities siua
				ON upcoming_activities.id = siua.upcoming_activity_id
		WHERE
			upcoming_activities.activity_type = 'software_install' AND
			siua.policy_id = ? AND
			siua.software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ?
			)
	`
	_, err = ds.writer(ctx).ExecContext(ctx, deleteUAStmt, policyID, globalOrTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete upcoming software installs for policy")
	}

	return ds.activateNextUpcomingActivityForBatchOfHosts(ctx, affectedHosts)
}

func (ds *Datastore) InsertSoftwareInstallRequest(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
	const (
		getInstallerStmt = `
SELECT
	filename, "version", title_id, COALESCE(st.name, '[deleted title]') title_name
FROM
	software_installers si
	LEFT JOIN software_titles st
		ON si.title_id = st.id
WHERE si.id = ?`

		insertUAStmt = `
INSERT INTO upcoming_activities
	(host_id, priority, user_id, fleet_initiated, activity_type, execution_id, payload)
VALUES
	(?, ?, ?, ?, 'software_install', ?,
		JSON_OBJECT(
			'self_service', ?,
			'installer_filename', ?,
			'version', ?,
			'software_title_name', ?,
			'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = ?)
		)
	)`

		insertSIUAStmt = `
INSERT INTO software_install_upcoming_activities
	(upcoming_activity_id, software_installer_id, policy_id, software_title_id)
VALUES
	(?, ?, ?, ?)`

		hostExistsStmt = `SELECT 1 FROM hosts WHERE id = ?`
	)

	// we need to explicitly do this check here because we can't set a FK constraint on the schema
	var hostExists bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostExists, hostExistsStmt, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", notFound("Host").WithID(hostID)
		}

		return "", ctxerr.Wrap(ctx, err, "checking if host exists")
	}

	var installerDetails struct {
		Filename  string  `db:"filename"`
		Version   string  `db:"version"`
		TitleID   *uint   `db:"title_id"`
		TitleName *string `db:"title_name"`
	}
	if err = sqlx.GetContext(ctx, ds.reader(ctx), &installerDetails, getInstallerStmt, softwareInstallerID); err != nil {
		if err == sql.ErrNoRows {
			return "", notFound("SoftwareInstaller").WithID(softwareInstallerID)
		}

		return "", ctxerr.Wrap(ctx, err, "getting installer data")
	}

	var userID *uint
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil && opts.PolicyID == nil {
		userID = &ctxUser.ID
	}
	execID := uuid.NewString()

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, insertUAStmt,
			hostID,
			opts.Priority(),
			userID,
			opts.IsFleetInitiated(),
			execID,
			opts.SelfService,
			installerDetails.Filename,
			installerDetails.Version,
			installerDetails.TitleName,
			userID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert software install request")
		}

		activityID, _ := res.LastInsertId()
		_, err = tx.ExecContext(ctx, insertSIUAStmt,
			activityID,
			softwareInstallerID,
			opts.PolicyID,
			installerDetails.TitleID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert software install request join table")
		}

		if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, ""); err != nil {
			return ctxerr.Wrap(ctx, err, "activate next activity")
		}
		return nil
	})
	return execID, ctxerr.Wrap(ctx, err, "inserting new install software request")
}

func (ds *Datastore) ProcessInstallerUpdateSideEffects(ctx context.Context, installerID uint, wasMetadataUpdated bool, wasPackageUpdated bool) error {
	var activateAffectedHostIDs []uint

	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		affectedHostIDs, err := ds.runInstallerUpdateSideEffectsInTransaction(ctx, tx, installerID, wasMetadataUpdated, wasPackageUpdated)
		if err != nil {
			return err
		}
		activateAffectedHostIDs = affectedHostIDs
		return nil
	})
	if err != nil {
		return err
	}
	return ds.activateNextUpcomingActivityForBatchOfHosts(ctx, activateAffectedHostIDs)
}

func (ds *Datastore) runInstallerUpdateSideEffectsInTransaction(ctx context.Context, tx sqlx.ExtContext, installerID uint, wasMetadataUpdated bool, wasPackageUpdated bool) (affectedHostIDs []uint, err error) {
	if wasMetadataUpdated || wasPackageUpdated { // cancel pending installs/uninstalls
		// TODO make this less naive; this assumes that installs/uninstalls execute and report back immediately
		_, err := tx.ExecContext(ctx, `DELETE FROM host_script_results WHERE execution_id IN (
				SELECT execution_id FROM host_software_installs WHERE software_installer_id = ? AND status = 'pending_uninstall'
			)`, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete pending uninstall scripts")
		}

		_, err = tx.ExecContext(ctx, `UPDATE setup_experience_status_results SET status=? WHERE status IN (?, ?) AND host_software_installs_execution_id IN (
			  SELECT execution_id FROM host_software_installs WHERE software_installer_id = ? AND status IN ('pending_install', 'pending_uninstall')
			UNION
			  SELECT ua.execution_id FROM upcoming_activities ua INNER JOIN software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
			  WHERE siua.software_installer_id = ? AND activity_type IN ('software_install', 'software_uninstall')
		)`, fleet.SetupExperienceStatusCancelled, fleet.SetupExperienceStatusPending, fleet.SetupExperienceStatusRunning, installerID, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "fail setup experience results dependant on deleted software install")
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM host_software_installs
			   WHERE software_installer_id = ? AND status IN('pending_install', 'pending_uninstall')`, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete pending host software installs/uninstalls")
		}

		if err := sqlx.SelectContext(ctx, tx, &affectedHostIDs, `SELECT
			DISTINCT host_id
		FROM
			upcoming_activities ua
			INNER JOIN software_install_upcoming_activities siua
				ON ua.id = siua.upcoming_activity_id
		WHERE
			siua.software_installer_id = ? AND
			ua.activated_at IS NOT NULL AND
			ua.activity_type IN ('software_install', 'software_uninstall')`, installerID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "select affected host IDs for software installs/uninstalls")
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM upcoming_activities
			USING
				upcoming_activities
				INNER JOIN software_install_upcoming_activities siua
					ON upcoming_activities.id = siua.upcoming_activity_id
			WHERE siua.software_installer_id = ? AND activity_type IN ('software_install', 'software_uninstall')`, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete upcoming host software installs/uninstalls")
		}
	}

	if wasPackageUpdated { // hide existing install counts
		_, err := tx.ExecContext(ctx, `UPDATE host_software_installs SET removed = TRUE
	  			WHERE software_installer_id = ? AND status IS NOT NULL AND host_deleted_at IS NULL`, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "hide existing install counts")
		}
	}

	return affectedHostIDs, nil
}

func (ds *Datastore) InsertSoftwareUninstallRequest(ctx context.Context, executionID string, hostID uint, softwareInstallerID uint, selfService bool) error {
	const (
		getInstallerStmt = `SELECT title_id, COALESCE(st.name, '[deleted title]') title_name
			FROM software_installers si LEFT JOIN software_titles st ON si.title_id = st.id WHERE si.id = ?`

		insertUAStmt = `
INSERT INTO upcoming_activities
	(host_id, priority, user_id, fleet_initiated, activity_type, execution_id, payload)
VALUES
	(?, ?, ?, ?, 'software_uninstall', ?,
		JSON_OBJECT(
			'installer_filename', '',
			'version', 'unknown',
			'software_title_name', ?,
			'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = ?),
			'self_service', ?
		)
	)`

		insertSIUAStmt = `
INSERT INTO software_install_upcoming_activities
	(upcoming_activity_id, software_installer_id, software_title_id)
VALUES
	(?, ?, ?)`

		hostExistsStmt = `SELECT 1 FROM hosts WHERE id = ?`
	)

	// we need to explicitly do this check here because we can't set a FK constraint on the schema
	var hostExists bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostExists, hostExistsStmt, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notFound("Host").WithID(hostID)
		}
		return ctxerr.Wrap(ctx, err, "checking if host exists")
	}

	var installerDetails struct {
		TitleID   *uint   `db:"title_id"`
		TitleName *string `db:"title_name"`
	}
	if err = sqlx.GetContext(ctx, ds.reader(ctx), &installerDetails, getInstallerStmt, softwareInstallerID); err != nil {
		if err == sql.ErrNoRows {
			return notFound("SoftwareInstaller").WithID(softwareInstallerID)
		}

		return ctxerr.Wrap(ctx, err, "getting installer data")
	}

	var userID *uint
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil {
		userID = &ctxUser.ID
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, insertUAStmt,
			hostID,
			0, // Uninstalls are never used in setup experience, so always default priority
			userID,
			false,
			executionID,
			installerDetails.TitleName,
			userID,
			selfService,
		)
		if err != nil {
			return err
		}

		activityID, _ := res.LastInsertId()
		_, err = tx.ExecContext(ctx, insertSIUAStmt,
			activityID,
			softwareInstallerID,
			installerDetails.TitleID,
		)
		if err != nil {
			return err
		}

		if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, ""); err != nil {
			return ctxerr.Wrap(ctx, err, "activate next activity")
		}
		return nil
	})

	return ctxerr.Wrap(ctx, err, "inserting new uninstall software request")
}

func (ds *Datastore) GetSoftwareInstallResults(ctx context.Context, resultsUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	query := `
SELECT
	hsi.execution_id AS execution_id,
	hsi.pre_install_query_output,
	hsi.post_install_script_output,
	hsi.install_script_output,
	hsi.host_id AS host_id,
	COALESCE(st.name, hsi.software_title_name) AS software_title,
	hsi.software_title_id,
	COALESCE(hsi.execution_status, '') AS status,
	hsi.installer_filename AS software_package,
	hsi.user_id AS user_id,
	hsi.post_install_script_exit_code,
	hsi.install_script_exit_code,
	hsi.self_service,
	hsi.host_deleted_at,
	hsi.policy_id,
	hsi.created_at as created_at,
	hsi.updated_at as updated_at
FROM
	host_software_installs hsi
	LEFT JOIN software_titles st ON hsi.software_title_id = st.id
WHERE
	hsi.execution_id = :execution_id AND
	hsi.uninstall = 0 AND
	hsi.canceled = 0

UNION

SELECT
	ua.execution_id AS execution_id,
	NULL AS pre_install_query_output,
	NULL AS post_install_script_output,
	NULL AS install_script_output,
	ua.host_id AS host_id,
	COALESCE(st.name, ua.payload->>'$.software_title_name') AS software_title,
	siua.software_title_id,
	'pending_install' AS status,
	ua.payload->>'$.installer_filename' AS software_package,
	ua.user_id AS user_id,
	NULL AS post_install_script_exit_code,
	NULL AS install_script_exit_code,
	ua.payload->'$.self_service' AS self_service,
	NULL AS host_deleted_at,
	siua.policy_id AS policy_id,
	ua.created_at as created_at,
	ua.updated_at as updated_at
FROM
	upcoming_activities ua
	INNER JOIN software_install_upcoming_activities siua
		ON ua.id = siua.upcoming_activity_id
	LEFT JOIN software_titles st
		ON siua.software_title_id = st.id
WHERE
	ua.execution_id = :execution_id AND
	ua.activity_type = 'software_install' AND
	ua.activated_at IS NULL -- if already activated, covered by the other SELECT
`

	stmt, args, err := sqlx.Named(query, map[string]any{
		"execution_id": resultsUUID,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build named query for get software install results")
	}

	var dest fleet.HostSoftwareInstallerResult
	err = sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostSoftwareInstallerResult"), "get host software installer results")
		}
		return nil, ctxerr.Wrap(ctx, err, "get host software installer results")
	}

	return &dest, nil
}

func (ds *Datastore) GetSummaryHostSoftwareInstalls(ctx context.Context, installerID uint) (*fleet.SoftwareInstallerStatusSummary, error) {
	var dest fleet.SoftwareInstallerStatusSummary

	stmt := `WITH

-- select most recent upcoming activities for each host
upcoming AS (
	SELECT
		ua.host_id,
		IF(ua.activity_type = 'software_install', :software_status_pending_install, :software_status_pending_uninstall) AS status
	FROM
		upcoming_activities ua
		JOIN software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
		JOIN hosts h ON host_id = h.id
		LEFT JOIN (
			upcoming_activities ua2
			INNER JOIN software_install_upcoming_activities siua2
				ON ua2.id = siua2.upcoming_activity_id
		) ON ua.host_id = ua2.host_id AND
			siua.software_installer_id = siua2.software_installer_id AND
			ua.activity_type = ua2.activity_type AND
			(ua2.priority < ua.priority OR ua2.created_at > ua.created_at)
	WHERE
		ua.activity_type IN('software_install', 'software_uninstall')
		AND ua2.id IS NULL
		AND siua.software_installer_id = :installer_id
),

-- select most recent past activities for each host
past AS (
	SELECT
		hsi.host_id,
		hsi.status
	FROM
		host_software_installs hsi
		JOIN hosts h ON host_id = h.id
		LEFT JOIN host_software_installs hsi2
			ON hsi.host_id = hsi2.host_id AND
				 hsi.software_installer_id = hsi2.software_installer_id AND
				 hsi2.removed = 0 AND
				 hsi2.canceled = 0 AND
				 hsi2.host_deleted_at IS NULL AND
				 (hsi.created_at < hsi2.created_at OR (hsi.created_at = hsi2.created_at AND hsi.id < hsi2.id))
	WHERE
		hsi2.id IS NULL
		AND hsi.software_installer_id = :installer_id
		AND hsi.host_id NOT IN(SELECT host_id FROM upcoming) -- antijoin to exclude hosts with upcoming activities
		AND hsi.host_deleted_at IS NULL
		AND hsi.removed = 0
		AND hsi.canceled = 0
)

-- count each status
SELECT
	COALESCE(SUM( IF(status = :software_status_pending_install, 1, 0)), 0) AS pending_install,
	COALESCE(SUM( IF(status = :software_status_failed_install, 1, 0)), 0) AS failed_install,
	COALESCE(SUM( IF(status = :software_status_pending_uninstall, 1, 0)), 0) AS pending_uninstall,
	COALESCE(SUM( IF(status = :software_status_failed_uninstall, 1, 0)), 0) AS failed_uninstall,
	COALESCE(SUM( IF(status = :software_status_installed, 1, 0)), 0) AS installed
FROM (

-- union most recent past and upcoming activities after joining to get statuses for most recent activities
SELECT
	past.host_id,
	past.status
FROM past
UNION
SELECT
	upcoming.host_id,
	upcoming.status
FROM upcoming
) t`

	query, args, err := sqlx.Named(stmt, map[string]interface{}{
		"installer_id":                      installerID,
		"software_status_pending_install":   fleet.SoftwareInstallPending,
		"software_status_failed_install":    fleet.SoftwareInstallFailed,
		"software_status_pending_uninstall": fleet.SoftwareUninstallPending,
		"software_status_failed_uninstall":  fleet.SoftwareUninstallFailed,
		"software_status_installed":         fleet.SoftwareInstalled,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get summary host software installs: named query")
	}

	err = sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get summary host software install status")
	}

	return &dest, nil
}

func (ds *Datastore) vppAppJoin(appID fleet.VPPAppID, status fleet.SoftwareInstallerStatus) (string, []interface{}, error) {
	// for pending status, we'll join through upcoming_activities
	if status == fleet.SoftwarePending || status == fleet.SoftwareInstallPending || status == fleet.SoftwareUninstallPending {
		stmt := `JOIN (
SELECT DISTINCT
	host_id
FROM
	upcoming_activities ua
	JOIN vpp_app_upcoming_activities vppua ON ua.id = vppua.upcoming_activity_id
WHERE
	%s) hss ON hss.host_id = h.id`

		filter := "vppua.adam_id = ? AND vppua.platform = ?"
		switch status {
		case fleet.SoftwareInstallPending:
			filter += " AND ua.activity_type = 'vpp_app_install'"
		case fleet.SoftwareUninstallPending:
			// TODO: Update this when VPP supports uninstall, for now we map uninstall to install to preserve existing behavior of VPP filters
			filter += " AND ua.activity_type = 'vpp_app_install'"
		default:
			// no change, we're just filtering by app_id and platform so it will pick up any
			// activity type that is associated with the app (i.e. both install and uninstall)
		}

		return fmt.Sprintf(stmt, filter), []interface{}{appID.AdamID, appID.Platform}, nil
	}

	// TODO: Update this when VPP supports uninstall so that we map for now we map the generic failed status to the install statuses
	if status == fleet.SoftwareFailed {
		status = fleet.SoftwareInstallFailed // TODO: When VPP supports uninstall this should become STATUS IN ('failed_install', 'failed_uninstall')
	}

	// NOTE(mna): the pre-unified queue version of this query did not check for
	// removed = 0, so I am porting the same behavior (there's even a test that
	// fails if I add removed = 0 condition).
	stmt := fmt.Sprintf(`JOIN (
SELECT
	hvsi.host_id
FROM
	host_vpp_software_installs hvsi
	INNER JOIN
		nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
	LEFT JOIN host_vpp_software_installs hvsi2
		ON hvsi.host_id = hvsi2.host_id AND
			 hvsi.adam_id = hvsi2.adam_id AND
			 hvsi.platform = hvsi2.platform AND
			 hvsi2.canceled = 0 AND
			 (hvsi.created_at < hvsi2.created_at OR (hvsi.created_at = hvsi2.created_at AND hvsi.id < hvsi2.id))
WHERE
	hvsi2.id IS NULL
	AND hvsi.adam_id = :adam_id
	AND hvsi.platform = :platform
	AND hvsi.canceled = 0
	AND (%s) = :status
	AND NOT EXISTS (
		SELECT 1
		FROM
			upcoming_activities ua
			JOIN vpp_app_upcoming_activities vaua ON ua.id = vaua.upcoming_activity_id
		WHERE
			ua.host_id = hvsi.host_id
			AND vaua.adam_id = hvsi.adam_id
			AND vaua.platform = hvsi.platform
			AND ua.activity_type = 'vpp_app_install'
	)
) hss ON hss.host_id = h.id
`, vppAppHostStatusNamedQuery("hvsi", "ncr", ""))

	return sqlx.Named(stmt, map[string]interface{}{
		"status":                    status,
		"adam_id":                   appID.AdamID,
		"platform":                  appID.Platform,
		"software_status_installed": fleet.SoftwareInstalled,
		"software_status_failed":    fleet.SoftwareInstallFailed,
		"software_status_pending":   fleet.SoftwareInstallPending,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
	})
}

func (ds *Datastore) softwareInstallerJoin(titleID uint, status fleet.SoftwareInstallerStatus) (string, []interface{}, error) {
	// for pending status, we'll join through upcoming_activities
	if status == fleet.SoftwarePending || status == fleet.SoftwareInstallPending || status == fleet.SoftwareUninstallPending {
		stmt := `JOIN (
SELECT DISTINCT
	host_id
FROM
	upcoming_activities ua
	JOIN software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
WHERE
	%s) hss ON hss.host_id = h.id`

		filter := "siua.software_title_id = ?"
		switch status {
		case fleet.SoftwareInstallPending:
			filter += " AND ua.activity_type = 'software_install'"
		case fleet.SoftwareUninstallPending:
			filter += " AND ua.activity_type = 'software_uninstall'"
		default:
			// no change
		}

		return fmt.Sprintf(stmt, filter), []interface{}{titleID}, nil
	}

	// for non-pending statuses, we'll join through host_software_installs filtered by the status
	statusFilter := "hsi.status = :status"
	if status == fleet.SoftwareFailed {
		// failed is a special case, we must include both install and uninstall failures
		statusFilter = "hsi.status IN (:installFailed, :uninstallFailed)"
	}

	stmt := fmt.Sprintf(`JOIN (
SELECT
	hsi.host_id
FROM
	host_software_installs hsi
	LEFT JOIN host_software_installs hsi2
		ON hsi.host_id = hsi2.host_id AND
			 hsi.software_title_id = hsi2.software_title_id AND
			 hsi2.removed = 0 AND
			 hsi2.canceled = 0 AND
			 (hsi.created_at < hsi2.created_at OR (hsi.created_at = hsi2.created_at AND hsi.id < hsi2.id))
WHERE
	hsi2.id IS NULL
	AND hsi.software_title_id = :title_id
	AND hsi.removed = 0
	AND hsi.canceled = 0
	AND %s
	AND NOT EXISTS (
		SELECT 1
		FROM
			upcoming_activities ua
			JOIN software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
		WHERE
			ua.host_id = hsi.host_id
			AND siua.software_title_id = hsi.software_title_id
			AND ua.activity_type = 'software_install'
	)
) hss ON hss.host_id = h.id
`, statusFilter)

	return sqlx.Named(stmt, map[string]interface{}{
		"status":          status,
		"installFailed":   fleet.SoftwareInstallFailed,
		"uninstallFailed": fleet.SoftwareUninstallFailed,
		"title_id":        titleID,
	})
}

func (ds *Datastore) GetHostLastInstallData(ctx context.Context, hostID, installerID uint) (*fleet.HostLastInstallData, error) {
	hostLastInstall, err := ds.getLatestUpcomingInstall(ctx, hostID, installerID)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		hostLastInstall, err = ds.getLatestPastInstall(ctx, hostID, installerID)
	}

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return hostLastInstall, err
}

func (ds *Datastore) getLatestUpcomingInstall(ctx context.Context, hostID, installerID uint) (*fleet.HostLastInstallData, error) {
	var hostLastInstall fleet.HostLastInstallData
	stmt := `
SELECT
	execution_id,
	'pending_install' AS status
FROM
	upcoming_activities
WHERE
	id = (
		SELECT
			MAX(ua.id)
		FROM
			upcoming_activities ua
		JOIN
			software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
		WHERE
			ua.activity_type = 'software_install' AND ua.host_id = ? AND siua.software_installer_id = ?)`

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hostLastInstall, stmt, hostID, installerID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get latest upcoming install")
	}

	return &hostLastInstall, nil
}

func (ds *Datastore) getLatestPastInstall(ctx context.Context, hostID, installerID uint) (*fleet.HostLastInstallData, error) {
	var hostLastInstall fleet.HostLastInstallData
	stmt := `
SELECT
	execution_id,
	status
FROM
	host_software_installs
WHERE
	id = (
		SELECT
			MAX(hsi.id)
		FROM
			host_software_installs hsi
		WHERE
			hsi.host_id = ? AND hsi.software_installer_id = ? AND hsi.canceled = 0)`

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hostLastInstall, stmt, hostID, installerID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get latest past install")
	}

	return &hostLastInstall, nil
}

func (ds *Datastore) CleanupUnusedSoftwareInstallers(ctx context.Context, softwareInstallStore fleet.SoftwareInstallerStore, removeCreatedBefore time.Time) error {
	if softwareInstallStore == nil {
		// no-op in this case, possible if not running with a Premium license
		return nil
	}

	// get the list of software installers hashes that are in use
	var storageIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &storageIDs, `SELECT DISTINCT storage_id FROM software_installers`); err != nil {
		return ctxerr.Wrap(ctx, err, "get list of software installers in use")
	}

	_, err := softwareInstallStore.Cleanup(ctx, storageIDs, removeCreatedBefore)
	return ctxerr.Wrap(ctx, err, "cleanup unused software installers")
}

func (ds *Datastore) BatchSetSoftwareInstallers(ctx context.Context, tmID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
	const upsertSoftwareTitles = `
INSERT INTO software_titles
  (name, source, browser, bundle_identifier)
VALUES
  %s
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  source = VALUES(source),
  browser = VALUES(browser),
  bundle_identifier = VALUES(bundle_identifier)
`

	const loadSoftwareTitles = `
SELECT
  id
FROM
  software_titles
WHERE (unique_identifier, source, browser) IN (%s)
`

	const unsetAllInstallersFromPolicies = `
UPDATE
  policies
SET
  software_installer_id = NULL
WHERE
  team_id = ?
`

	const deleteAllPendingUninstallScriptExecutions = `
		DELETE FROM host_script_results WHERE execution_id IN (
			SELECT execution_id FROM host_software_installs WHERE status = 'pending_uninstall'
			AND software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ?
			)
		)
`

	const cancelSetupExperienceStatusForAllDeletedPendingSoftwareInstalls = `
UPDATE setup_experience_status_results SET status=? WHERE status IN (?, ?) AND host_software_installs_execution_id IN (
	  SELECT execution_id FROM host_software_installs hsi INNER JOIN software_installers si ON hsi.software_installer_id=si.id
	  WHERE hsi.status IN ('pending_install', 'pending_uninstall') AND si.global_or_team_id = ?
	UNION
	  SELECT ua.execution_id FROM upcoming_activities ua INNER JOIN software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
	  INNER JOIN software_installers si ON siua.software_installer_id=si.id
	  WHERE ua.activity_type IN ('software_install', 'software_uninstall') AND si.global_or_team_id = ?
)
`

	const deleteAllPendingSoftwareInstallsHSI = `
		DELETE FROM host_software_installs
		WHERE status IN('pending_install', 'pending_uninstall')
		AND software_installer_id IN (
			SELECT id FROM software_installers WHERE global_or_team_id = ?
		)
`

	const loadAffectedHostsPendingSoftwareInstallsUA = `
		SELECT
			DISTINCT host_id
		FROM
			upcoming_activities ua
		INNER JOIN software_install_upcoming_activities siua
			ON ua.id = siua.upcoming_activity_id
		WHERE
			ua.activity_type IN ('software_install', 'software_uninstall') AND
			ua.activated_at IS NOT NULL AND
			siua.software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ?
		)
`

	const deleteAllPendingSoftwareInstallsUA = `
		DELETE FROM upcoming_activities
		USING upcoming_activities
		INNER JOIN software_install_upcoming_activities siua
			ON upcoming_activities.id = siua.upcoming_activity_id
		WHERE
			activity_type IN ('software_install', 'software_uninstall') AND
			siua.software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ?
		)
`
	const markAllSoftwareInstallsAsRemoved = `
		UPDATE host_software_installs SET removed = TRUE
		WHERE status IS NOT NULL AND host_deleted_at IS NULL
		AND software_installer_id IN (
			SELECT id FROM software_installers WHERE global_or_team_id = ?
		)
`

	const deleteAllInstallersInTeam = `
DELETE FROM
  software_installers
WHERE
  global_or_team_id = ?
`

	const deletePendingUninstallScriptExecutionsNotInList = `
		DELETE FROM host_script_results WHERE execution_id IN (
			SELECT execution_id FROM host_software_installs WHERE status = 'pending_uninstall'
			AND software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id NOT IN (?)
			)
		)
`

	const cancelSetupExperienceStatusForDeletedSoftwareInstalls = `
		UPDATE setup_experience_status_results SET status=? WHERE status IN (?, ?) AND host_software_installs_execution_id IN (
			  SELECT execution_id FROM host_software_installs hsi INNER JOIN software_installers si ON hsi.software_installer_id=si.id
			  WHERE hsi.status IN ('pending_install', 'pending_uninstall') AND si.global_or_team_id = ? AND si.title_id NOT IN (?)
			UNION
			  SELECT ua.execution_id FROM upcoming_activities ua INNER JOIN software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
			  INNER JOIN software_installers si ON siua.software_installer_id=si.id
			  WHERE ua.activity_type IN ('software_install', 'software_uninstall') AND si.global_or_team_id = ? AND si.title_id NOT IN (?)
		)
	`

	const deletePendingSoftwareInstallsNotInListHSI = `
		DELETE FROM host_software_installs
		WHERE status IN('pending_install', 'pending_uninstall')
		AND software_installer_id IN (
			SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id NOT IN (?)
		)
`

	const loadAffectedHostsPendingSoftwareInstallsNotInListUA = `
		SELECT
			DISTINCT host_id
		FROM
			upcoming_activities ua
		INNER JOIN software_install_upcoming_activities siua
			ON ua.id = siua.upcoming_activity_id
		WHERE
			ua.activity_type IN ('software_install', 'software_uninstall') AND
			ua.activated_at IS NOT NULL AND
			siua.software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id NOT IN (?)
			)
`

	const deletePendingSoftwareInstallsNotInListUA = `
		DELETE FROM upcoming_activities
		USING upcoming_activities
		INNER JOIN software_install_upcoming_activities siua
			ON upcoming_activities.id = siua.upcoming_activity_id
		WHERE
			activity_type IN ('software_install', 'software_uninstall') AND
			siua.software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id NOT IN (?)
			)
`
	const markSoftwareInstallsNotInListAsRemoved = `
		UPDATE host_software_installs SET removed = TRUE
			WHERE status IS NOT NULL AND host_deleted_at IS NULL
				AND software_installer_id IN (
					SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id NOT IN (?)
			   )
`

	const unsetInstallersNotInListFromPolicies = `
UPDATE
  policies
SET
  software_installer_id = NULL
WHERE
  software_installer_id IN (
    SELECT id FROM software_installers
    WHERE global_or_team_id = ? AND
    title_id NOT IN (?)
  )
`

	const countInstallDuringSetupNotInList = `
SELECT
  COUNT(*)
FROM
  software_installers
WHERE
  global_or_team_id = ? AND
  title_id NOT IN (?) AND
  install_during_setup = 1
`

	const deleteInstallersNotInList = `
DELETE FROM
  software_installers
WHERE
  global_or_team_id = ? AND
  title_id NOT IN (?)
`

	const checkExistingInstaller = `
SELECT id,
storage_id != ? is_package_modified,
install_script_content_id != ? OR uninstall_script_content_id != ? OR pre_install_query != ? OR
COALESCE(post_install_script_content_id != ? OR
	(post_install_script_content_id IS NULL AND ? IS NOT NULL) OR
	(? IS NULL AND post_install_script_content_id IS NOT NULL)
, FALSE) is_metadata_modified FROM software_installers
WHERE global_or_team_id = ?	AND title_id IN (SELECT id FROM software_titles WHERE unique_identifier = ? AND source = ? AND browser = '')
`

	const insertNewOrEditedInstaller = `
INSERT INTO software_installers (
	team_id,
	global_or_team_id,
	storage_id,
	filename,
	extension,
	version,
	install_script_content_id,
	uninstall_script_content_id,
	pre_install_query,
	post_install_script_content_id,
	platform,
	self_service,
	upgrade_code,
	title_id,
	user_id,
	user_name,
	user_email,
	url,
	package_ids,
	install_during_setup,
	fleet_maintained_app_id
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
  (SELECT id FROM software_titles WHERE unique_identifier = ? AND source = ? AND browser = ''),
  ?, (SELECT name FROM users WHERE id = ?), (SELECT email FROM users WHERE id = ?), ?, ?, COALESCE(?, false), ?
)
ON DUPLICATE KEY UPDATE
  install_script_content_id = VALUES(install_script_content_id),
  uninstall_script_content_id = VALUES(uninstall_script_content_id),
  post_install_script_content_id = VALUES(post_install_script_content_id),
  storage_id = VALUES(storage_id),
  filename = VALUES(filename),
  extension = VALUES(extension),
  version = VALUES(version),
  pre_install_query = VALUES(pre_install_query),
  platform = VALUES(platform),
  self_service = VALUES(self_service),
  upgrade_code = VALUES(upgrade_code),
  user_id = VALUES(user_id),
  user_name = VALUES(user_name),
  user_email = VALUES(user_email),
  url = VALUES(url),
  install_during_setup = COALESCE(?, install_during_setup)
`

	const loadSoftwareInstallerID = `
SELECT
	id
FROM
	software_installers
WHERE
	global_or_team_id = ?	AND
	-- this is guaranteed to select a single title_id, due to unique index
	title_id IN (SELECT id FROM software_titles WHERE unique_identifier = ? AND source = ? AND browser = '')
`

	const deleteInstallerLabelsNotInList = `
DELETE FROM
	software_installer_labels
WHERE
	software_installer_id = ? AND
	label_id NOT IN (?)
`

	const deleteAllInstallerLabels = `
DELETE FROM
	software_installer_labels
WHERE
	software_installer_id = ?
`

	const upsertInstallerLabels = `
INSERT INTO
	software_installer_labels (
		software_installer_id,
		label_id,
		exclude
	)
VALUES
	%s
ON DUPLICATE KEY UPDATE
	exclude = VALUES(exclude)
`

	const loadExistingInstallerLabels = `
SELECT
	label_id,
	exclude
FROM
	software_installer_labels
WHERE
	software_installer_id = ?
`

	const deleteAllInstallerCategories = `
DELETE FROM
	software_installer_software_categories
WHERE
	software_installer_id = ?
`

	const deleteInstallerCategoriesNotInList = `
DELETE FROM
	software_installer_software_categories
WHERE
	software_installer_id = ? AND
	software_category_id NOT IN (?)
`

	const upsertInstallerCategories = `
INSERT IGNORE INTO
	software_installer_software_categories (
		software_installer_id,
		software_category_id
	)
VALUES
	%s
`

	// use a team id of 0 if no-team
	var globalOrTeamID uint
	if tmID != nil {
		globalOrTeamID = *tmID
	}

	// if we're batch-setting installers and replacing the ones installed during
	// setup in the same go, no need to validate that we don't delete one marked
	// as install during setup (since we're overwriting those). This is always
	// called from fleetctl gitops, so it should always be the case anyway.
	var replacingInstallDuringSetup bool
	if len(installers) == 0 || installers[0].InstallDuringSetup != nil {
		replacingInstallDuringSetup = true
	}

	var activateAffectedHostIDs []uint

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// if no installers are provided, just delete whatever was in
		// the table
		if len(installers) == 0 {
			if _, err := tx.ExecContext(ctx, unsetAllInstallersFromPolicies, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "unset all obsolete installers in policies")
			}

			if _, err := tx.ExecContext(ctx, deleteAllPendingUninstallScriptExecutions, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete all pending uninstall script executions")
			}

			if _, err := tx.ExecContext(ctx, cancelSetupExperienceStatusForAllDeletedPendingSoftwareInstalls, fleet.SetupExperienceStatusCancelled, fleet.SetupExperienceStatusPending, fleet.SetupExperienceStatusRunning,
				globalOrTeamID, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "cancel pending setup experience software installs")
			}

			if _, err := tx.ExecContext(ctx, deleteAllPendingSoftwareInstallsHSI, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete all pending host software install records")
			}

			var affectedHostIDs []uint
			if err := sqlx.SelectContext(ctx, tx, &affectedHostIDs,
				loadAffectedHostsPendingSoftwareInstallsUA, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "load affected hosts for upcoming software installs")
			}
			activateAffectedHostIDs = affectedHostIDs

			if _, err := tx.ExecContext(ctx, deleteAllPendingSoftwareInstallsUA, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete all upcoming pending host software install records")
			}

			if _, err := tx.ExecContext(ctx, markAllSoftwareInstallsAsRemoved, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "mark all host software installs as removed")
			}

			if _, err := tx.ExecContext(ctx, deleteAllInstallersInTeam, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete obsolete software installers")
			}

			return nil
		}

		var args []any
		for _, installer := range installers {
			args = append(
				args,
				installer.Title,
				installer.Source,
				"",
				func() *string {
					if strings.TrimSpace(installer.BundleIdentifier) != "" {
						return &installer.BundleIdentifier
					}
					return nil
				}(),
			)
		}

		values := strings.TrimSuffix(
			strings.Repeat("(?,?,?,?),", len(installers)),
			",",
		)
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(upsertSoftwareTitles, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert new/edited software title")
		}

		var titleIDs []uint
		args = []any{}
		for _, installer := range installers {
			args = append(
				args,
				BundleIdentifierOrName(installer.BundleIdentifier, installer.Title),
				installer.Source,
				"",
			)
		}
		values = strings.TrimSuffix(
			strings.Repeat("(?,?,?),", len(installers)),
			",",
		)

		if err := sqlx.SelectContext(ctx, tx, &titleIDs, fmt.Sprintf(loadSoftwareTitles, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "load existing titles")
		}

		stmt, args, err := sqlx.In(unsetInstallersNotInListFromPolicies, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to unset obsolete installers from policies")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "unset obsolete software installers from policies")
		}

		// check if any in the list are install_during_setup, fail if there is one
		if !replacingInstallDuringSetup {
			stmt, args, err = sqlx.In(countInstallDuringSetupNotInList, globalOrTeamID, titleIDs)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build statement to check installers install_during_setup")
			}
			var countInstallDuringSetup int
			if err := sqlx.GetContext(ctx, tx, &countInstallDuringSetup, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "check installers installed during setup")
			}
			if countInstallDuringSetup > 0 {
				return errDeleteInstallerInstalledDuringSetup
			}
		}

		stmt, args, err = sqlx.In(deletePendingUninstallScriptExecutionsNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete pending uninstall script executions")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete pending uninstall script executions")
		}

		stmt, args, err = sqlx.In(cancelSetupExperienceStatusForDeletedSoftwareInstalls, fleet.SetupExperienceStatusCancelled, fleet.SetupExperienceStatusPending, fleet.SetupExperienceStatusRunning,
			globalOrTeamID, titleIDs, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to cancel pending setup experience software installs")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "cancel pending setup experience software installs for obsolete host software install records")
		}

		stmt, args, err = sqlx.In(deletePendingSoftwareInstallsNotInListHSI, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete pending software installs")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete pending host software install records")
		}

		stmt, args, err = sqlx.In(loadAffectedHostsPendingSoftwareInstallsNotInListUA, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to load affected hosts for upcoming software installs")
		}
		var affectedHostIDs []uint
		if err := sqlx.SelectContext(ctx, tx, &affectedHostIDs, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "load affected hosts for upcoming software installs")
		}
		activateAffectedHostIDs = affectedHostIDs

		stmt, args, err = sqlx.In(deletePendingSoftwareInstallsNotInListUA, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete upcoming pending software installs")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete upcoming pending host software install records")
		}

		stmt, args, err = sqlx.In(markSoftwareInstallsNotInListAsRemoved, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to mark obsolete host software installs as removed")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "mark obsolete host software installs as removed")
		}

		stmt, args, err = sqlx.In(deleteInstallersNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete obsolete installers")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete software installers")
		}

		for _, installer := range installers {
			if installer.ValidatedLabels == nil {
				return ctxerr.Errorf(ctx, "labels have not been validated for installer with name %s", installer.Filename)
			}

			isRes, err := insertScriptContents(ctx, tx, installer.InstallScript)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "inserting install script contents for software installer with name %q", installer.Filename)
			}
			installScriptID, _ := isRes.LastInsertId()

			uisRes, err := insertScriptContents(ctx, tx, installer.UninstallScript)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "inserting uninstall script contents for software installer with name %q", installer.Filename)
			}
			uninstallScriptID, _ := uisRes.LastInsertId()

			var postInstallScriptID *int64
			if installer.PostInstallScript != "" {
				pisRes, err := insertScriptContents(ctx, tx, installer.PostInstallScript)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "inserting post-install script contents for software installer with name %q", installer.Filename)
				}

				insertID, _ := pisRes.LastInsertId()
				postInstallScriptID = &insertID
			}

			wasUpdatedArgs := []interface{}{
				// package update
				installer.StorageID,
				// metadata update
				installScriptID,
				uninstallScriptID,
				installer.PreInstallQuery,
				postInstallScriptID,
				postInstallScriptID,
				postInstallScriptID,
				// WHERE clause
				globalOrTeamID,
				BundleIdentifierOrName(installer.BundleIdentifier, installer.Title),
				installer.Source,
			}

			// pull existing installer state if it exists so we can diff for side effects post-update
			type existingInstallerUpdateCheckResult struct {
				InstallerID        uint `db:"id"`
				IsPackageModified  bool `db:"is_package_modified"`
				IsMetadataModified bool `db:"is_metadata_modified"`
			}
			var existing []existingInstallerUpdateCheckResult
			err = sqlx.SelectContext(ctx, tx, &existing, checkExistingInstaller, wasUpdatedArgs...)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return ctxerr.Wrapf(ctx, err, "checking for existing installer with name %q", installer.Filename)
				}
			}

			args := []interface{}{
				tmID,
				globalOrTeamID,
				installer.StorageID,
				installer.Filename,
				installer.Extension,
				installer.Version,
				installScriptID,
				uninstallScriptID,
				installer.PreInstallQuery,
				postInstallScriptID,
				installer.Platform,
				installer.SelfService,
				installer.UpgradeCode,
				BundleIdentifierOrName(installer.BundleIdentifier, installer.Title),
				installer.Source,
				installer.UserID,
				installer.UserID,
				installer.UserID,
				installer.URL,
				strings.Join(installer.PackageIDs, ","),
				installer.InstallDuringSetup,
				installer.FleetMaintainedAppID,
				installer.InstallDuringSetup,
			}
			upsertQuery := insertNewOrEditedInstaller
			if len(existing) > 0 && existing[0].IsPackageModified { // update uploaded_at for updated installer package
				upsertQuery = fmt.Sprintf("%s, uploaded_at = NOW()", upsertQuery)
			}

			if _, err := tx.ExecContext(ctx, upsertQuery, args...); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert new/edited installer with name %q", installer.Filename)
			}

			// now that the software installer is created/updated, load its installer
			// ID (cannot use res.LastInsertID due to the upsert statement, won't
			// give the id in case of update)
			var installerID uint
			if err := sqlx.GetContext(ctx, tx, &installerID, loadSoftwareInstallerID, globalOrTeamID, BundleIdentifierOrName(installer.BundleIdentifier, installer.Title), installer.Source); err != nil {
				return ctxerr.Wrapf(ctx, err, "load id of new/edited installer with name %q", installer.Filename)
			}

			// process the labels associated with that software installer
			if len(installer.ValidatedLabels.ByName) == 0 {
				// no label to apply, so just delete all existing labels if any
				res, err := tx.ExecContext(ctx, deleteAllInstallerLabels, installerID)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "delete installer labels for %s", installer.Filename)
				}

				if n, _ := res.RowsAffected(); n > 0 && len(existing) > 0 {
					// if it did delete a row, then the target changed so pending
					// installs/uninstalls must be deleted
					existing[0].IsMetadataModified = true
				}
			} else {
				// there are new labels to apply, delete only the obsolete ones
				labelIDs := make([]uint, 0, len(installer.ValidatedLabels.ByName))
				for _, lbl := range installer.ValidatedLabels.ByName {
					labelIDs = append(labelIDs, lbl.LabelID)
				}
				stmt, args, err := sqlx.In(deleteInstallerLabelsNotInList, installerID, labelIDs)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "build statement to delete installer labels not in list")
				}

				res, err := tx.ExecContext(ctx, stmt, args...)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "delete installer labels not in list for %s", installer.Filename)
				}
				if n, _ := res.RowsAffected(); n > 0 && len(existing) > 0 {
					// if it did delete a row, then the target changed so pending
					// installs/uninstalls must be deleted
					existing[0].IsMetadataModified = true
				}

				excludeLabels := installer.ValidatedLabels.LabelScope == fleet.LabelScopeExcludeAny
				if len(existing) > 0 && !existing[0].IsMetadataModified {
					// load the remaining labels for that installer, so that we can detect
					// if any label changed (if the counts differ, then labels did change,
					// otherwise if the exclude bool changed, the target did change).
					var existingLabels []struct {
						LabelID uint `db:"label_id"`
						Exclude bool `db:"exclude"`
					}
					if err := sqlx.SelectContext(ctx, tx, &existingLabels, loadExistingInstallerLabels, installerID); err != nil {
						return ctxerr.Wrapf(ctx, err, "load existing labels for installer with name %q", installer.Filename)
					}

					if len(existingLabels) != len(labelIDs) {
						existing[0].IsMetadataModified = true
					}
					if len(existingLabels) > 0 && existingLabels[0].Exclude != excludeLabels {
						// same labels are provided, but the include <-> exclude changed
						existing[0].IsMetadataModified = true
					}
				}

				// upsert the new labels now that obsolete ones have been deleted
				var upsertLabelArgs []any
				for _, lblID := range labelIDs {
					upsertLabelArgs = append(upsertLabelArgs, installerID, lblID, excludeLabels)
				}
				upsertLabelValues := strings.TrimSuffix(strings.Repeat("(?,?,?),", len(installer.ValidatedLabels.ByName)), ",")

				_, err = tx.ExecContext(ctx, fmt.Sprintf(upsertInstallerLabels, upsertLabelValues), upsertLabelArgs...)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "insert new/edited labels for installer with name %q", installer.Filename)
				}
			}

			if len(installer.CategoryIDs) == 0 {
				// delete all categories if there are any
				_, err := tx.ExecContext(ctx, deleteAllInstallerCategories, installerID)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "delete installer categories for %s", installer.Filename)
				}
			} else {
				// there are new categories to apply, delete only the obsolete ones
				stmt, args, err := sqlx.In(deleteInstallerCategoriesNotInList, installerID, installer.CategoryIDs)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "build statement to delete installer categories not in list")
				}

				_, err = tx.ExecContext(ctx, stmt, args...)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "delete installer categories not in list for %s", installer.Filename)
				}

				var upsertCategoriesArgs []any
				for _, catID := range installer.CategoryIDs {
					upsertCategoriesArgs = append(upsertCategoriesArgs, installerID, catID)
				}
				upsertCategoriesValues := strings.TrimSuffix(strings.Repeat("(?,?),", len(installer.CategoryIDs)), ",")
				_, err = tx.ExecContext(ctx, fmt.Sprintf(upsertInstallerCategories, upsertCategoriesValues), upsertCategoriesArgs...)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "insert new/edited categories for installer with name %q", installer.Filename)
				}
			}

			// perform side effects if this was an update (related to pending (un)install requests)
			if len(existing) > 0 {
				affectedHostIDs, err := ds.runInstallerUpdateSideEffectsInTransaction(
					ctx,
					tx,
					existing[0].InstallerID,
					existing[0].IsMetadataModified,
					existing[0].IsPackageModified,
				)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "processing installer with name %q", installer.Filename)
				}
				activateAffectedHostIDs = append(activateAffectedHostIDs, affectedHostIDs...)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	return ds.activateNextUpcomingActivityForBatchOfHosts(ctx, activateAffectedHostIDs)
}

func (ds *Datastore) HasSelfServiceSoftwareInstallers(ctx context.Context, hostPlatform string, hostTeamID *uint) (bool, error) {
	if fleet.IsLinux(hostPlatform) {
		hostPlatform = "linux"
	}
	stmt := `SELECT 1
		WHERE EXISTS (
			SELECT 1
			FROM software_installers
			WHERE self_service = 1 AND platform = ? AND global_or_team_id = ?
		) OR EXISTS (
			SELECT 1
			FROM vpp_apps_teams
			WHERE self_service = 1 AND platform = ? AND global_or_team_id = ?
		)`
	var globalOrTeamID uint
	if hostTeamID != nil {
		globalOrTeamID = *hostTeamID
	}
	args := []interface{}{hostPlatform, globalOrTeamID, hostPlatform, globalOrTeamID}
	var hasInstallers bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hasInstallers, stmt, args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, ctxerr.Wrap(ctx, err, "check for self-service software installers")
	}
	return hasInstallers, nil
}

func (ds *Datastore) GetDetailsForUninstallFromExecutionID(ctx context.Context, executionID string) (string, bool, error) {
	stmt := `
	SELECT COALESCE(st.name, hsi.software_title_name) name, hsi.self_service
	FROM software_titles st
	INNER JOIN software_installers si ON si.title_id = st.id
	INNER JOIN host_software_installs hsi ON hsi.software_installer_id = si.id
	WHERE hsi.execution_id = ? AND hsi.uninstall = TRUE

	UNION

	SELECT st.name, COALESCE(ua.payload->'$.self_service', FALSE) self_service
	FROM
		software_titles st
		INNER JOIN software_installers si ON si.title_id = st.id
		INNER JOIN software_install_upcoming_activities siua
			ON siua.software_installer_id = si.id
		INNER JOIN upcoming_activities ua ON ua.id = siua.upcoming_activity_id
	WHERE
		ua.execution_id = ? AND
		ua.activity_type = 'software_uninstall'
	`
	var result struct {
		Name        string `db:"name"`
		SelfService bool   `db:"self_service"`
	}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &result, stmt, executionID, executionID)
	if err != nil {
		return "", false, ctxerr.Wrap(ctx, err, "get software details for uninstall activity from execution ID")
	}
	return result.Name, result.SelfService, nil
}

func (ds *Datastore) GetSoftwareInstallersPendingUninstallScriptPopulation(ctx context.Context) (map[uint]string, error) {
	query := `SELECT id, storage_id FROM software_installers WHERE package_ids = ''
                                                 AND extension NOT IN ('exe', 'tar.gz', 'dmg', 'zip')`
	type result struct {
		ID        uint   `db:"id"`
		StorageID string `db:"storage_id"`
	}

	var results []result
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installers without package ID")
	}
	if len(results) == 0 {
		return nil, nil
	}
	idMap := make(map[uint]string, len(results))
	for _, r := range results {
		idMap[r.ID] = r.StorageID
	}
	return idMap, nil
}

func (ds *Datastore) GetMSIInstallersWithoutUpgradeCode(ctx context.Context) (map[uint]string, error) {
	query := `SELECT id, storage_id FROM software_installers WHERE extension = 'msi' AND upgrade_code = ''`
	type result struct {
		ID        uint   `db:"id"`
		StorageID string `db:"storage_id"`
	}

	var results []result
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get MSI installers without upgrade code")
	}
	if len(results) == 0 {
		return nil, nil
	}
	idMap := make(map[uint]string, len(results))
	for _, r := range results {
		idMap[r.ID] = r.StorageID
	}
	return idMap, nil
}

func (ds *Datastore) UpdateInstallerUpgradeCode(ctx context.Context, id uint, upgradeCode string) error {
	query := `UPDATE software_installers SET upgrade_code = ? WHERE id = ?`
	_, err := ds.writer(ctx).ExecContext(ctx, query, upgradeCode, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update software installer upgrade code")
	}
	return nil
}

func (ds *Datastore) UpdateSoftwareInstallerWithoutPackageIDs(ctx context.Context, id uint,
	payload fleet.UploadSoftwareInstallerPayload,
) error {
	uninstallScriptID, err := ds.getOrGenerateScriptContentsID(ctx, payload.UninstallScript)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get or generate uninstall script contents ID")
	}
	query := `
		UPDATE software_installers
		SET package_ids = ?, uninstall_script_content_id = ?, extension = ?
		WHERE id = ?
	`
	_, err = ds.writer(ctx).ExecContext(ctx, query, strings.Join(payload.PackageIDs, ","), uninstallScriptID, payload.Extension, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update software installer without package ID")
	}
	return nil
}

func (ds *Datastore) GetSoftwareInstallers(ctx context.Context, teamID uint) ([]fleet.SoftwarePackageResponse, error) {
	const loadInsertedSoftwareInstallers = `
SELECT
  team_id,
  title_id,
  url,
  storage_id as hash_sha256,
  fleet_maintained_app_id
FROM
  software_installers
WHERE global_or_team_id = ?
`
	var softwarePackages []fleet.SoftwarePackageResponse
	// Using ds.writer(ctx) on purpose because this method is to be called after applying software.
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &softwarePackages, loadInsertedSoftwareInstallers, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installers")
	}
	return softwarePackages, nil
}

func (ds *Datastore) IsSoftwareInstallerLabelScoped(ctx context.Context, installerID, hostID uint) (bool, error) {
	return ds.isSoftwareLabelScoped(ctx, installerID, hostID, softwareTypeInstaller)
}

func (ds *Datastore) IsVPPAppLabelScoped(ctx context.Context, vppAppTeamID, hostID uint) (bool, error) {
	return ds.isSoftwareLabelScoped(ctx, vppAppTeamID, hostID, softwareTypeVPP)
}

func (ds *Datastore) isSoftwareLabelScoped(ctx context.Context, softwareID, hostID uint, swType softwareType) (bool, error) {
	stmt := `
		SELECT 1 FROM (

			-- no labels
			SELECT 0 AS count_installer_labels, 0 AS count_host_labels, 0 as count_host_updated_after_labels
			WHERE NOT EXISTS (
				SELECT 1 FROM %[1]s_labels sil WHERE sil.%[1]s_id = :software_id
			)

			UNION

			-- include any
			SELECT
				COUNT(*) AS count_installer_labels,
				COUNT(lm.label_id) AS count_host_labels,
				0 as count_host_updated_after_labels
			FROM
				%[1]s_labels sil
				LEFT OUTER JOIN label_membership lm ON lm.label_id = sil.label_id
				AND lm.host_id = :host_id
			WHERE
				sil.%[1]s_id = :software_id
				AND sil.exclude = 0
			HAVING
				count_installer_labels > 0 AND count_host_labels > 0

			UNION

			-- exclude any, ignore software that depends on labels created
			-- _after_ the label_updated_at timestamp of the host (because
			-- we don't have results for that label yet, the host may or may
			-- not be a member).
			SELECT
				COUNT(*) AS count_installer_labels,
				COUNT(lm.label_id) AS count_host_labels,
				SUM(CASE
				WHEN
					lbl.created_at IS NOT NULL AND lbl.label_membership_type = 0 AND (SELECT label_updated_at FROM hosts WHERE id = :host_id) >= lbl.created_at THEN 1
				WHEN
					lbl.created_at IS NOT NULL AND lbl.label_membership_type = 1 THEN 1
				ELSE
					0
				END) as count_host_updated_after_labels
			FROM
				%[1]s_labels sil
				LEFT OUTER JOIN labels lbl
					ON lbl.id = sil.label_id
				LEFT OUTER JOIN label_membership lm
					ON lm.label_id = sil.label_id AND lm.host_id = :host_id
			WHERE
				sil.%[1]s_id = :software_id
				AND sil.exclude = 1
			HAVING
				count_installer_labels > 0 AND count_installer_labels = count_host_updated_after_labels AND count_host_labels = 0
			) t
	`

	stmt = fmt.Sprintf(stmt, swType)
	namedArgs := map[string]any{
		"host_id":     hostID,
		"software_id": softwareID,
	}
	stmt, args, err := sqlx.Named(stmt, namedArgs)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "build named query for is software label scoped")
	}

	var res bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ctxerr.Wrap(ctx, err, "is software label scoped")
	}

	return res, nil
}

const labelScopedFilter = `
SELECT
	1
FROM (
		-- no labels
		SELECT
			0 AS count_installer_labels,
			0 AS count_host_labels,
			0 AS count_host_updated_after_labels
		WHERE NOT EXISTS ( SELECT 1 FROM %[1]s_labels sil WHERE sil.%[1]s_id = ?)

		UNION

		-- include any
		SELECT
			COUNT(*) AS count_installer_labels,
			COUNT(lm.label_id) AS count_host_labels,
			0 AS count_host_updated_after_labels
		FROM
			%[1]s_labels sil
		LEFT OUTER JOIN label_membership lm ON lm.label_id = sil.label_id
		AND lm.host_id = h.id
		WHERE
			sil.%[1]s_id = ?
			AND sil.exclude = 0
		HAVING
			count_installer_labels > 0
			AND count_host_labels > 0

		UNION

		-- exclude any, ignore software that depends on labels created
		-- _after_ the label_updated_at timestamp of the host (because
		-- we don't have results for that label yet, the host may or may
		-- not be a member).
		SELECT
			COUNT(*) AS count_installer_labels,
			COUNT(lm.label_id) AS count_host_labels,
			SUM(
				CASE
				WHEN lbl.created_at IS NOT NULL AND lbl.label_membership_type = 0 AND (SELECT label_updated_at FROM hosts WHERE id = h.id) >= lbl.created_at THEN 1
				WHEN lbl.created_at IS NOT NULL AND lbl.label_membership_type = 1 THEN 1
				ELSE 0 END) AS count_host_updated_after_labels
		FROM
			%[1]s_labels sil
		LEFT OUTER JOIN labels lbl ON lbl.id = sil.label_id
		LEFT OUTER JOIN label_membership lm ON lm.label_id = sil.label_id AND lm.host_id = h.id
WHERE
	sil.%[1]s_id = ?
	AND sil.exclude = 1
HAVING
	count_installer_labels > 0
	AND count_installer_labels = count_host_updated_after_labels
	AND count_host_labels = 0) t`

func (ds *Datastore) GetIncludedHostIDMapForSoftwareInstaller(ctx context.Context, installerID uint) (map[uint]struct{}, error) {
	return ds.getIncludedHostIDMapForSoftware(ctx, ds.writer(ctx), installerID, softwareTypeInstaller)
}

func (ds *Datastore) getIncludedHostIDMapForSoftware(ctx context.Context, tx sqlx.ExtContext, softwareID uint, swType softwareType) (map[uint]struct{}, error) {
	filter := fmt.Sprintf(labelScopedFilter, swType)
	stmt := fmt.Sprintf(`SELECT
	h.id
FROM
	hosts h
WHERE
	EXISTS (%s)
`, filter)

	var hostIDs []uint
	if err := sqlx.SelectContext(ctx, tx, &hostIDs, stmt, softwareID, softwareID, softwareID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing hosts included in software scope")
	}

	res := make(map[uint]struct{}, len(hostIDs))
	for _, id := range hostIDs {
		res[id] = struct{}{}
	}

	return res, nil
}

func (ds *Datastore) GetExcludedHostIDMapForSoftwareInstaller(ctx context.Context, installerID uint) (map[uint]struct{}, error) {
	return ds.getExcludedHostIDMapForSoftware(ctx, installerID, softwareTypeInstaller)
}

func (ds *Datastore) getExcludedHostIDMapForSoftware(ctx context.Context, softwareID uint, swType softwareType) (map[uint]struct{}, error) {
	filter := fmt.Sprintf(labelScopedFilter, swType)
	stmt := fmt.Sprintf(`SELECT
	h.id
FROM
	hosts h
WHERE
	NOT EXISTS (%s)
`, filter)

	var hostIDs []uint
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &hostIDs, stmt, softwareID, softwareID, softwareID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing hosts excluded from software scope")
	}

	res := make(map[uint]struct{}, len(hostIDs))
	for _, id := range hostIDs {
		res[id] = struct{}{}
	}

	return res, nil
}

func (ds *Datastore) GetTeamsWithInstallerByHash(ctx context.Context, sha256, url string) (map[uint]*fleet.ExistingSoftwareInstaller, error) {
	stmt := `
SELECT
	si.id AS installer_id,
	si.team_id AS team_id,
	si.filename AS filename,
	si.extension AS extension,
	si.version AS version,
	si.platform AS platform,
	st.source AS source,
	st.bundle_identifier AS bundle_identifier,
	st.name AS title,
	si.package_ids AS package_ids
FROM
	software_installers si
	JOIN software_titles st ON si.title_id = st.id
WHERE
	si.storage_id = ?%s`

	var urlFilter string
	args := []any{sha256}
	if url != "" {
		urlFilter = " AND url = ?"
		args = append(args, url)
	}
	stmt = fmt.Sprintf(stmt, urlFilter)

	var installers []*fleet.ExistingSoftwareInstaller
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &installers, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installer by hash")
	}

	set := make(map[uint]*fleet.ExistingSoftwareInstaller, len(installers))
	for _, installer := range installers {
		// team ID 0 is No team in this context
		var tmID uint
		if installer.TeamID != nil {
			tmID = *installer.TeamID
		}
		if _, ok := set[tmID]; ok {
			return nil, ctxerr.New(ctx, fmt.Sprintf("cannot have multiple installers with the same hash %q on one team", sha256))
		}
		if installer.PackageIDList != "" {
			installer.PackageIDs = strings.Split(installer.PackageIDList, ",")
		}
		set[tmID] = installer
	}

	return set, nil
}
