// filepath: cmd/fleetctl/generate_gitops_test.go
package fleetctl

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

type MockClient struct {
	IsFree bool
}

func (c *MockClient) GetAppConfig() (*fleet.EnrichedAppConfig, error) {
	b, err := os.ReadFile("./testdata/generateGitops/appConfig.json")
	if err != nil {
		return nil, err
	}
	var appConfig fleet.EnrichedAppConfig
	if err := json.Unmarshal(b, &appConfig); err != nil {
		return nil, err
	}
	if c.IsFree == true {
		appConfig.License.Tier = fleet.TierFree
	}
	return &appConfig, nil
}

func (MockClient) GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error) {
	spec := &fleet.EnrollSecretSpec{
		Secrets: []*fleet.EnrollSecret{
			{
				Secret: "some-secret-number-one",
			},
			{
				Secret: "some-secret-number-two",
			},
		},
	}
	return spec, nil
}

func (MockClient) ListTeams(query string) ([]fleet.Team, error) {
	b, err := os.ReadFile("./testdata/generateGitops/teamConfig.json")
	if err != nil {
		return nil, err
	}
	var config fleet.TeamConfig
	if err := json.Unmarshal(b, &config); err != nil {
		return nil, err
	}
	teams := []fleet.Team{
		{
			ID:     1,
			Name:   "Team A",
			Config: config,
			Secrets: []*fleet.EnrollSecret{
				{
					Secret: "some-team-secret",
				},
			},
		},
	}
	return teams, nil
}

func (MockClient) ListScripts(query string) ([]*fleet.Script, error) {
	switch query {
	case "team_id=1":
		return []*fleet.Script{{
			ID:              2,
			TeamID:          ptr.Uint(1),
			Name:            "Script B.ps1",
			ScriptContentID: 2,
		}}, nil
	case "team_id=0":
		return []*fleet.Script{{
			ID:              3,
			TeamID:          ptr.Uint(0),
			Name:            "Script Z.ps1",
			ScriptContentID: 3,
		}}, nil
	case "team_id=2", "team_id=3", "team_id=4", "team_id=5":
		return nil, nil
	case "team_id=6":
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (MockClient) ListConfigurationProfiles(teamID *uint) ([]*fleet.MDMConfigProfilePayload, error) {
	if teamID == nil {
		return []*fleet.MDMConfigProfilePayload{
			{
				ProfileUUID: "global-macos-mobileconfig-profile-uuid",
				Name:        "Global MacOS MobileConfig Profile",
				Platform:    "darwin",
				Identifier:  "com.example.global-macos-mobileconfig-profile",
				LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{
					LabelName: "Label A",
				}, {
					LabelName: "Label B",
				}},
			},
			{
				ProfileUUID: "global-macos-json-profile-uuid",
				Name:        "Global MacOS JSON Profile",
				Platform:    "darwin",
				Identifier:  "com.example.global-macos-json-profile",
				LabelsExcludeAny: []fleet.ConfigurationProfileLabel{{
					LabelName: "Label C",
				}},
			},
			{
				ProfileUUID: "global-windows-profile-uuid",
				Name:        "Global Windows Profile",
				Platform:    "windows",
				Identifier:  "com.example.global-windows-profile",
				LabelsIncludeAny: []fleet.ConfigurationProfileLabel{{
					LabelName: "Label D",
				}},
			},
		}, nil
	}
	if *teamID == 1 {
		return []*fleet.MDMConfigProfilePayload{
			{
				ProfileUUID: "test-mobileconfig-profile-uuid",
				Name:        "Team MacOS MobileConfig Profile",
				Platform:    "darwin",
				Identifier:  "com.example.team-macos-mobileconfig-profile",
			},
		}, nil
	}
	if *teamID == 0 || *teamID == 2 || *teamID == 3 || *teamID == 4 || *teamID == 5 {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected team ID: %v", *teamID)
}

func (MockClient) GetScriptContents(scriptID uint) ([]byte, error) {
	if scriptID == 2 {
		return []byte("pop goes the weasel!"), nil
	}
	if scriptID == 3 {
		return []byte("#!/usr/bin/env pwsh\necho \"Hello from Script B!\""), nil
	}
	return nil, errors.New("script not found")
}

func (MockClient) GetProfileContents(profileID string) ([]byte, error) {
	switch profileID {
	case "global-macos-mobileconfig-profile-uuid":
		return []byte("<xml>global macos mobileconfig profile</xml>"), nil
	case "global-macos-json-profile-uuid":
		return []byte(`{"profile": "global macos json profile"}`), nil
	case "global-windows-profile-uuid":
		return []byte("<xml>global windows profile</xml>"), nil
	case "test-mobileconfig-profile-uuid":
		return []byte("<xml>test mobileconfig profile</xml>"), nil
	}
	return nil, errors.New("profile not found")
}

func (MockClient) GetTeam(teamID uint) (*fleet.Team, error) {
	if teamID == 1 {
		b, err := os.ReadFile("./testdata/generateGitops/teamConfig.json")
		if err != nil {
			return nil, err
		}
		var config fleet.TeamConfig
		if err := json.Unmarshal(b, &config); err != nil {
			return nil, err
		}
		return &fleet.Team{
			ID:     1,
			Name:   "Test Team",
			Config: config,
			Secrets: []*fleet.EnrollSecret{
				{
					Secret: "some-team-secret",
				},
			},
		}, nil
	}

	return nil, errors.New("team not found")
}

func (MockClient) ListSoftwareTitles(query string) ([]fleet.SoftwareTitleListResult, error) {
	switch query {
	case "available_for_install=1&team_id=1":
		return []fleet.SoftwareTitleListResult{
			{
				ID:         1,
				Name:       "My Software Package",
				HashSHA256: ptr.String("software-package-hash"),
				SoftwarePackage: &fleet.SoftwarePackageOrApp{
					Name:     "my-software.pkg",
					Platform: "darwin",
					Version:  "13.37",
				},
			},
			{
				ID:   2,
				Name: "My App Store App",
				AppStoreApp: &fleet.SoftwarePackageOrApp{
					AppStoreID: "com.example.team-software",
				},
				HashSHA256: ptr.String("app-store-app-hash"),
			},
		}, nil
	case "available_for_install=1&team_id=0":
		return []fleet.SoftwareTitleListResult{}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (MockClient) GetPolicies(teamID *uint) ([]*fleet.Policy, error) {
	if teamID == nil {
		return []*fleet.Policy{
			{
				PolicyData: fleet.PolicyData{
					ID:          1,
					Name:        "Global Policy",
					Query:       "SELECT * FROM global_policy WHERE id = 1",
					Resolution:  ptr.String("Do a global thing"),
					Description: "This is a global policy",
					Platform:    "darwin",
					LabelsIncludeAny: []fleet.LabelIdent{{
						LabelName: "Label A",
					}, {
						LabelName: "Label B",
					}},
					ConditionalAccessEnabled: true,
				},
				InstallSoftware: &fleet.PolicySoftwareTitle{
					SoftwareTitleID: 1,
				},
			},
		}, nil
	}
	return []*fleet.Policy{
		{
			PolicyData: fleet.PolicyData{
				ID:                       1,
				Name:                     "Team Policy",
				Query:                    "SELECT * FROM team_policy WHERE id = 1",
				Resolution:               ptr.String("Do a team thing"),
				Description:              "This is a team policy",
				Platform:                 "linux,windows",
				ConditionalAccessEnabled: true,
			},
			RunScript: &fleet.PolicyScript{
				ID: 1,
			},
		},
	}, nil
}

func (MockClient) GetQueries(teamID *uint, name *string) ([]fleet.Query, error) {
	if teamID == nil {
		return []fleet.Query{
			{
				ID:                 1,
				Name:               "Global Query",
				Query:              "SELECT * FROM global_query WHERE id = 1",
				Description:        "This is a global query",
				Platform:           "darwin",
				Interval:           3600,
				ObserverCanRun:     true,
				AutomationsEnabled: true,
				LabelsIncludeAny: []fleet.LabelIdent{{
					LabelName: "Label A",
				}, {
					LabelName: "Label B",
				}},
				MinOsqueryVersion: "1.2.3",
				Logging:           "stdout",
			},
		}, nil
	}
	return []fleet.Query{
		{
			ID:                 1,
			Name:               "Team Query",
			Query:              "SELECT * FROM team_query WHERE id = 1",
			Description:        "This is a team query",
			Platform:           "linux,windows",
			Interval:           1800,
			ObserverCanRun:     false,
			AutomationsEnabled: true,
			MinOsqueryVersion:  "4.5.6",
			Logging:            "stderr",
		},
	}, nil
}

//nolint:gocritic // ignore captLocal
func (MockClient) GetSoftwareTitleByID(ID uint, teamID *uint) (*fleet.SoftwareTitle, error) {
	switch ID {
	case 1:
		if *teamID != 1 {
			return nil, errors.New("team ID mismatch")
		}
		return &fleet.SoftwareTitle{
			ID: 1,
			SoftwarePackage: &fleet.SoftwareInstaller{
				LabelsIncludeAny: []fleet.SoftwareScopeLabel{{
					LabelName: "Label A",
				}, {
					LabelName: "Label B",
				}},
				PreInstallQuery:   "SELECT * FROM pre_install_query",
				InstallScript:     "foo",
				PostInstallScript: "bar",
				UninstallScript:   "baz",
				SelfService:       true,
				Platform:          "darwin",
				URL:               "https://example.com/download/my-software.pkg",
			},
		}, nil
	case 2:
		if *teamID != 1 {
			return nil, errors.New("team ID mismatch")
		}
		return &fleet.SoftwareTitle{
			ID: 2,
			AppStoreApp: &fleet.VPPAppStoreApp{
				LabelsExcludeAny: []fleet.SoftwareScopeLabel{{
					LabelName: "Label C",
				}, {
					LabelName: "Label D",
				}},
			},
		}, nil
	default:
		return nil, errors.New("software title not found")
	}
}

func (MockClient) GetLabels() ([]*fleet.LabelSpec, error) {
	return []*fleet.LabelSpec{{
		Name:                "Label A",
		Platform:            "linux,macos",
		Description:         "Label A description",
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		Query:               "SELECT * FROM osquery_info",
	}, {
		Name:                "Label B",
		Description:         "Label B description",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
		Hosts:               []string{"host1", "host2"},
	}, {
		Name:                "Label C",
		Description:         "Label C description",
		LabelMembershipType: fleet.LabelMembershipTypeHostVitals,
		HostVitalsCriteria:  ptr.RawMessage(json.RawMessage(`{"vital": "end_user_idp_group", "value": "some-group"}`)),
	}}, nil
}

func (MockClient) Me() (*fleet.User, error) {
	return &fleet.User{
		ID:         1,
		Name:       "Test User",
		Email:      "test@example.com",
		GlobalRole: ptr.String("admin"),
	}, nil
}

func (MockClient) GetEULAMetadata() (*fleet.MDMEULA, error) {
	return &fleet.MDMEULA{
		Name:  "test.pdf",
		Token: "test-eula-token",
	}, nil
}

func (MockClient) GetEULAContent(token string) ([]byte, error) {
	return []byte("This is the EULA content."), nil
}

func (MockClient) GetSetupExperienceSoftware(teamID uint) ([]fleet.SoftwareTitleListResult, error) {
	if teamID == 1 {
		return []fleet.SoftwareTitleListResult{
			{
				ID:         1,
				Name:       "My Software Package",
				HashSHA256: ptr.String("software-package-hash"),
				SoftwarePackage: &fleet.SoftwarePackageOrApp{
					InstallDuringSetup: ptr.Bool(true),
					Name:               "my-software.pkg",
					Platform:           "darwin",
					Version:            "13.37",
				},
			},
		}, nil
	}
	if teamID == 2 {
		return []fleet.SoftwareTitleListResult{
			{
				ID:         1,
				Name:       "My Software Package",
				HashSHA256: ptr.String("software-package-hash"),
				SoftwarePackage: &fleet.SoftwarePackageOrApp{
					InstallDuringSetup: ptr.Bool(false),
					Name:               "my-software.pkg",
					Platform:           "darwin",
					Version:            "13.37",
				},
			},
			{
				ID:         2,
				Name:       "My Other Software Package",
				HashSHA256: ptr.String("software-package-hash"),
			},
		}, nil
	}
	if teamID == 0 || teamID == 3 || teamID == 4 || teamID == 5 {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected team ID: %d", teamID)
}

func (MockClient) GetBootstrapPackageMetadata(teamID uint, forUpdate bool) (*fleet.MDMAppleBootstrapPackage, error) {
	if teamID == 3 {
		return &fleet.MDMAppleBootstrapPackage{
			Name: "Bootstrap Package for Team 1",
		}, nil
	}
	if teamID == 0 || teamID == 1 || teamID == 2 || teamID == 4 || teamID == 5 {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected team ID: %d", teamID)
}

func (MockClient) GetSetupExperienceScript(teamID uint) (*fleet.Script, error) {
	if teamID == 4 {
		return &fleet.Script{
			Name: "Setup Experience Script for Team 1",
		}, nil
	}
	if teamID == 0 || teamID == 1 || teamID == 2 || teamID == 3 || teamID == 5 {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected team ID: %d", teamID)
}

func (MockClient) GetAppleMDMEnrollmentProfile(teamID uint) (*fleet.MDMAppleSetupAssistant, error) {
	if teamID == 5 {
		return &fleet.MDMAppleSetupAssistant{
			Name: "Apple MDM Enrollment Profile for Team 1",
		}, nil
	}
	if teamID == 0 || teamID == 1 || teamID == 2 || teamID == 3 || teamID == 4 {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected team ID: %d", teamID)
}

func compareDirs(t *testing.T, sourceDir, targetDir string) {
	err := filepath.WalkDir(sourceDir, func(srcPath string, d os.DirEntry, walkErr error) error {
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, srcPath)
		require.NoError(t, err, "Error getting relative path: %v", err)

		tgtPath := filepath.Join(targetDir, relPath)

		targetInfo, err := os.Stat(tgtPath)
		require.NoError(t, err, "Error getting target file info: %v", err)
		require.False(t, targetInfo.IsDir(), "Expected file but found directory: %s", tgtPath)

		srcData, err := os.ReadFile(srcPath)
		require.NoError(t, err, "Error reading source file: %v", err)

		tgtData, err := os.ReadFile(tgtPath)
		require.NoError(t, err, "Error reading target file: %v", err)

		require.Equal(t, string(srcData), string(tgtData), "File contents do not match for %s", relPath)

		return nil
	})
	if err != nil {
		t.Fatalf("Error walking source directory: %v", err)
	}
}

func TestGenerateGitops(t *testing.T) {
	fleetClient := &MockClient{}
	action := createGenerateGitopsAction(fleetClient)
	buf := new(bytes.Buffer)
	tempDir := os.TempDir() + "/" + uuid.New().String()
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.String("dir", tempDir, "")

	cliContext := cli.NewContext(&cli.App{
		Name:      "test",
		Usage:     "test",
		Writer:    buf,
		ErrWriter: buf,
	}, flagSet, nil)
	err := action(cliContext)
	require.NoError(t, err, buf.String())

	compareDirs(t, "./testdata/generateGitops/test_dir_premium", tempDir)

	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	})
}

func TestGenerateGitopsFree(t *testing.T) {
	fleetClient := &MockClient{}
	fleetClient.IsFree = true
	action := createGenerateGitopsAction(fleetClient)
	buf := new(bytes.Buffer)
	tempDir := os.TempDir() + "/" + uuid.New().String()
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.String("dir", tempDir, "")

	cliContext := cli.NewContext(&cli.App{
		Name:      "test",
		Usage:     "test",
		Writer:    buf,
		ErrWriter: buf,
	}, flagSet, nil)
	err := action(cliContext)
	require.NoError(t, err, buf.String())

	compareDirs(t, "./testdata/generateGitops/test_dir_free", tempDir)

	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	})
}

func TestGenerateOrgSettings(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	orgSettingsRaw, err := cmd.generateOrgSettings()
	require.NoError(t, err)
	require.NotNil(t, orgSettingsRaw)
	var orgSettings map[string]interface{}
	b, err := yaml.Marshal(orgSettingsRaw)
	require.NoError(t, err)
	fmt.Println("Org settings raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &orgSettings)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedOrgSettings.yaml")
	require.NoError(t, err)
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, orgSettings)
}

func TestGeneratedOrgSettingsNoSSO(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	appConfig.SSOSettings = nil

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	orgSettingsRaw, err := cmd.generateOrgSettings()
	require.NoError(t, err)
	require.NotNil(t, orgSettingsRaw)
	var orgSettings map[string]any
	b, err := yaml.Marshal(orgSettingsRaw)
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &orgSettings)
	require.NoError(t, err)
}

func TestGenerateOrgSettingsInsecure(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.Bool("insecure", true, "Output sensitive information in plaintext.")
	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, flagSet, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	orgSettingsRaw, err := cmd.generateOrgSettings()
	require.NoError(t, err)
	require.NotNil(t, orgSettingsRaw)
	var orgSettings map[string]interface{}
	b, err := yaml.Marshal(orgSettingsRaw)
	require.NoError(t, err)
	fmt.Println("Org settings raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &orgSettings)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedOrgSettings-insecure.yaml")
	require.NoError(t, err)
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, orgSettings)
}

func TestGenerateTeamSettings(t *testing.T) {
	// Get the test team.
	fleetClient := &MockClient{}
	team, err := fleetClient.GetTeam(1)
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    nil,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	TeamSettingsRaw, err := cmd.generateTeamSettings("team.yml", team)
	require.NoError(t, err)
	require.NotNil(t, TeamSettingsRaw)
	var TeamSettings map[string]interface{}
	b, err := yaml.Marshal(TeamSettingsRaw)
	require.NoError(t, err)
	fmt.Println("Team settings raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &TeamSettings)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamSettings.yaml")
	require.NoError(t, err)
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, TeamSettings)
}

func TestGenerateTeamSettingsInsecure(t *testing.T) {
	// Get the test team.
	fleetClient := &MockClient{}
	team, err := fleetClient.GetTeam(1)
	require.NoError(t, err)

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.Bool("insecure", true, "Output sensitive information in plaintext.")
	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, flagSet, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    nil,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	TeamSettingsRaw, err := cmd.generateTeamSettings("team.yml", team)
	require.NoError(t, err)
	require.NotNil(t, TeamSettingsRaw)
	var TeamSettings map[string]interface{}
	b, err := yaml.Marshal(TeamSettingsRaw)
	require.NoError(t, err)
	fmt.Println("Team settings raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &TeamSettings)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamSettings-insecure.yaml")
	require.NoError(t, err)
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, TeamSettings)
}

// For the purpose of testing macos_setup generation,
// Team 1 has setup experience software with InstallDuringSetup enabled,
// Team 2 has setup experience software with InstallDuringSetup disabled,
// Team 3 has a bootstrap package,
// Team 4 has a setup experience script,
// Team 5 has an Apple MDM enrollment profile.
func TestGenerateControls(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
		ScriptList:   make(map[uint]string),
	}

	// Generate global controls.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	mdmConfig := fleet.TeamMDM{
		EnableDiskEncryption: appConfig.MDM.EnableDiskEncryption.Value,
		MacOSUpdates:         appConfig.MDM.MacOSUpdates,
		IOSUpdates:           appConfig.MDM.IOSUpdates,
		IPadOSUpdates:        appConfig.MDM.IPadOSUpdates,
		WindowsUpdates:       appConfig.MDM.WindowsUpdates,
		MacOSSetup:           appConfig.MDM.MacOSSetup,
	}
	controlsRaw, err := cmd.generateControls(nil, "", &mdmConfig)
	require.NoError(t, err)
	require.NotNil(t, controlsRaw)
	var controls map[string]interface{}
	b, err := yaml.Marshal(controlsRaw)
	require.NoError(t, err)
	fmt.Println("Controls raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &controls)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedGlobalControls.yaml")
	require.NoError(t, err)
	var expectedControls map[string]interface{}
	err = yaml.Unmarshal(b, &expectedControls)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedControls, controls)

	if fileContents, ok := cmd.FilesToWrite["lib/profiles/global-macos-mobileconfig-profile.mobileconfig"]; ok {
		require.Equal(t, "<xml>global macos mobileconfig profile</xml>", fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}

	if fileContents, ok := cmd.FilesToWrite["lib/profiles/global-macos-json-profile.json"]; ok {
		require.Equal(t, `{"profile": "global macos json profile"}`, fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}

	if fileContents, ok := cmd.FilesToWrite["lib/profiles/global-windows-profile.xml"]; ok {
		require.Equal(t, "<xml>global windows profile</xml>", fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}

	// Generate controls for no-team, sending in an MDM config with "EndUserAuthentication" disabled.
	// Mocks for no team don't return any scripts, bootstrap, software, or profiles,
	// so it should _not_ generate a macos_setup config.
	mdmConfig = fleet.TeamMDM{
		MacOSSetup: fleet.MacOSSetup{
			EnableEndUserAuthentication: false,
		},
	}
	controlsRaw, err = cmd.generateControls(ptr.Uint(0), "no_team", &mdmConfig)
	require.NoError(t, err)
	// Check that the controls do not contain a macos_setup section
	_, ok := controlsRaw["macos_setup"]
	require.False(t, ok, "Expected no macos_setup section for no-team controls")

	// Try that again, but with an MDM config that has "EndUserAuthentication" enabled.
	mdmConfig = fleet.TeamMDM{
		MacOSSetup: fleet.MacOSSetup{
			EnableEndUserAuthentication: true,
		},
	}
	controlsRaw, err = cmd.generateControls(ptr.Uint(0), "no_team", &mdmConfig)
	require.NoError(t, err)
	// Check that the controls do contain a macos_setup section
	verifyControlsHasMacosSetup(t, controlsRaw)

	// Generate controls for a team.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	// Note that this team has setup experience software, so we expect a macos_setup section.
	controlsRaw, err = cmd.generateControls(ptr.Uint(1), "some_team", nil)
	require.NoError(t, err)
	require.NotNil(t, controlsRaw)
	b, err = yaml.Marshal(controlsRaw)
	require.NoError(t, err)
	fmt.Println("Controls raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &controls)
	require.NoError(t, err)

	// Get the expected controls YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamControls.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &expectedControls)
	require.NoError(t, err)

	if fileContents, ok := cmd.FilesToWrite["lib/some_team/profiles/team-macos-mobileconfig-profile.mobileconfig"]; ok {
		require.Equal(t, "<xml>test mobileconfig profile</xml>", fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}

	if fileContents, ok := cmd.FilesToWrite["lib/some_team/scripts/Script B.ps1"]; ok {
		require.Equal(t, "pop goes the weasel!", fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}

	// Compare.
	require.Equal(t, expectedControls, controls)

	// Generate controls for a team with software but none that installs during setup.
	controlsRaw, err = cmd.generateControls(ptr.Uint(2), "some_team", nil)
	require.NoError(t, err)
	require.NotNil(t, controlsRaw)
	require.False(t, ok, "Expected no macos_setup section for no-team controls")

	// Generate controls for a team with a bootstrap pacakge.
	controlsRaw, err = cmd.generateControls(ptr.Uint(3), "some_team", nil)
	require.NoError(t, err)
	verifyControlsHasMacosSetup(t, controlsRaw)

	// Generate controls for a team with a setup experience script.
	controlsRaw, err = cmd.generateControls(ptr.Uint(4), "some_team", nil)
	require.NoError(t, err)
	verifyControlsHasMacosSetup(t, controlsRaw)

	// Generate controls for a team with a setup experience profile.
	controlsRaw, err = cmd.generateControls(ptr.Uint(5), "some_team", nil)
	require.NoError(t, err)
	verifyControlsHasMacosSetup(t, controlsRaw)
}

func TestGenerateSoftware(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
		SoftwareList: make(map[uint]Software),
	}

	softwareRaw, err := cmd.generateSoftware("team.yml", 1, "some-team")
	require.NoError(t, err)
	require.NotNil(t, softwareRaw)
	var software map[string]interface{}
	b, err := yaml.Marshal(softwareRaw)
	require.NoError(t, err)
	fmt.Println("software raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &software)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamSoftware.yaml")
	require.NoError(t, err)
	var expectedSoftware map[string]interface{}
	err = yaml.Unmarshal(b, &expectedSoftware)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedSoftware, software)

	if fileContents, ok := cmd.FilesToWrite["lib/some-team/scripts/my-software-package-darwin-install"]; ok {
		require.Equal(t, "foo", fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}

	if fileContents, ok := cmd.FilesToWrite["lib/some-team/scripts/my-software-package-darwin-postinstall"]; ok {
		require.Equal(t, "bar", fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}

	if fileContents, ok := cmd.FilesToWrite["lib/some-team/scripts/my-software-package-darwin-uninstall"]; ok {
		require.Equal(t, "baz", fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}

	if fileContents, ok := cmd.FilesToWrite["lib/some-team/queries/my-software-package-darwin-preinstallquery.yml"]; ok {
		require.Equal(t, []map[string]interface{}{{
			"query": "SELECT * FROM pre_install_query",
		}}, fileContents)
	} else {
		t.Fatalf("Expected file not found")
	}
}

func TestGeneratePolicies(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
		SoftwareList: map[uint]Software{
			1: {
				Hash:    "team-software-hash",
				Comment: "__TEAM_SOFTWARE_COMMENT_TOKEN__",
			},
		},
		ScriptList: map[uint]string{
			1: "/path/to/script1.sh",
		},
	}

	policiesRaw, err := cmd.generatePolicies(nil, "default.yml")
	require.NoError(t, err)
	require.NotNil(t, policiesRaw)
	var policies []map[string]interface{}
	b, err := yaml.Marshal(policiesRaw)
	require.NoError(t, err)
	fmt.Println("policies raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &policies)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedGlobalPolicies.yaml")
	require.NoError(t, err)
	var expectedPolicies []map[string]interface{}
	err = yaml.Unmarshal(b, &expectedPolicies)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedPolicies, policies)

	// Generate policies for a team.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	policiesRaw, err = cmd.generatePolicies(ptr.Uint(1), "some_team")
	require.NoError(t, err)
	require.NotNil(t, policiesRaw)
	b, err = yaml.Marshal(policiesRaw)
	require.NoError(t, err)
	fmt.Println("policies raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &policies)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamPolicies.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &expectedPolicies)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedPolicies, policies)
}

func TestGenerateQueries(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	queriesRaw, err := cmd.generateQueries(nil)
	require.NoError(t, err)
	require.NotNil(t, queriesRaw)
	var queries []map[string]interface{}
	b, err := yaml.Marshal(queriesRaw)
	require.NoError(t, err)
	fmt.Println("queries raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &queries)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedGlobalQueries.yaml")
	require.NoError(t, err)
	var expectedQueries []map[string]interface{}
	err = yaml.Unmarshal(b, &expectedQueries)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedQueries, queries)

	// Generate queries for a team.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	queriesRaw, err = cmd.generateQueries(ptr.Uint(1))
	require.NoError(t, err)
	require.NotNil(t, queriesRaw)
	b, err = yaml.Marshal(queriesRaw)
	require.NoError(t, err)
	fmt.Println("queries raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &queries)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamQueries.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &expectedQueries)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedQueries, queries)
}

func TestGenerateLabels(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	labelsRaw, err := cmd.generateLabels()
	require.NoError(t, err)
	require.NotNil(t, labelsRaw)
	var labels []map[string]interface{}
	b, err := yaml.Marshal(labelsRaw)
	require.NoError(t, err)
	fmt.Println("labels raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &labels)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedLabels.yaml")
	require.NoError(t, err)
	var expectedlabels []map[string]interface{}
	err = yaml.Unmarshal(b, &expectedlabels)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedlabels, labels)
}

func verifyControlsHasMacosSetup(t *testing.T, controlsRaw map[string]interface{}) {
	macosSetup, ok := controlsRaw["macos_setup"].(string)
	require.True(t, ok, "Expected macos_setup section to be a string")
	require.Equal(t, macosSetup, "TODO: update with your macos_setup configuration")
}

func TestGenerateControlsAndMDMWithoutMDMEnabledAndConfigured(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)
	appConfig.MDM.EnabledAndConfigured = false
	appConfig.MDM.WindowsEnabledAndConfigured = false
	appConfig.MDM.AppleBusinessManager = optjson.Slice[fleet.MDMAppleABMAssignmentInfo]{}
	appConfig.MDM.AppleServerURL = ""
	appConfig.MDM.EndUserAuthentication = fleet.MDMEndUserAuthentication{}
	appConfig.MDM.VolumePurchasingProgram = optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo]{}

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
		ScriptList:   make(map[uint]string),
	}
	// teamId=6 is unhandled for MDM APIs to make sure the APIs to get MDM stuff
	// aren't called if MDM is not enabled and configured.
	controlsRaw, err := cmd.generateControls(ptr.Uint(6), "some_team", nil)
	require.NoError(t, err)
	require.Contains(t, controlsRaw, "scripts") // Uploading scripts does not require MDM turned on.
	require.Empty(t, controlsRaw["scripts"])

	// teamId=6 is unhandled for MDM APIs to make sure the APIs to get MDM stuff
	// aren't called if MDM is not enabled and configured.
	mdmRaw, err := cmd.generateMDM(&appConfig.MDM)
	require.NoError(t, err)
	// Verify all keys are set to empty.
	for _, key := range []string{
		"apple_business_manager",
		"apple_server_url",
		"end_user_authentication",
		"end_user_license_agreement",
		"volume_purchasing_program",
	} {
		require.Contains(t, mdmRaw, key)
		require.Empty(t, mdmRaw[key])
	}
}
