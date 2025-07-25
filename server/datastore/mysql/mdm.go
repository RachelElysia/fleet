package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/go-kit/log/level"
	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetMDMCommandPlatform(ctx context.Context, commandUUID string) (string, error) {
	stmt := `
SELECT CASE
	WHEN EXISTS (SELECT 1 FROM nano_commands WHERE command_uuid = ?) THEN 'darwin'
	WHEN EXISTS (SELECT 1 FROM windows_mdm_commands WHERE command_uuid = ?) THEN 'windows'
	ELSE ''
END AS platform
`

	var p string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &p, stmt, commandUUID, commandUUID); err != nil {
		return "", err
	}
	if p == "" {
		return "", ctxerr.Wrap(ctx, notFound("MDMCommand").WithName(commandUUID))
	}

	return p, nil
}

func getCombinedMDMCommandsQuery(ds *Datastore, hostFilter string) (string, []interface{}) {
	appleStmt := `
SELECT
    nvq.id as host_uuid,
    nvq.command_uuid,
    COALESCE(NULLIF(nvq.status, ''), 'Pending') as status,
    COALESCE(nvq.result_updated_at, nvq.created_at) as updated_at,
    nvq.request_type as request_type,
    h.hostname,
    h.team_id
FROM
    nano_view_queue nvq
INNER JOIN
    hosts h
ON
    nvq.id = h.uuid
WHERE
   nvq.active = 1
`

	windowsStmt := `
SELECT
    mwe.host_uuid,
    wmc.command_uuid,
    COALESCE(NULLIF(wmcr.status_code, ''), 'Pending') as status,
    COALESCE(wmc.updated_at, wmc.created_at) as updated_at,
    wmc.target_loc_uri as request_type,
    h.hostname,
    h.team_id
FROM windows_mdm_commands wmc
LEFT JOIN windows_mdm_command_queue wmcq ON wmcq.command_uuid = wmc.command_uuid
LEFT JOIN windows_mdm_command_results wmcr ON wmc.command_uuid = wmcr.command_uuid
INNER JOIN mdm_windows_enrollments mwe ON wmcq.enrollment_id = mwe.id OR wmcr.enrollment_id = mwe.id
INNER JOIN hosts h ON h.uuid = mwe.host_uuid
WHERE TRUE
`

	var params []interface{}
	appleStmtWithFilter, params := ds.whereFilterHostsByIdentifier(hostFilter, appleStmt, params)
	windowsStmtWithFilter, params := ds.whereFilterHostsByIdentifier(hostFilter, windowsStmt, params)

	stmt := fmt.Sprintf(
		`SELECT * FROM ((%s) UNION ALL (%s)) as combined_commands WHERE `,
		appleStmtWithFilter, windowsStmtWithFilter,
	)

	return stmt, params
}

func (ds *Datastore) ListMDMCommands(
	ctx context.Context,
	tmFilter fleet.TeamFilter,
	listOpts *fleet.MDMCommandListOptions,
) ([]*fleet.MDMCommand, error) {
	if listOpts != nil && listOpts.Filters.HostIdentifier != "" {
		// separate codepath for more performant query by host identifier
		return ds.listMDMCommandsByHostIdentifier(ctx, tmFilter, listOpts)
	}

	jointStmt, params := getCombinedMDMCommandsQuery(ds, listOpts.Filters.HostIdentifier)
	jointStmt += ds.whereFilterHostsByTeams(tmFilter, "h")
	jointStmt, params = addRequestTypeFilter(jointStmt, &listOpts.Filters, params)
	jointStmt, params = appendListOptionsWithCursorToSQL(jointStmt, params, &listOpts.ListOptions)
	var results []*fleet.MDMCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, jointStmt, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list commands")
	}
	return results, nil
}

// listMDMCommandsByHostIdentifier retrieves MDM commands by host identifier. It is implemented as a
// distinct code path to optimize the query for use cases where a client may be polling for the
// status of commands for a specific host.
//
// TODO: Additional optimizations not implemented yet:
//   - restrict ordering by date to a new sorted index (probably `nano_enrollment_queue (id,
//     created_at DESC)` would be a good candidate)
//   - only search by hostname as a fallback if no results are found for UUID or hardware serial
func (ds *Datastore) listMDMCommandsByHostIdentifier(
	ctx context.Context,
	teamFilter fleet.TeamFilter,
	listOpts *fleet.MDMCommandListOptions,
) ([]*fleet.MDMCommand, error) {
	if listOpts == nil || listOpts.Filters.HostIdentifier == "" {
		return nil, ctxerr.Wrap(ctx, errors.New("listMDMCommandsByHostIdentifier requires non-empty listOpts.Filters.HostIdentifier"))
	}

	// First, search for host by identifier (hostname, uuid, or hardware_serial).
	//
	// NOTE: We're not using existing methods like ds.whereFilterHostsByIdentifier,
	// ds.HostIDsByIdentifier, ds.HostLiteByIdentifier because those methods are poorly
	// optimized for the indexes we currently have on the hosts table.
	// They filter with disjunctive conditions like `hostname = ? OR uuid = ?` as well as
	// `? IN(hostname, uuid)`. These existing queries aren't really suited for either composite
	// indexes or indexes on individual columns, and the optimizer ends up with executions that
	// resort full table scans or minimally filtered results (when the optimizer is using
	// indexes on team id and the like. Full-text indexes might be an option, but we've had
	// difficulties managing those for the hosts table in the past.
	//
	// So we're writing a custom query here that uses a UNION with three subqueries, each targeting
	// a specific column index: hostname, uuid, and hardware_serial.

	identifier := listOpts.Filters.HostIdentifier
	whereTeam := ds.whereFilterHostsByTeams(teamFilter, "h")
	columns := "id, uuid, hardware_serial, hostname, platform, team_id"

	// TODO: Add index for `hostname` or remove query? If removing, we'd need to update API
	// documentation? Breaking change? For now, adding a secondary team filter inside hostname part
	// of the union subquery to narrow the scope somewhat
	stmt := `
SELECT ` + columns + ` FROM (
	SELECT ` + columns + ` FROM hosts h WHERE hostname = ? AND ` + whereTeam + `
	UNION SELECT ` + columns + ` FROM hosts WHERE uuid = ?
	UNION SELECT ` + columns + ` FROM hosts WHERE hardware_serial = ? ) h
WHERE ` + whereTeam

	var dest []fleet.Host // NOTE: we're using the hosts struct for convenience, but it will not be fully populated
	args := []any{identifier, identifier, identifier}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &dest, stmt, args...)
	switch {
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "get host by identifier for mdm")
	case len(dest) == 0:
		// TODO: should we return an empty slice or an error?
		return []*fleet.MDMCommand{}, nil
	case len(dest) > 1:
		// TODO: how should we handle this unexpected case?
		level.Debug(ds.logger).Log("msg", "list mdm commands: multiple hosts found for identifier",
			"identifier", identifier, "count", len(dest),
		)
	}

	// Next, build the query to list MDM commands. If the found host(s) are on the same platform,
	// we can optimize the query by skipping the UNION ALL and using a single query targeted to the
	// platform.

	var appleStmt, winStmt string
	var appleParams, winParams []any
	var appleUUIDs, winUUIDs []string
	byUUID := make(map[string]fleet.Host, len(dest)) // map UUID to host so that we can loop over command results to add hostname and team info and avoid joining hosts to commands in DB
	for _, h := range dest {
		if prev, ok := byUUID[h.UUID]; ok {
			// TODO: how should we handle this unexpected case?
			level.Debug(ds.logger).Log("msg", "list mdm commands: multiple hosts found for identifier",
				"keeping", fmt.Sprintf("id: %d uuid: %s serial: %s hostname: %s platform: %s team: %+v", h.ID, h.UUID, h.HardwareSerial, h.Hostname, h.Platform, h.TeamID),
				"skipping", fmt.Sprintf("id: %d uuid: %s serial: %s hostname: %s platform: %s team: %+v", prev.ID, prev.UUID, prev.HardwareSerial, prev.Hostname, prev.Platform, prev.TeamID),
			)
		}
		byUUID[h.UUID] = h
		switch fleet.MDMPlatform(h.Platform) {
		case "darwin":
			appleUUIDs = append(appleUUIDs, h.UUID)
		case "windows":
			winUUIDs = append(winUUIDs, h.UUID)
		}
	}

	if len(appleUUIDs) > 0 {
		appleParams = []any{appleUUIDs}
		appleStmt = `
SELECT
	nq.id AS host_uuid,
	nc.command_uuid,
	COALESCE(ncr.updated_at, nc.created_at) AS updated_at,
	COALESCE(NULLIF(ncr.status, ''), 'Pending') AS status,
	request_type
FROM
	nano_enrollment_queue nq
	JOIN nano_commands nc ON nq.command_uuid = nc.command_uuid
	LEFT JOIN nano_command_results ncr ON nq.id = ncr.id
		AND nc.command_uuid = ncr.command_uuid
WHERE
	nq.id IN(?) AND nq.active = 1`

		appleStmt, appleParams = addRequestTypeFilter(appleStmt, &listOpts.Filters, appleParams)
		appleStmt, appleParams, err = sqlx.In(appleStmt, appleParams...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "prepare query to list MDM commands for Apple devices")
		}
	}

	if len(winUUIDs) > 0 {
		winParams = []any{winUUIDs}
		winStmt = `
SELECT
	mwe.host_uuid,
	wq.command_uuid,
	COALESCE(wcr.updated_at, wc.created_at) AS updated_at,
	COALESCE(NULLIF(wcr.status_code, ''), 'Pending') AS status,
	wc.target_loc_uri AS request_type
FROM
	windows_mdm_command_queue wq
	JOIN mdm_windows_enrollments mwe ON mwe.id = wq.enrollment_id
	JOIN windows_mdm_commands wc ON wc.command_uuid = wq.command_uuid
	LEFT JOIN windows_mdm_command_results wcr ON wcr.command_uuid = wq.command_uuid
		AND wcr.enrollment_id = wq.enrollment_id
WHERE
	mwe.host_uuid IN (?)`

		winStmt, winParams = addRequestTypeFilter(winStmt, &listOpts.Filters, winParams)
		winStmt, winParams, err = sqlx.In(winStmt, winParams...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "prepare query to list MDM commands for Windows devices")
		}
	}

	var listStmt string
	var params []any
	switch {
	case len(appleUUIDs) > 0 && len(winUUIDs) > 0:
		listStmt = fmt.Sprintf(`SELECT * FROM ((%s) UNION ALL (%s)) u`,
			appleStmt, winStmt)
		params = append(params, appleParams...)
		params = append(params, winParams...)
	case len(appleUUIDs) > 0:
		listStmt = appleStmt
		params = appleParams
	case len(winUUIDs) > 0:
		listStmt = winStmt
		params = winParams
	}

	// TODO: Maybe move this to the service method? What about pagination metadata?
	if listOpts.OrderKey == "" {
		listOpts.OrderKey = "updated_at"
	}
	// // FIXME: We probably ought to modify how listOptionsFromRequest in transport.go applies the
	// // default order direction. Defaulting to ascending doesn't make sense for date fields like
	// // updated_at. List options are decoded by transport before the specific gets the request
	// // struct so there's no way apply a different default because at that point we can't tell if
	// // the direction was set by the user or not. One approach would be to have listOptionsFromRequest
	// // check if the order key is a date field (i.e. it ends with "_at") and default to descending
	// // in those cases.
	// if listOpts.OrderDirection == "" {
	// 	listOpts.OrderDirection = fleet.OrderDescending
	// }
	if listOpts.PerPage == 0 {
		listOpts.PerPage = 10
	}
	listStmt, params = appendListOptionsWithCursorToSQL(listStmt, params, &listOpts.ListOptions)

	var results []*fleet.MDMCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, listStmt, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list commands")
	}

	// Add hostname and team info to the results based on the host UUIDs.
	for i := range results {
		if host, ok := byUUID[results[i].HostUUID]; ok {
			results[i].Hostname = host.Hostname
			results[i].TeamID = host.TeamID
		}
	}

	return results, nil
}

func addRequestTypeFilter(stmt string, filter *fleet.MDMCommandFilters, params []interface{}) (string, []interface{}) {
	if filter.RequestType != "" {
		stmt += " AND request_type = ?"
		params = append(params, filter.RequestType)
	}

	return stmt, params
}

func (ds *Datastore) getMDMCommand(ctx context.Context, q sqlx.QueryerContext, cmdUUID string) (*fleet.MDMCommand, error) {
	stmt, _ := getCombinedMDMCommandsQuery(ds, "")
	stmt += "command_uuid = ?"

	var cmd fleet.MDMCommand
	if err := sqlx.GetContext(ctx, q, &cmd, stmt, cmdUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get mdm command by UUID")
	}
	return &cmd, nil
}

func (ds *Datastore) BatchSetMDMProfiles(ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile,
	winProfiles []*fleet.MDMWindowsConfigProfile, macDeclarations []*fleet.MDMAppleDeclaration, profilesVariablesByIdentifier []fleet.MDMProfileIdentifierFleetVariables,
) (updates fleet.MDMProfilesUpdates, err error) {
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		if updates.WindowsConfigProfile, err = ds.batchSetMDMWindowsProfilesDB(ctx, tx, tmID, winProfiles); err != nil {
			return ctxerr.Wrap(ctx, err, "batch set windows profiles")
		}

		// for now, only apple profiles support Fleet variables
		if updates.AppleConfigProfile, err = ds.batchSetMDMAppleProfilesDB(ctx, tx, tmID, macProfiles, profilesVariablesByIdentifier); err != nil {
			return ctxerr.Wrap(ctx, err, "batch set apple profiles")
		}

		if updates.AppleDeclaration, err = ds.batchSetMDMAppleDeclarations(ctx, tx, tmID, macDeclarations); err != nil {
			return ctxerr.Wrap(ctx, err, "batch set apple declarations")
		}

		return nil
	})
	return updates, err
}

func (ds *Datastore) ListMDMConfigProfiles(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]*fleet.MDMConfigProfilePayload, *fleet.PaginationMetadata, error) {
	// this lists custom profiles, it explicitly filters out the fleet-reserved
	// ones (reserved identifiers for Apple profiles, reserved names for Windows).

	var profs []*fleet.MDMConfigProfilePayload

	const selectStmt = `
SELECT
	profile_uuid,
	team_id,
	name,
	scope,
	platform,
	identifier,
	checksum,
	created_at,
	uploaded_at
FROM (
	SELECT
		profile_uuid,
		team_id,
		name,
		scope,
		'darwin' as platform,
		identifier,
		checksum,
		created_at,
		uploaded_at
	FROM
		mdm_apple_configuration_profiles
	WHERE
		team_id = ? AND
		identifier NOT IN (?)

	UNION ALL

	SELECT
		profile_uuid,
		team_id,
		name,
		'' as scope,
		'windows' as platform,
		'' as identifier,
		'' as checksum,
		created_at,
		uploaded_at
	FROM
		mdm_windows_configuration_profiles
	WHERE
		team_id = ? AND
		name NOT IN (?)

	UNION ALL

	SELECT
		declaration_uuid AS profile_uuid,
		team_id,
		name,
		scope,
		'darwin' AS platform,
		identifier,
		token AS checksum,
		created_at,
		uploaded_at
	FROM mdm_apple_declarations
	WHERE team_id = ? AND
		name NOT IN (?)
) as combined_profiles
`

	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	fleetIdentsMap := mobileconfig.FleetPayloadIdentifiers()
	fleetIdentifiers := make([]string, 0, len(fleetIdentsMap))
	for k := range fleetIdentsMap {
		fleetIdentifiers = append(fleetIdentifiers, k)
	}
	fleetNamesMap := mdm.FleetReservedProfileNames()
	fleetNames := make([]string, 0, len(fleetNamesMap))
	for k := range fleetNamesMap {
		fleetNames = append(fleetNames, k)
	}

	args := []any{globalOrTeamID, fleetIdentifiers, globalOrTeamID, fleetNames, globalOrTeamID, fleetNames}
	stmt, args := appendListOptionsWithCursorToSQL(selectStmt, args, &opt)

	stmt, args, err := sqlx.In(stmt, args...)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "sqlx.In ListMDMConfigProfiles")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profs, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select profiles")
	}

	var metaData *fleet.PaginationMetadata
	if opt.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(profs) > int(opt.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			profs = profs[:len(profs)-1]
		}
	}

	// load the labels associated with those profiles
	var winProfUUIDs, macProfUUIDs, macDeclUUIDs []string
	for _, prof := range profs {
		if prof.Platform == "windows" {
			winProfUUIDs = append(winProfUUIDs, prof.ProfileUUID)
		} else {
			if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAppleDeclarationUUIDPrefix) {
				macDeclUUIDs = append(macDeclUUIDs, prof.ProfileUUID)
				continue
			}

			macProfUUIDs = append(macProfUUIDs, prof.ProfileUUID)
		}
	}
	labels, err := ds.listProfileLabelsForProfiles(ctx, winProfUUIDs, macProfUUIDs, macDeclUUIDs)
	if err != nil {
		return nil, nil, err
	}

	// match the labels with their profiles
	profMap := make(map[string]*fleet.MDMConfigProfilePayload, len(profs))
	for _, prof := range profs {
		profMap[prof.ProfileUUID] = prof
	}
	for _, label := range labels {
		if prof, ok := profMap[label.ProfileUUID]; ok {
			switch {
			case label.Exclude && label.RequireAll:
				// this should never happen so log it for debugging
				level.Debug(ds.logger).Log("msg", "unsupported profile label: cannot be both exclude and require all",
					"profile_uuid", label.ProfileUUID,
					"label_name", label.LabelName,
				)
			case label.Exclude && !label.RequireAll:
				prof.LabelsExcludeAny = append(prof.LabelsExcludeAny, label)
			case !label.Exclude && !label.RequireAll:
				prof.LabelsIncludeAny = append(prof.LabelsIncludeAny, label)
			default:
				// default include all
				prof.LabelsIncludeAll = append(prof.LabelsIncludeAll, label)
			}
		}
	}

	return profs, metaData, nil
}

func (ds *Datastore) listProfileLabelsForProfiles(ctx context.Context, winProfUUIDs, macProfUUIDs, macDeclUUIDs []string) ([]fleet.ConfigurationProfileLabel, error) {
	// load the labels associated with those profiles
	const labelsStmt = `
SELECT
	COALESCE(apple_profile_uuid, windows_profile_uuid) as profile_uuid,
	label_name,
	COALESCE(label_id, 0) as label_id,
	IF(label_id IS NULL, 1, 0) as broken,
	exclude,
	require_all
FROM
	mdm_configuration_profile_labels mcpl
WHERE
	mcpl.apple_profile_uuid IN (?) OR
	mcpl.windows_profile_uuid IN (?)
UNION ALL
SELECT
	apple_declaration_uuid as profile_uuid,
	label_name,
	COALESCE(label_id, 0) as label_id,
	IF(label_id IS NULL, 1, 0) as broken,
	exclude,
	require_all
FROM
	mdm_declaration_labels mdl
WHERE
	mdl.apple_declaration_uuid IN (?)
ORDER BY
	profile_uuid, label_name
`
	// ensure there's at least one (non-matching) value in the slice so the IN
	// clause is valid
	if len(winProfUUIDs) == 0 {
		winProfUUIDs = []string{"-"}
	}
	if len(macProfUUIDs) == 0 {
		macProfUUIDs = []string{"-"}
	}
	if len(macDeclUUIDs) == 0 {
		macDeclUUIDs = []string{"-"}
	}

	stmt, args, err := sqlx.In(labelsStmt, macProfUUIDs, winProfUUIDs, macDeclUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.In to list labels for profiles")
	}

	var labels []fleet.ConfigurationProfileLabel
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select profiles labels")
	}
	return labels, nil
}

func (ds *Datastore) BulkSetPendingMDMHostProfiles(
	ctx context.Context,
	hostIDs, teamIDs []uint,
	profileUUIDs, hostUUIDs []string,
) (updates fleet.MDMProfilesUpdates, err error) {
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		updates, err = ds.bulkSetPendingMDMHostProfilesDB(ctx, tx, hostIDs, teamIDs, profileUUIDs, hostUUIDs)
		return err
	})
	return updates, err
}

// Note that team ID 0 is used for profiles that apply to hosts in no team
// (i.e. pass 0 in that case as part of the teamIDs slice). Only one of the
// slice arguments can have values.
func (ds *Datastore) bulkSetPendingMDMHostProfilesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostIDs, teamIDs []uint,
	profileUUIDs, hostUUIDs []string,
) (updates fleet.MDMProfilesUpdates, err error) {
	var (
		countArgs     int
		macProfUUIDs  []string
		winProfUUIDs  []string
		hasAppleDecls bool
	)

	if len(hostIDs) > 0 {
		countArgs++
	}
	if len(teamIDs) > 0 {
		countArgs++
	}
	if len(profileUUIDs) > 0 {
		countArgs++

		// split into mac and win profiles
		for _, puid := range profileUUIDs {
			if strings.HasPrefix(puid, fleet.MDMAppleProfileUUIDPrefix) { //nolint:gocritic // ignore ifElseChain
				macProfUUIDs = append(macProfUUIDs, puid)
			} else if strings.HasPrefix(puid, fleet.MDMAppleDeclarationUUIDPrefix) {
				hasAppleDecls = true
			} else {
				// Note: defaulting to windows profiles without checking the prefix as
				// many tests fail otherwise and it's a whole rabbit hole that I can't
				// address at the moment.
				winProfUUIDs = append(winProfUUIDs, puid)
			}
		}
	}
	if len(hostUUIDs) > 0 {
		countArgs++
	}
	if countArgs > 1 {
		return updates, errors.New("only one of hostIDs, teamIDs, profileUUIDs or hostUUIDs can be provided")
	}
	if countArgs == 0 {
		return updates, nil
	}

	var countProfUUIDs int
	if len(macProfUUIDs) > 0 {
		countProfUUIDs++
	}
	if len(winProfUUIDs) > 0 {
		countProfUUIDs++
	}
	if hasAppleDecls {
		countProfUUIDs++
	}
	if countProfUUIDs > 1 {
		return updates, errors.New("profile uuids must be all Apple profiles, all Apple declarations, or all Windows profiles")
	}

	var (
		hosts    []fleet.Host
		args     []any
		uuidStmt string
	)

	switch {
	case len(hostUUIDs) > 0:
		// TODO: if a very large number (~65K) of uuids was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `SELECT uuid, platform FROM hosts WHERE uuid IN (?)`
		args = append(args, hostUUIDs)

	case len(hostIDs) > 0:
		// TODO: if a very large number (~65K) of uuids was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `SELECT uuid, platform FROM hosts WHERE id IN (?)`
		args = append(args, hostIDs)

	case len(teamIDs) > 0:
		// TODO: if a very large number (~65K) of team IDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `SELECT uuid, platform FROM hosts WHERE `
		if len(teamIDs) == 1 && teamIDs[0] == 0 {
			uuidStmt += `team_id IS NULL`
		} else {
			uuidStmt += `team_id IN (?)`
			args = append(args, teamIDs)
			for _, tmID := range teamIDs {
				if tmID == 0 {
					uuidStmt += ` OR team_id IS NULL`
					break
				}
			}
		}

	case len(macProfUUIDs) > 0:
		// TODO: if a very large number (~65K/2) of profile UUIDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `
SELECT DISTINCT h.uuid, h.platform
FROM hosts h
JOIN mdm_apple_configuration_profiles macp
	ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
LEFT JOIN host_mdm_apple_profiles hmap
	ON h.uuid = hmap.host_uuid
WHERE
	macp.profile_uuid IN (?) AND (h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados')
OR
	hmap.profile_uuid IN (?) AND (h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados')`
		args = append(args, macProfUUIDs, macProfUUIDs)

	case len(winProfUUIDs) > 0:
		// TODO: if a very large number (~65K/2) of profile IDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `
SELECT DISTINCT h.uuid, h.platform
FROM hosts h
JOIN mdm_windows_configuration_profiles mawp
	ON h.team_id = mawp.team_id OR (h.team_id IS NULL AND mawp.team_id = 0)
LEFT JOIN host_mdm_windows_profiles hmwp
	ON h.uuid = hmwp.host_uuid
WHERE
	mawp.profile_uuid IN (?) AND h.platform = 'windows'
OR
	hmwp.profile_uuid IN (?) AND h.platform = 'windows'`
		args = append(args, winProfUUIDs, winProfUUIDs)

	}

	// TODO: this could be optimized to avoid querying for platform when
	// profileIDs or profileUUIDs are provided.
	if len(hosts) == 0 && !hasAppleDecls {
		uuidStmt, args, err := sqlx.In(uuidStmt, args...)
		if err != nil {
			return updates, ctxerr.Wrap(ctx, err, "prepare query to load host UUIDs")
		}
		if err := sqlx.SelectContext(ctx, tx, &hosts, uuidStmt, args...); err != nil {
			return updates, ctxerr.Wrap(ctx, err, "execute query to load host UUIDs")
		}
	}

	var appleHosts []string
	var winHosts []string
	for _, h := range hosts {
		switch h.Platform {
		case "darwin", "ios", "ipados":
			appleHosts = append(appleHosts, h.UUID)
		case "windows":
			winHosts = append(winHosts, h.UUID)
		default:
			level.Debug(ds.logger).Log(
				"msg", "tried to set profile status for a host with unsupported platform",
				"platform", h.Platform,
				"host_uuid", h.UUID,
			)
		}
	}

	updates.AppleConfigProfile, err = ds.bulkSetPendingMDMAppleHostProfilesDB(ctx, tx, appleHosts, profileUUIDs)
	if err != nil {
		return updates, ctxerr.Wrap(ctx, err, "bulk set pending apple host profiles")
	}

	updates.WindowsConfigProfile, err = ds.bulkSetPendingMDMWindowsHostProfilesDB(ctx, tx, winHosts, profileUUIDs)
	if err != nil {
		return updates, ctxerr.Wrap(ctx, err, "bulk set pending windows host profiles")
	}

	const defaultBatchSize = 1000
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}
	// TODO(roberto): this method currently sets the state of all
	// declarations for all hosts. I don't see an immediate concern
	// (and my hunch is that we could even do the same for
	// profiles) but this could be optimized to use only a provided
	// set of host uuids.
	//
	// Note(victor): Why is the status being set to nil? Shouldn't it be set to pending?
	// Or at least pending for install and nil for remove profiles. Please update this comment if you know.
	// This method is called bulkSetPendingMDMHostProfilesDB, so it is confusing that the status is NOT explicitly set to pending.
	_, updates.AppleDeclaration, err = mdmAppleBatchSetHostDeclarationStateDB(ctx, tx, batchSize, nil)
	if err != nil {
		return updates, ctxerr.Wrap(ctx, err, "bulk set pending apple declarations")
	}

	return updates, nil
}

func (ds *Datastore) UpdateHostMDMProfilesVerification(ctx context.Context, host *fleet.Host, toVerify, toFail, toRetry []string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := setMDMProfilesVerifiedDB(ctx, tx, host, toVerify); err != nil {
			return err
		}
		if err := setMDMProfilesFailedDB(ctx, tx, host, toFail); err != nil {
			return err
		}
		if err := setMDMProfilesRetryDB(ctx, tx, host, toRetry); err != nil {
			return err
		}
		return nil
	})
}

// setMDMProfilesRetryDB sets the status of the given identifiers to retry (nil) and increments the retry count
func setMDMProfilesRetryDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host, identifiersOrNames []string) error {
	if len(identifiersOrNames) == 0 {
		return nil
	}

	const baseStmt = `
UPDATE
	%s
SET
	status = NULL,
	detail = '',
	retries = retries + 1
WHERE
	host_uuid = ?
	AND operation_type = ?
	-- do not increment retry unnecessarily if the status is already null, no MDM command was sent
	AND status IS NOT NULL
	AND %s IN(?)`

	args := []interface{}{
		host.UUID,
		fleet.MDMOperationTypeInstall,
		identifiersOrNames,
	}

	var stmt string
	switch host.Platform {
	case "darwin", "ios", "ipados":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_apple_profiles", "profile_identifier")
	case "windows":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_windows_profiles", "profile_name")
	default:
		return fmt.Errorf("unsupported platform %s", host.Platform)
	}
	stmt, args, err := sqlx.In(stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building sql statement to set retry host profiles")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "setting retry host profiles")
	}
	return nil
}

// setMDMProfilesFailedDB sets the status of the given identifiers to failed if the current status
// is verifying or verified. It also sets the detail to a message indicating that the profile was
// either verifying or verified. Only profiles with the install operation type are updated.
func setMDMProfilesFailedDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host, identifiersOrNames []string) error {
	if len(identifiersOrNames) == 0 {
		return nil
	}

	const baseStmt = `
UPDATE
	%s
SET
	detail = if(status = ?, ?, ?),
	status = ?
WHERE
	host_uuid = ?
	AND status IN(?)
	AND operation_type = ?
	AND %s IN(?)`

	var stmt string
	switch host.Platform {
	case "darwin", "ios", "ipados":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_apple_profiles", "profile_identifier")
	case "windows":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_windows_profiles", "profile_name")
	default:
		return fmt.Errorf("unsupported platform %s", host.Platform)
	}

	args := []interface{}{
		fleet.MDMDeliveryVerifying,
		fleet.HostMDMProfileDetailFailedWasVerifying,
		fleet.HostMDMProfileDetailFailedWasVerified,
		fleet.MDMDeliveryFailed,
		host.UUID,
		[]interface{}{
			fleet.MDMDeliveryVerifying,
			fleet.MDMDeliveryVerified,
		},
		fleet.MDMOperationTypeInstall,
		identifiersOrNames,
	}
	stmt, args, err := sqlx.In(stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building sql statement to set failed host profiles")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "setting failed host profiles")
	}
	return nil
}

// setMDMProfilesVerifiedDB sets the status of the given identifiers to verified if the current
// status is verifying. Only profiles with the install operation type are updated.
func setMDMProfilesVerifiedDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host, identifiersOrNames []string) error {
	if len(identifiersOrNames) == 0 {
		return nil
	}

	const baseStmt = `
UPDATE
	%s
SET
	detail = '',
	status = ?
WHERE
	host_uuid = ?
	AND status IN(?)
	AND operation_type = ?
	AND %s IN(?)`

	var stmt string
	switch host.Platform {
	case "darwin", "ios", "ipados":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_apple_profiles", "profile_identifier")
	case "windows":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_windows_profiles", "profile_name")
	default:
		return fmt.Errorf("unsupported platform %s", host.Platform)
	}

	args := []interface{}{
		fleet.MDMDeliveryVerified,
		host.UUID,
		[]interface{}{
			fleet.MDMDeliveryPending,
			fleet.MDMDeliveryVerifying,
			fleet.MDMDeliveryFailed,
		},
		fleet.MDMOperationTypeInstall,
		identifiersOrNames,
	}
	stmt, args, err := sqlx.In(stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building sql statement to set verified host macOS profiles")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "setting verified host profiles")
	}
	return nil
}

func (ds *Datastore) GetHostMDMProfilesExpectedForVerification(ctx context.Context, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error) {
	var teamID uint
	if host.TeamID != nil {
		teamID = *host.TeamID
	}

	switch host.Platform {
	case "darwin", "ios", "ipados":
		return ds.getHostMDMAppleProfilesExpectedForVerification(ctx, teamID, host)
	case "windows":
		return ds.getHostMDMWindowsProfilesExpectedForVerification(ctx, teamID, host.ID)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", host.Platform)
	}
}

func (ds *Datastore) getHostMDMWindowsProfilesExpectedForVerification(ctx context.Context, teamID, hostID uint) (map[string]*fleet.ExpectedMDMProfile, error) {
	stmt := `
-- profiles without labels
SELECT
    mwcp.profile_uuid AS profile_uuid,
	name,
	syncml AS raw_profile,
	min(mwcp.uploaded_at) AS earliest_install_date,
	0 AS count_profile_labels,
	0 AS count_non_broken_labels,
	0 AS count_host_labels
FROM
	mdm_windows_configuration_profiles mwcp
WHERE
	mwcp.team_id = ? AND
	NOT EXISTS (
		SELECT
			1
		FROM
			mdm_configuration_profile_labels mcpl
		WHERE
			mcpl.windows_profile_uuid = mwcp.profile_uuid
	)
GROUP BY profile_uuid, name, syncml

UNION

-- label-based profiles where the host is a member of all the labels (include-all).
-- by design, "include" labels cannot match if they are broken (the host cannot be
-- a member of a deleted label).
SELECT
	mwcp.profile_uuid AS profile_uuid,
	name,
	syncml AS raw_profile,
	min(mwcp.uploaded_at) AS earliest_install_date,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) as count_non_broken_labels,
	COUNT(lm.label_id) AS count_host_labels
FROM
	mdm_windows_configuration_profiles mwcp
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.windows_profile_uuid = mwcp.profile_uuid AND mcpl.exclude = 0 AND mcpl.require_all = 1
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	mwcp.team_id = ?
GROUP BY
	profile_uuid, name, syncml
HAVING
	count_profile_labels > 0 AND
	count_host_labels = count_profile_labels

UNION

-- label-based entities where the host is NOT a member of any of the labels (exclude-any).
-- explicitly ignore profiles with broken excluded labels so that they are never applied.
SELECT
	mwcp.profile_uuid AS profile_uuid,
	name,
	syncml AS raw_profile,
	min(mwcp.uploaded_at) AS earliest_install_date,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) as count_non_broken_labels,
	COUNT(lm.label_id) AS count_host_labels
FROM
	mdm_windows_configuration_profiles mwcp
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.windows_profile_uuid = mwcp.profile_uuid AND mcpl.exclude = 1
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	mwcp.team_id = ?
GROUP BY
	profile_uuid, name, syncml
HAVING
	-- considers only the profiles with labels, without any broken label, and with the host not in any label
	count_profile_labels > 0 AND
	count_profile_labels = count_non_broken_labels AND
	count_host_labels = 0

UNION

-- label-based profiles where the host is a member of at least one of the labels (include-any)
SELECT
	mwcp.profile_uuid AS profile_uuid,
	name,
	syncml AS raw_profile,
	min(mwcp.uploaded_at) AS earliest_install_date,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) as count_non_broken_labels,
	COUNT(lm.label_id) AS count_host_labels
FROM
	mdm_windows_configuration_profiles mwcp
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.windows_profile_uuid = mwcp.profile_uuid AND mcpl.exclude = 0 AND mcpl.require_all = 0
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	mwcp.team_id = ?
GROUP BY
	profile_uuid, name, syncml
HAVING
	count_profile_labels > 0 AND
	count_host_labels > 0
`
	var profiles []*fleet.ExpectedMDMProfile
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, teamID, hostID, teamID, hostID, teamID, hostID, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "running query for windows profiles")
	}

	byName := make(map[string]*fleet.ExpectedMDMProfile, len(profiles))
	for _, r := range profiles {
		byName[r.Name] = r
	}

	return byName, nil
}

func (ds *Datastore) getHostMDMAppleProfilesExpectedForVerification(ctx context.Context, teamID uint, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error) {
	// TODO This will need to be updated to support scopes
	stmt := `
-- profiles without labels
SELECT
	macp.profile_uuid AS profile_uuid,
	macp.identifier AS identifier,
	0 AS count_profile_labels,
	0 AS count_non_broken_labels,
	0 AS count_host_labels,
	earliest_install_date
FROM
	mdm_apple_configuration_profiles macp
	JOIN (
		SELECT
			checksum,
			min(uploaded_at) AS earliest_install_date
		FROM
			mdm_apple_configuration_profiles
		GROUP BY checksum
	) cs ON macp.checksum = cs.checksum
WHERE
	macp.team_id = ? AND
	NOT EXISTS (
		SELECT
			1
		FROM
			mdm_configuration_profile_labels mcpl
		WHERE
			mcpl.apple_profile_uuid = macp.profile_uuid
	)

UNION

-- label-based profiles where the host is a member of all the labels (include-all)
-- by design, "include" labels cannot match if they are broken (the host cannot be
-- a member of a deleted label).
SELECT
	macp.profile_uuid AS profile_uuid,
	macp.identifier AS identifier,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) AS count_non_broken_labels,
	COUNT(lm.label_id) AS count_host_labels,
	min(earliest_install_date) AS earliest_install_date
FROM
	mdm_apple_configuration_profiles macp
	JOIN (
		SELECT
			checksum,
			min(uploaded_at) AS earliest_install_date
		FROM
			mdm_apple_configuration_profiles
		GROUP BY checksum
	) cs ON macp.checksum = cs.checksum
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.apple_profile_uuid = macp.profile_uuid AND mcpl.exclude = 0 AND mcpl.require_all = 1
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	macp.team_id = ?
GROUP BY
	profile_uuid, identifier
HAVING
	count_profile_labels > 0 AND
	count_host_labels = count_profile_labels

UNION

-- label-based entities where the host is NOT a member of any of the labels (exclude-any).
-- explicitly ignore profiles with broken excluded labels so that they are never applied.
SELECT
	macp.profile_uuid AS profile_uuid,
	macp.identifier AS identifier,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) AS count_non_broken_labels,
	COUNT(lm.label_id) AS count_host_labels,
	min(earliest_install_date) AS earliest_install_date
FROM
	mdm_apple_configuration_profiles macp
	JOIN (
		SELECT
			checksum,
			min(uploaded_at) AS earliest_install_date
		FROM
			mdm_apple_configuration_profiles
		GROUP BY checksum
	) cs ON macp.checksum = cs.checksum
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.apple_profile_uuid = macp.profile_uuid AND mcpl.exclude = 1
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	macp.team_id = ?
GROUP BY
	profile_uuid, identifier
HAVING
	-- considers only the profiles with labels, without any broken label, and with the host not in any label
	count_profile_labels > 0 AND
	count_profile_labels = count_non_broken_labels AND
	count_host_labels = 0

UNION

-- label-based profiles where the host is a member of at least one of the labels (include-any)
SELECT
	macp.profile_uuid AS profile_uuid,
	macp.identifier AS identifier,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) AS count_non_broken_labels,
	COUNT(lm.label_id) AS count_host_labels,
	min(earliest_install_date) AS earliest_install_date
FROM
	mdm_apple_configuration_profiles macp
	JOIN (
		SELECT
			checksum,
			min(uploaded_at) AS earliest_install_date
		FROM
			mdm_apple_configuration_profiles
		GROUP BY checksum
	) cs ON macp.checksum = cs.checksum
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.apple_profile_uuid = macp.profile_uuid AND mcpl.exclude = 0 AND mcpl.require_all = 0
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	macp.team_id = ?
GROUP BY
	profile_uuid, identifier
HAVING
	count_profile_labels > 0 AND
	count_host_labels > 0
`

	var rows []*fleet.ExpectedMDMProfile
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, teamID, host.ID, teamID, host.ID, teamID, host.ID, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("getting expected profiles for host in team %d", teamID))
	}

	// Fetch variables_updated_at for host profiles that have it set and override
	// earliest_install_date if it's older than variables_updated_at.
	variableUpdateTimes := []struct {
		ProfileUUID        string    `db:"profile_uuid"`
		VariablesUpdatedAt time.Time `db:"variables_updated_at"`
	}{}
	variableUpdateTimesStmt := `
	SELECT profile_uuid, variables_updated_at AS variables_updated_at
	FROM host_mdm_apple_profiles
	WHERE host_uuid = ? AND variables_updated_at IS NOT NULL
	`

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &variableUpdateTimes, variableUpdateTimesStmt, host.UUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("getting expected profiles for host in team %d", teamID))
	}
	variableUpdateTimesByProfileUUID := make(map[string]time.Time, len(variableUpdateTimes))
	for _, r := range variableUpdateTimes {
		variableUpdateTimesByProfileUUID[r.ProfileUUID] = r.VariablesUpdatedAt
	}

	expectedProfilesByIdentifier := make(map[string]*fleet.ExpectedMDMProfile, len(rows))
	for _, r := range rows {
		if variableUpdateTime, ok := variableUpdateTimesByProfileUUID[r.ProfileUUID]; ok && variableUpdateTime.After(r.EarliestInstallDate) {
			r.EarliestInstallDate = variableUpdateTime
		}
		expectedProfilesByIdentifier[r.Identifier] = r
	}

	return expectedProfilesByIdentifier, nil
}

func (ds *Datastore) GetHostMDMProfilesRetryCounts(ctx context.Context, host *fleet.Host) ([]fleet.HostMDMProfileRetryCount, error) {
	const darwinStmt = `
SELECT
	profile_identifier,
	retries
FROM
	host_mdm_apple_profiles hmap
WHERE
	hmap.host_uuid = ?`

	const windowsStmt = `
SELECT
	profile_name,
	retries
FROM
	host_mdm_windows_profiles hmwp
WHERE
	hmwp.host_uuid = ?`

	var stmt string
	switch host.Platform {
	case "darwin", "ios", "ipados":
		stmt = darwinStmt
	case "windows":
		stmt = windowsStmt
	default:
		return nil, fmt.Errorf("unsupported platform %s", host.Platform)
	}

	var dest []fleet.HostMDMProfileRetryCount
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dest, stmt, host.UUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("getting retry counts for host %s", host.UUID))
	}

	return dest, nil
}

func (ds *Datastore) GetHostMDMProfileRetryCountByCommandUUID(ctx context.Context, host *fleet.Host, cmdUUID string) (fleet.HostMDMProfileRetryCount, error) {
	const darwinStmt = `
SELECT
	profile_identifier, retries
FROM
	host_mdm_apple_profiles hmap
WHERE
	hmap.host_uuid = ?
	AND hmap.command_uuid = ?`

	const windowsStmt = `
SELECT
	profile_uuid, retries
FROM
	host_mdm_windows_profiles hmwp
WHERE
	hmwp.host_uuid = ?
	AND hmwp.command_uuid = ?`

	var stmt string
	switch host.Platform {
	case "darwin", "ios", "ipados":
		stmt = darwinStmt
	case "windows":
		stmt = windowsStmt
	default:
		return fleet.HostMDMProfileRetryCount{}, fmt.Errorf("unsupported platform %s", host.Platform)
	}

	var dest fleet.HostMDMProfileRetryCount
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, host.UUID, cmdUUID); err != nil {
		if err == sql.ErrNoRows {
			return dest, notFound("HostMDMCommand").WithMessage(fmt.Sprintf("command uuid %s not found for host uuid %s", cmdUUID, host.UUID))
		}
		return dest, ctxerr.Wrap(ctx, err, fmt.Sprintf("getting retry count for host %s command uuid %s", host.UUID, cmdUUID))
	}

	return dest, nil
}

func batchSetProfileLabelAssociationsDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	profileLabels []fleet.ConfigurationProfileLabel,
	profileUUIDsWithoutLabels []string,
	platform string,
) (updatedDB bool, err error) {
	if len(profileLabels)+len(profileUUIDsWithoutLabels) == 0 {
		return false, nil
	}

	var platformPrefix string
	switch platform {
	case "darwin":
		// map "darwin" to "apple" to be consistent with other
		// "platform-agnostic" datastore methods. We initially used "darwin"
		// because that's what hosts use (as the data is reported by osquery)
		// and sometimes we want to dynamically select a table based on host
		// data.
		platformPrefix = "apple"
	case "windows":
		platformPrefix = "windows"
	default:
		return false, fmt.Errorf("unsupported platform %s", platform)
	}

	// delete any profile+label tuple that is NOT in the list of provided tuples
	// but are associated with the provided profiles (so we don't delete
	// unrelated profile+label tuples)
	deleteStmt := `
	  DELETE FROM mdm_configuration_profile_labels
	  WHERE (%s_profile_uuid, label_id) NOT IN (%s) AND
	  %s_profile_uuid IN (?)
	`

	// used when only profileUUIDsWithoutLabels is provided, there are no
	// labels to keep, delete all labels for profiles in this list.
	deleteNoLabelStmt := `
	  DELETE FROM mdm_configuration_profile_labels
	  WHERE %s_profile_uuid IN (?)
	`

	upsertStmt := `
	  INSERT INTO mdm_configuration_profile_labels
              (%s_profile_uuid, label_id, label_name, exclude, require_all)
          VALUES
              %s
          ON DUPLICATE KEY UPDATE
              label_id = VALUES(label_id),
              exclude = VALUES(exclude),
			  require_all = VALUES(require_all)
	`

	selectStmt := `
		SELECT %s_profile_uuid as profile_uuid, label_id, label_name, exclude, require_all FROM mdm_configuration_profile_labels
		WHERE (%s_profile_uuid, label_name) IN (%s)
	`

	if len(profileLabels) == 0 {
		deleteNoLabelStmt = fmt.Sprintf(deleteNoLabelStmt, platformPrefix)
		deleteNoLabelStmt, args, err := sqlx.In(deleteNoLabelStmt, profileUUIDsWithoutLabels)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "sqlx.In delete labels for profiles without labels")
		}

		var result sql.Result
		if result, err = tx.ExecContext(ctx, deleteNoLabelStmt, args...); err != nil {
			return false, ctxerr.Wrap(ctx, err, "deleting labels for profiles without labels")
		}
		if result != nil {
			rows, _ := result.RowsAffected()
			updatedDB = rows > 0
		}
		return updatedDB, nil
	}

	var (
		insertBuilder         strings.Builder
		selectOrDeleteBuilder strings.Builder
		selectParams          []any
		insertParams          []any
		deleteParams          []any

		setProfileUUIDs = make(map[string]struct{})
	)

	labelsToInsert := make(map[string]*fleet.ConfigurationProfileLabel, len(profileLabels))
	for i, pl := range profileLabels {
		labelsToInsert[fmt.Sprintf("%s\n%s", pl.ProfileUUID, pl.LabelName)] = &profileLabels[i]
		if i > 0 {
			insertBuilder.WriteString(",")
			selectOrDeleteBuilder.WriteString(",")
		}
		insertBuilder.WriteString("(?, ?, ?, ?, ?)")
		selectOrDeleteBuilder.WriteString("(?, ?)")
		selectParams = append(selectParams, pl.ProfileUUID, pl.LabelName)
		insertParams = append(insertParams, pl.ProfileUUID, pl.LabelID, pl.LabelName, pl.Exclude, pl.RequireAll)
		deleteParams = append(deleteParams, pl.ProfileUUID, pl.LabelID)

		setProfileUUIDs[pl.ProfileUUID] = struct{}{}
	}

	// Determine if we need to update the database
	var existingProfileLabels []fleet.ConfigurationProfileLabel
	err = sqlx.SelectContext(ctx, tx, &existingProfileLabels,
		fmt.Sprintf(selectStmt, platformPrefix, platformPrefix, selectOrDeleteBuilder.String()), selectParams...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "selecting existing profile labels")
	}

	updateNeeded := false
	if len(existingProfileLabels) == len(labelsToInsert) {
		for _, existing := range existingProfileLabels {
			toInsert, ok := labelsToInsert[fmt.Sprintf("%s\n%s", existing.ProfileUUID, existing.LabelName)]
			// The fleet.ConfigurationProfileLabel struct has no pointers, so we can use standard cmp.Equal
			if !ok || !cmp.Equal(existing, *toInsert) {
				updateNeeded = true
				break
			}
		}
	} else {
		updateNeeded = true
	}

	if updateNeeded {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(upsertStmt, platformPrefix, insertBuilder.String()), insertParams...)
		if err != nil {
			if isChildForeignKeyError(err) {
				// one of the provided labels doesn't exist
				return false, foreignKey("mdm_configuration_profile_labels", fmt.Sprintf("(profile, label)=(%v)", insertParams))
			}

			return false, ctxerr.Wrap(ctx, err, "setting label associations for profile")
		}
		updatedDB = true
	}

	deleteStmt = fmt.Sprintf(deleteStmt, platformPrefix, selectOrDeleteBuilder.String(), platformPrefix)

	profUUIDs := make([]string, 0, len(setProfileUUIDs)+len(profileUUIDsWithoutLabels))
	for k := range setProfileUUIDs {
		profUUIDs = append(profUUIDs, k)
	}
	profUUIDs = append(profUUIDs, profileUUIDsWithoutLabels...)
	deleteArgs := deleteParams
	deleteArgs = append(deleteArgs, profUUIDs)

	deleteStmt, args, err := sqlx.In(deleteStmt, deleteArgs...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "sqlx.In delete labels for profiles")
	}
	var result sql.Result
	if result, err = tx.ExecContext(ctx, deleteStmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "deleting labels for profiles")
	}
	if result != nil {
		rows, _ := result.RowsAffected()
		updatedDB = updatedDB || rows > 0
	}

	return updatedDB, nil
}

func (ds *Datastore) MDMGetEULAMetadata(ctx context.Context) (*fleet.MDMEULA, error) {
	// Currently, there can only be one EULA in the database, and we're
	// hardcoding it's id to be 1 in order to enforce this restriction.
	stmt := "SELECT name, created_at, token, sha256 FROM eulas WHERE id = 1"
	var eula fleet.MDMEULA
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &eula, stmt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMEULA"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get EULA metadata")
	}
	return &eula, nil
}

func (ds *Datastore) MDMGetEULABytes(ctx context.Context, token string) (*fleet.MDMEULA, error) {
	stmt := "SELECT name, bytes FROM eulas WHERE token = ?"
	var eula fleet.MDMEULA
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &eula, stmt, token); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMEULA"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get EULA bytes")
	}
	return &eula, nil
}

func (ds *Datastore) MDMInsertEULA(ctx context.Context, eula *fleet.MDMEULA) error {
	// We're intentionally hardcoding the id to be 1 because we only want to
	// allow one EULA.
	stmt := `
          INSERT INTO eulas (id, name, bytes, token, sha256)
	  VALUES (1, ?, ?, ?, ?)
	`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, eula.Name, eula.Bytes, eula.Token, eula.Sha256)
	if err != nil {
		if IsDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMEULA", eula.Token))
		}
		return ctxerr.Wrap(ctx, err, "create EULA")
	}

	return nil
}

func (ds *Datastore) MDMDeleteEULA(ctx context.Context, token string) error {
	stmt := "DELETE FROM eulas WHERE token = ?"
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, token)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete EULA")
	}

	deleted, _ := res.RowsAffected()
	if deleted != 1 {
		return ctxerr.Wrap(ctx, notFound("MDMEULA"))
	}
	return nil
}

func (ds *Datastore) GetHostCertAssociationsToExpire(ctx context.Context, expiryDays, limit int) ([]fleet.SCEPIdentityAssociation, error) {
	// TODO(roberto): this is not good because we don't have any indexes on
	// h.uuid, due to time constraints, I'm assuming that this
	// function is called with a relatively low amount of shas
	//
	// Note that we use GROUP BY because we can't guarantee unique entries
	// based on uuid in the hosts table.
	stmt, args, err := sqlx.In(`
SELECT
    h.uuid AS host_uuid,
    ncaa.sha256 AS sha256,
    COALESCE(MAX(hm.fleet_enroll_ref), '') AS enroll_reference,
    ne.enrolled_from_migration,
    ne.type
FROM (
    -- grab only the latest certificate associated with this device
    SELECT
        n1.id,
	n1.sha256,
	n1.cert_not_valid_after,
	n1.renew_command_uuid
    FROM
        nano_cert_auth_associations n1
    WHERE
        n1.sha256 = (
            SELECT
                n2.sha256
            FROM
                nano_cert_auth_associations n2
            WHERE
                n1.id = n2.id
            ORDER BY
                n2.created_at DESC,
                n2.sha256 ASC
            LIMIT 1
        )
) ncaa
JOIN
    hosts h ON h.uuid = ncaa.id
LEFT JOIN
    host_mdm hm ON hm.host_id = h.id
LEFT JOIN
    nano_enrollments ne ON ne.id = ncaa.id
WHERE
    ncaa.cert_not_valid_after BETWEEN '0000-00-00' AND DATE_ADD(CURDATE(), INTERVAL ? DAY)
    AND ncaa.renew_command_uuid IS NULL
    AND ne.enabled = 1
GROUP BY
    host_uuid, ncaa.sha256, ncaa.cert_not_valid_after
ORDER BY
    cert_not_valid_after ASC
LIMIT ?`, expiryDays, limit)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building sqlx.In query")
	}

	var uuids []fleet.SCEPIdentityAssociation
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &uuids, stmt, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get identity certs close to expiry")
	}
	return uuids, nil
}

func (ds *Datastore) SetCommandForPendingSCEPRenewal(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
	if len(assocs) == 0 {
		return nil
	}

	var sb strings.Builder
	args := make([]any, len(assocs)*3)
	for i, assoc := range assocs {
		sb.WriteString("(?, ?, ?),")
		args[i*3] = assoc.HostUUID
		args[i*3+1] = assoc.SHA256
		args[i*3+2] = cmdUUID
	}

	stmt := fmt.Sprintf(`
		INSERT INTO nano_cert_auth_associations (id, sha256, renew_command_uuid) VALUES %s
		ON DUPLICATE KEY UPDATE
			renew_command_uuid = VALUES(renew_command_uuid)
	`, strings.TrimSuffix(sb.String(), ","))

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return fmt.Errorf("failed to update cert associations: %w", err)
		}

		// NOTE: we can't use insertOnDuplicateDidInsert because the
		// LastInsertId check only works tables that have an
		// auto-incrementing primary key. See notes in that function
		// and insertOnDuplicateDidUpdate to understand the mechanism.
		affected, _ := res.RowsAffected()
		if affected == 1 {
			return errors.New("this function can only be used to update existing associations")
		}

		return nil
	})
}

func (ds *Datastore) CleanSCEPRenewRefs(ctx context.Context, hostUUID string) error {
	stmt := `
	UPDATE nano_cert_auth_associations
	SET renew_command_uuid = NULL
	WHERE id = ?`

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning SCEP renew references")
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		return ctxerr.Errorf(ctx, "nano association for host.uuid %s doesn't exist", hostUUID)
	}

	return nil
}

func (ds *Datastore) GetHostMDMProfileInstallStatus(ctx context.Context, hostUUID string, profUUID string) (fleet.MDMDeliveryStatus, error) {
	table, column, err := getTableAndColumnNameForHostMDMProfileUUID(profUUID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting table and column")
	}

	selectStmt := fmt.Sprintf(`
SELECT
	COALESCE(status, ?) as status
	FROM
	%s
WHERE
	operation_type = ?
	AND host_uuid = ?
	AND %s = ?
`, table, column)

	var status fleet.MDMDeliveryStatus
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &status, selectStmt, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall, hostUUID, profUUID); err != nil {
		if err == sql.ErrNoRows {
			return "", notFound("HostMDMProfile").WithMessage("unable to match profile to host")
		}
		return "", ctxerr.Wrap(ctx, err, "get MDM profile status")
	}
	return status, nil
}

func (ds *Datastore) ResendHostMDMProfile(ctx context.Context, hostUUID string, profUUID string) error {
	table, column, err := getTableAndColumnNameForHostMDMProfileUUID(profUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting table and column")
	}

	// update the status to NULL to trigger resending on the next cron run
	updateStmt := fmt.Sprintf(`UPDATE %s SET status = NULL WHERE host_uuid = ? AND %s = ?`, table, column)

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, updateStmt, hostUUID, profUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "resending host MDM profile")
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			// this should never happen, log for debugging
			level.Debug(ds.logger).Log("msg", "resend profile status not updated", "host_uuid", hostUUID, "profile_uuid", profUUID)
		}

		return nil
	})
}

func getTableAndColumnNameForHostMDMProfileUUID(profUUID string) (table, column string, err error) {
	switch {
	case strings.HasPrefix(profUUID, fleet.MDMAppleDeclarationUUIDPrefix):
		return "host_mdm_apple_declarations", "declaration_uuid", nil
	case strings.HasPrefix(profUUID, fleet.MDMAppleProfileUUIDPrefix):
		return "host_mdm_apple_profiles", "profile_uuid", nil
	case strings.HasPrefix(profUUID, fleet.MDMWindowsProfileUUIDPrefix):
		return "host_mdm_windows_profiles", "profile_uuid", nil
	default:
		return "", "", fmt.Errorf("invalid profile UUID prefix %s", profUUID)
	}
}

func (ds *Datastore) AreHostsConnectedToFleetMDM(ctx context.Context, hosts []*fleet.Host) (map[string]bool, error) {
	var (
		appleUUIDs []any
		winUUIDs   []any
	)

	res := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		switch h.Platform {
		case "darwin", "ipados", "ios":
			appleUUIDs = append(appleUUIDs, h.UUID)
		case "windows":
			winUUIDs = append(winUUIDs, h.UUID)
		}
		res[h.UUID] = false
	}

	setConnectedUUIDs := func(stmt string, uuids []any, mp map[string]bool) error {
		var res []string

		if len(uuids) > 0 {
			stmt, args, err := sqlx.In(stmt, uuids)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "building sqlx.In statement")
			}
			err = sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt, args...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "retrieving hosts connected to fleet")
			}
		}

		for _, uuid := range res {
			mp[uuid] = true
		}

		return nil
	}

	// NOTE: if you change any of the conditions in this query, please
	// update the `hostMDMSelect` constant too, which has a
	// `connected_to_fleet` condition, any relevant filters, and the
	// query used in isAppleHostConnectedToFleetMDM.
	const appleStmt = `
	  SELECT ne.id
	  FROM nano_enrollments ne
	    JOIN hosts h ON h.uuid = ne.id
	    JOIN host_mdm hm ON hm.host_id = h.id
	  WHERE ne.id IN (?)
	    AND ne.enabled = 1
	    AND ne.type IN ('Device', 'User Enrollment (Device)')
	    AND hm.enrolled = 1
	`
	if err := setConnectedUUIDs(appleStmt, appleUUIDs, res); err != nil {
		return nil, err
	}

	// NOTE: if you change any of the conditions in this query, please
	// update the `hostMDMSelect` constant too, which has a
	// `connected_to_fleet` condition, and any relevant filters, and the
	// query used in isWindowsHostConnectedToFleetMDM.
	const winStmt = `
	  SELECT mwe.host_uuid
	  FROM mdm_windows_enrollments mwe
	    JOIN hosts h ON h.uuid = mwe.host_uuid
	    JOIN host_mdm hm ON hm.host_id = h.id
	  WHERE mwe.host_uuid IN (?)
	    AND mwe.device_state = '` + microsoft_mdm.MDMDeviceStateEnrolled + `'
	    AND hm.enrolled = 1
	`
	if err := setConnectedUUIDs(winStmt, winUUIDs, res); err != nil {
		return nil, err
	}

	return res, nil
}

func (ds *Datastore) IsHostConnectedToFleetMDM(ctx context.Context, host *fleet.Host) (bool, error) {
	if host.Platform == "windows" {
		return isWindowsHostConnectedToFleetMDM(ctx, ds.reader(ctx), host)
	} else if host.Platform == "darwin" || host.Platform == "ipados" || host.Platform == "ios" {
		return isAppleHostConnectedToFleetMDM(ctx, ds.reader(ctx), host)
	}

	return false, nil
}

func batchSetProfileVariableAssociationsDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	profileVariablesByUUID []fleet.MDMProfileUUIDFleetVariables,
	platform string,
) error {
	if len(profileVariablesByUUID) == 0 {
		return nil
	}

	var platformPrefix string
	switch platform {
	case "darwin":
		platformPrefix = "apple"
	case "windows":
		platformPrefix = "windows"
	default:
		return fmt.Errorf("unsupported platform %s", platform)
	}

	// collect the profile uuids to clear
	profileUUIDsToDelete := make([]string, 0, len(profileVariablesByUUID))
	// small optimization - if there are no variables to insert, we can stop here
	var varsToSet bool
	for _, profVars := range profileVariablesByUUID {
		profileUUIDsToDelete = append(profileUUIDsToDelete, profVars.ProfileUUID)
		if len(profVars.FleetVariables) > 0 {
			varsToSet = true
		}
	}

	// delete variables associated with those profiles
	clearVarsForProfilesStmt := fmt.Sprintf(`DELETE FROM mdm_configuration_profile_variables WHERE %s_profile_uuid IN (?)`, platformPrefix)
	clearVarsForProfilesStmt, args, err := sqlx.In(clearVarsForProfilesStmt, profileUUIDsToDelete)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In delete variables for profiles")
	}
	if _, err := tx.ExecContext(ctx, clearVarsForProfilesStmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting variables for profiles")
	}

	if !varsToSet {
		return nil
	}

	// load fleet variables to map them to their IDs
	type varDef struct {
		ID       uint   `db:"id"`
		Name     string `db:"name"`
		IsPrefix bool   `db:"is_prefix"`
	}

	var varDefs []varDef
	const varsStmt = `SELECT id, name, is_prefix FROM fleet_variables`
	if err := sqlx.SelectContext(ctx, tx, &varDefs, varsStmt); err != nil {
		return fmt.Errorf("failed to load fleet variables: %w", err)
	}

	// map the variables to their IDs (this looks terrible with the nested fors
	// but those are all very small "n" - single-digit vars by profile and same
	// in varDefs, so this is more efficient than building lookup maps).
	type profVarTuple struct {
		ProfileUUID string
		VarID       uint
	}
	profVars := make([]profVarTuple, 0, len(profileVariablesByUUID))
	for _, pv := range profileVariablesByUUID {
		for _, v := range pv.FleetVariables {
			// variables received here do not have the FLEET_VAR_ prefix, but variables
			// in the fleet_variables table do.
			v = "FLEET_VAR_" + v
			for _, def := range varDefs {
				if !def.IsPrefix && def.Name == v {
					profVars = append(profVars, profVarTuple{pv.ProfileUUID, def.ID})
					break
				}
				if def.IsPrefix && strings.HasPrefix(v, def.Name) {
					profVars = append(profVars, profVarTuple{pv.ProfileUUID, def.ID})
					break
				}
			}
		}
	}

	const batchSize = 1000 // number of parameters is this times number of placeholders
	generateValueArgs := func(p profVarTuple) (string, []any) {
		valuePart := "(?, ?),"
		args := []any{p.ProfileUUID, p.VarID}
		return valuePart, args
	}

	executeUpsertBatch := func(valuePart string, args []any) error {
		stmt := fmt.Sprintf(`
			INSERT INTO mdm_configuration_profile_variables (
				%s_profile_uuid,
				fleet_variable_id
			)
			VALUES %s
			ON DUPLICATE KEY UPDATE
				fleet_variable_id = VALUES(fleet_variable_id)
		`, platformPrefix, strings.TrimSuffix(valuePart, ","))

		_, err := tx.ExecContext(ctx, stmt, args...)
		return err
	}

	err = batchProcessDB(profVars, batchSize, generateValueArgs, executeUpsertBatch)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upserting profile variables")
	}
	return nil
}

func (ds *Datastore) BatchResendMDMProfileToHosts(ctx context.Context, profileUUID string, filters fleet.BatchResendMDMProfileFilters) (int64, error) {
	table, column, err := getTableAndColumnNameForHostMDMProfileUUID(profileUUID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "getting table and column")
	}

	// update the status to NULL to trigger resending on the next cron run
	updateStmt := fmt.Sprintf(`UPDATE %s SET status = NULL WHERE %s = ? AND status = ?`, table, column)

	var count int64
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, updateStmt, profileUUID, filters.ProfileStatus)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "resending MDM profile on hosts")
		}
		count, _ = res.RowsAffected()
		return nil
	})
	return count, err
}

func (ds *Datastore) GetMDMConfigProfileStatus(ctx context.Context, profileUUID string) (fleet.MDMConfigProfileStatus, error) {
	switch {
	case strings.HasPrefix(profileUUID, fleet.MDMAppleProfileUUIDPrefix):
		return ds.getAppleMDMConfigProfileStatus(ctx, profileUUID)
	case strings.HasPrefix(profileUUID, fleet.MDMAppleDeclarationUUIDPrefix):
		return ds.getAppleMDMDeclarationStatus(ctx, profileUUID)
	case strings.HasPrefix(profileUUID, fleet.MDMWindowsProfileUUIDPrefix):
		return ds.getWindowsMDMConfigProfileStatus(ctx, profileUUID)
	default:
		return fleet.MDMConfigProfileStatus{}, ctxerr.Wrap(ctx, notFound("ConfigurationProfile").WithName(profileUUID))
	}
}

func (ds *Datastore) getWindowsMDMConfigProfileStatus(ctx context.Context, profileUUID string) (fleet.MDMConfigProfileStatus, error) {
	var counts fleet.MDMConfigProfileStatus

	stmt := `
SELECT
	CASE
		WHEN hmwp.status = :status_failed THEN
			'failed'
		WHEN COALESCE(hmwp.status, :status_pending) = :status_pending THEN
			'pending'
		WHEN hmwp.status = :status_verifying THEN
			'verifying'
		WHEN hmwp.status = :status_verified THEN
			'verified'
		ELSE
			''
	END AS final_status,
	SUM(1) AS count
FROM
	hosts h
	JOIN host_mdm hmdm ON h.id = hmdm.host_id
	JOIN mdm_windows_enrollments mwe ON h.uuid = mwe.host_uuid
	JOIN host_mdm_windows_profiles hmwp ON hmwp.host_uuid = h.uuid
WHERE
	mwe.device_state = :device_state_enrolled AND
	h.platform = 'windows' AND
	hmdm.is_server = 0 AND
	hmdm.enrolled = 1 AND
	hmwp.profile_uuid = :profile_uuid
GROUP BY
	final_status`

	stmt, args, err := sqlx.Named(stmt, map[string]any{
		"status_failed":         fleet.MDMDeliveryFailed,
		"status_pending":        fleet.MDMDeliveryPending,
		"status_verifying":      fleet.MDMDeliveryVerifying,
		"status_verified":       fleet.MDMDeliveryVerified,
		"device_state_enrolled": microsoft_mdm.MDMDeviceStateEnrolled,
		"profile_uuid":          profileUUID,
	})
	if err != nil {
		return counts, ctxerr.Wrap(ctx, err, "prepare arguments with sqlx.Named")
	}

	var rows []statusCounts
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, args...)
	if err != nil {
		return counts, err
	}

	for _, row := range rows {
		switch row.Status {
		case "failed":
			counts.Failed = row.Count
		case "pending":
			counts.Pending = row.Count
		case "verifying":
			counts.Verifying = row.Count
		case "verified":
			counts.Verified = row.Count
		case "":
			level.Debug(ds.logger).Log("msg", fmt.Sprintf("counted %d windows hosts for profile %s with mdm turned on but no profiles", row.Count, profileUUID))
		default:
			return counts, ctxerr.New(ctx, fmt.Sprintf("unexpected mdm windows status count: status=%s, count=%d", row.Status, row.Count))
		}
	}
	return counts, nil
}

func (ds *Datastore) getAppleMDMConfigProfileStatus(ctx context.Context, profileUUID string) (fleet.MDMConfigProfileStatus, error) {
	var counts fleet.MDMConfigProfileStatus

	// NOTE: the case computation of the status must follow the same logic as in
	// sqlJoinMDMAppleProfilesStatus (for non-file-vault, since this is for
	// custom settings).
	stmt := `
SELECT
	COUNT(id) AS count,
	CASE
		WHEN hmap.status = :status_failed THEN
			'failed'
		WHEN COALESCE(hmap.status, :status_pending) = :status_pending THEN
			'pending'
		WHEN hmap.status = :status_verifying THEN
			'verifying'
		WHEN hmap.status = :status_verified THEN
			'verified'
	END AS final_status
FROM
	hosts h
	JOIN host_mdm_apple_profiles hmap ON h.uuid = hmap.host_uuid
WHERE
	platform IN ('darwin', 'ios', 'ipados') AND
	hmap.profile_uuid = :profile_uuid AND
	( hmap.status NOT IN (:status_verified, :status_verifying) OR hmap.operation_type = :operation_install )
GROUP BY
	final_status HAVING final_status IS NOT NULL`

	stmt, args, err := sqlx.Named(stmt, map[string]any{
		"status_failed":     fleet.MDMDeliveryFailed,
		"status_pending":    fleet.MDMDeliveryPending,
		"status_verifying":  fleet.MDMDeliveryVerifying,
		"status_verified":   fleet.MDMDeliveryVerified,
		"operation_install": fleet.MDMOperationTypeInstall,
		"profile_uuid":      profileUUID,
	})
	if err != nil {
		return counts, ctxerr.Wrap(ctx, err, "prepare arguments with sqlx.Named")
	}

	var dest []struct {
		Count  uint   `db:"count"`
		Status string `db:"final_status"`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dest, stmt, args...); err != nil {
		return counts, err
	}

	byStatus := make(map[string]uint)
	for _, s := range dest {
		if _, ok := byStatus[s.Status]; ok {
			return counts, fmt.Errorf("duplicate status %s", s.Status)
		}
		byStatus[s.Status] = s.Count
	}

	for s, c := range byStatus {
		switch fleet.MDMDeliveryStatus(s) {
		case fleet.MDMDeliveryFailed:
			counts.Failed = c
		case fleet.MDMDeliveryPending:
			counts.Pending = c
		case fleet.MDMDeliveryVerifying:
			counts.Verifying = c
		case fleet.MDMDeliveryVerified:
			counts.Verified = c
		default:
			return counts, fmt.Errorf("unknown status %s", s)
		}
	}

	return counts, nil
}

func (ds *Datastore) getAppleMDMDeclarationStatus(ctx context.Context, declUUID string) (fleet.MDMConfigProfileStatus, error) {
	var counts fleet.MDMConfigProfileStatus

	// NOTE: the case computation of the status must follow the same logic as in
	// sqlJoinMDMAppleDeclarationsStatus.
	stmt := `
SELECT
	COUNT(id) AS count,
	CASE
		WHEN hmad.status = :status_failed THEN
			'failed'
		WHEN COALESCE(hmad.status, :status_pending) = :status_pending THEN
			'pending'
		WHEN hmad.status = :status_verifying THEN
			'verifying'
		WHEN hmad.status = :status_verified THEN
			'verified'
	END AS final_status
FROM
	hosts h
	JOIN host_mdm_apple_declarations hmad ON h.uuid = hmad.host_uuid
WHERE
	h.platform IN ('darwin', 'ios', 'ipados') AND
	( hmad.status NOT IN (:status_verified, :status_verifying) OR hmad.operation_type = :operation_install ) AND
	hmad.declaration_uuid = :declaration_uuid
GROUP BY
	final_status HAVING final_status IS NOT NULL`

	stmt, args, err := sqlx.Named(stmt, map[string]any{
		"status_failed":     fleet.MDMDeliveryFailed,
		"status_pending":    fleet.MDMDeliveryPending,
		"status_verifying":  fleet.MDMDeliveryVerifying,
		"status_verified":   fleet.MDMDeliveryVerified,
		"operation_install": fleet.MDMOperationTypeInstall,
		"declaration_uuid":  declUUID,
	})
	if err != nil {
		return counts, ctxerr.Wrap(ctx, err, "prepare arguments with sqlx.Named")
	}

	var dest []struct {
		Count  uint   `db:"count"`
		Status string `db:"final_status"`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dest, stmt, args...); err != nil {
		return counts, err
	}

	byStatus := make(map[string]uint)
	for _, s := range dest {
		if _, ok := byStatus[s.Status]; ok {
			return counts, fmt.Errorf("duplicate status %s", s.Status)
		}
		byStatus[s.Status] = s.Count
	}

	for s, c := range byStatus {
		switch fleet.MDMDeliveryStatus(s) {
		case fleet.MDMDeliveryFailed:
			counts.Failed = c
		case fleet.MDMDeliveryPending:
			counts.Pending = c
		case fleet.MDMDeliveryVerifying:
			counts.Verifying = c
		case fleet.MDMDeliveryVerified:
			counts.Verified = c
		default:
			return counts, fmt.Errorf("unknown status %s", s)
		}
	}

	return counts, nil
}

func (ds *Datastore) IsHostPendingVPPInstallVerification(ctx context.Context, hostUUID string) (bool, error) {
	stmt := `
SELECT EXISTS (
	SELECT 1
    FROM host_mdm_commands hmc
    JOIN hosts h ON hmc.host_id = h.id
    WHERE
		h.uuid = ? AND
		hmc.command_type = ?
) AS exists_flag
`
	var exists bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &exists, stmt, hostUUID, fleet.VerifySoftwareInstallVPPPrefix); err != nil {
		return false, ctxerr.Wrap(ctx, err, "check for acknowledged mdm command by host")
	}
	return exists, nil
}
