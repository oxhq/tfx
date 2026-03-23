package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var errConfigNotFound = errors.New("tfx config not found")

type wrapperConfig struct {
	Project         string           `yaml:"project"`
	Description     string           `yaml:"description"`
	Lanes           []string         `yaml:"lanes"`
	DefaultLane     string           `yaml:"default_lane"`
	RequireApproval bool             `yaml:"require_approval"`
	SecretPrompt    string           `yaml:"secret_prompt"`
	SecretEnv       string           `yaml:"secret_env"`
	WorkingDir      string           `yaml:"working_dir"`
	DefaultFlow     string           `yaml:"default_flow"`
	DefaultProfile  string           `yaml:"default_profile"`
	BeforeAll       []wrapperStep    `yaml:"before_all"`
	AfterAll        []wrapperStep    `yaml:"after_all"`
	OnFailure       []wrapperStep    `yaml:"on_failure"`
	Steps           []wrapperStep    `yaml:"steps"`
	Flows           []wrapperProfile `yaml:"flows"`
	Profiles        []wrapperProfile `yaml:"profiles"`
}

type wrapperProfile struct {
	Name            string        `yaml:"name"`
	Description     string        `yaml:"description"`
	Lanes           []string      `yaml:"lanes"`
	DefaultLane     string        `yaml:"default_lane"`
	RequireApproval *bool         `yaml:"require_approval"`
	SecretPrompt    string        `yaml:"secret_prompt"`
	SecretEnv       string        `yaml:"secret_env"`
	WorkingDir      string        `yaml:"working_dir"`
	BeforeAll       []wrapperStep `yaml:"before_all"`
	AfterAll        []wrapperStep `yaml:"after_all"`
	OnFailure       []wrapperStep `yaml:"on_failure"`
	Steps           []wrapperStep `yaml:"steps"`
}

type wrapperPlan struct {
	Name            string
	Description     string
	Lanes           []string
	DefaultLane     string
	RequireApproval bool
	SecretPrompt    string
	SecretEnv       string
	WorkingDir      string
	BeforeAll       []wrapperStep
	AfterAll        []wrapperStep
	OnFailure       []wrapperStep
	Steps           []wrapperStep
}

type wrapperStep struct {
	ID              string                `yaml:"id"`
	Title           string                `yaml:"title"`
	Detail          string                `yaml:"detail"`
	Command         []string              `yaml:"command"`
	Dir             string                `yaml:"dir"`
	Lanes           []string              `yaml:"lanes"`
	Env             map[string]string     `yaml:"env"`
	Artifacts       []ArtifactDeclaration `yaml:"artifacts"`
	ContinueOnError bool                  `yaml:"continue_on_error"`
}

func loadWrapperConfig() (cfg wrapperConfig, cwd string, configPath string, err error) {
	cwd, err = os.Getwd()
	if err != nil {
		return wrapperConfig{}, "", "", err
	}
	return loadWrapperConfigFrom(cwd)
}

func loadWrapperConfigFrom(
	cwd string,
) (cfg wrapperConfig, resolvedCWD string, configPath string, err error) {
	configPath, err = findConfigPath(cwd)
	if err != nil {
		if !errors.Is(err, errConfigNotFound) {
			return wrapperConfig{}, "", "", err
		}
		cfg = defaultWrapperConfig(cwd)
		return cfg, cwd, "", cfg.sanitize(cfg.WorkingDir)
	}

	// #nosec G304 -- configPath is discovered from a fixed set of filenames in the workspace tree.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return wrapperConfig{}, "", "", fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return wrapperConfig{}, "", "", fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.sanitize(filepath.Dir(configPath)); err != nil {
		return wrapperConfig{}, "", "", err
	}

	return cfg, cwd, configPath, nil
}

func findConfigPath(start string) (string, error) {
	candidates := []string{"tfx.yaml", "tfx.yml", ".tfx.yaml", ".tfx.yml"}

	dir := start
	for {
		for _, name := range candidates {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errConfigNotFound
}

func defaultWrapperConfig(cwd string) wrapperConfig {
	rootDir := cwd
	if goMod, err := findNearestFile(cwd, "go.mod"); err == nil {
		rootDir = filepath.Dir(goMod)
	}

	project := filepath.Base(rootDir)
	if project == "." || project == string(filepath.Separator) || project == "" {
		project = "workspace"
	}

	if _, err := os.Stat(filepath.Join(rootDir, "go.mod")); err == nil {
		return wrapperConfig{
			Project:         project,
			Description:     "Fallback Go module pipeline",
			Lanes:           []string{"preview", "canary", "production"},
			DefaultLane:     "preview",
			RequireApproval: true,
			SecretPrompt:    "Release token",
			SecretEnv:       "TFX_SECRET",
			WorkingDir:      rootDir,
			Steps: []wrapperStep{
				{
					ID:      "test",
					Title:   "Run tests",
					Detail:  "Execute go test ./...",
					Command: []string{"go", "test", "./..."},
				},
				{
					ID:      "build",
					Title:   "Build project",
					Detail:  "Execute go build ./...",
					Command: []string{"go", "build", "./..."},
				},
				{
					ID:      "vet",
					Title:   "Vet project",
					Detail:  "Execute go vet ./...",
					Command: []string{"go", "vet", "./..."},
					Lanes:   []string{"canary", "production"},
				},
			},
		}
	}

	return wrapperConfig{
		Project:         project,
		Description:     "Fallback command runner",
		Lanes:           []string{"preview", "canary", "production"},
		DefaultLane:     "preview",
		RequireApproval: false,
		SecretPrompt:    "",
		SecretEnv:       "",
		WorkingDir:      rootDir,
		Steps: []wrapperStep{
			{
				ID:      "pwd",
				Title:   "Show working directory",
				Detail:  "Execute pwd",
				Command: []string{"pwd"},
			},
			{
				ID:      "list",
				Title:   "List workspace",
				Detail:  "Execute ls",
				Command: []string{"ls"},
			},
		},
	}
}

func findNearestFile(start string, name string) (string, error) {
	dir := start
	for {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}

func (cfg *wrapperConfig) sanitize(baseDir string) error {
	if cfg.Project == "" {
		cfg.Project = filepath.Base(baseDir)
	}
	cfg.Description = strings.TrimSpace(cfg.Description)
	cfg.SecretPrompt = strings.TrimSpace(cfg.SecretPrompt)
	cfg.SecretEnv = strings.TrimSpace(cfg.SecretEnv)
	cfg.DefaultFlow = strings.TrimSpace(cfg.DefaultFlow)
	cfg.DefaultProfile = strings.TrimSpace(cfg.DefaultProfile)

	if cfg.WorkingDir == "" {
		cfg.WorkingDir = baseDir
	} else if !filepath.IsAbs(cfg.WorkingDir) {
		cfg.WorkingDir = filepath.Join(baseDir, cfg.WorkingDir)
	}
	if cfg.SecretPrompt == "" && cfg.SecretEnv != "" {
		cfg.SecretPrompt = "Secret value"
	}

	cfg.Lanes = normalizeStringList(cfg.Lanes)
	if len(cfg.Lanes) == 0 {
		cfg.Lanes = []string{"preview", "canary", "production"}
	}

	if cfg.DefaultLane == "" || !containsString(cfg.Lanes, cfg.DefaultLane) {
		cfg.DefaultLane = cfg.Lanes[0]
	}

	if err := sanitizeSteps(cfg.Steps, cfg.WorkingDir); err != nil {
		return err
	}
	if err := sanitizeSteps(cfg.BeforeAll, cfg.WorkingDir); err != nil {
		return err
	}
	if err := sanitizeSteps(cfg.AfterAll, cfg.WorkingDir); err != nil {
		return err
	}
	if err := sanitizeSteps(cfg.OnFailure, cfg.WorkingDir); err != nil {
		return err
	}

	if len(cfg.Flows) > 0 && len(cfg.Profiles) > 0 {
		return fmt.Errorf("wrapper config cannot define both flows and profiles")
	}
	if cfg.DefaultFlow != "" && cfg.DefaultProfile != "" &&
		cfg.DefaultFlow != cfg.DefaultProfile {
		return fmt.Errorf("default_flow and default_profile must match when both are set")
	}
	if cfg.DefaultFlow == "" && cfg.DefaultProfile != "" {
		cfg.DefaultFlow = cfg.DefaultProfile
	}

	profiles := cfg.namedPlans()
	if len(profiles) == 0 {
		if len(cfg.Steps) == 0 {
			return fmt.Errorf("wrapper config must define at least one step or one named flow")
		}
		return nil
	}

	var target *[]wrapperProfile
	if len(cfg.Flows) > 0 {
		target = &cfg.Flows
	} else {
		target = &cfg.Profiles
	}

	for i := range *target {
		if err := cfg.sanitizeProfile(&(*target)[i], baseDir); err != nil {
			return err
		}
	}

	if cfg.DefaultFlow == "" {
		cfg.DefaultFlow = (*target)[0].Name
	}
	if !containsProfileName(*target, cfg.DefaultFlow) {
		return fmt.Errorf("default flow %q is not defined", cfg.DefaultFlow)
	}
	cfg.DefaultProfile = cfg.DefaultFlow

	return nil
}

func (cfg *wrapperConfig) sanitizeProfile(
	profile *wrapperProfile,
	baseDir string,
) error {
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Description = strings.TrimSpace(profile.Description)
	profile.SecretPrompt = strings.TrimSpace(profile.SecretPrompt)
	profile.SecretEnv = strings.TrimSpace(profile.SecretEnv)

	if profile.Name == "" {
		return fmt.Errorf("named flow is missing name")
	}

	if profile.WorkingDir == "" {
		profile.WorkingDir = cfg.WorkingDir
	} else if !filepath.IsAbs(profile.WorkingDir) {
		profile.WorkingDir = filepath.Join(baseDir, profile.WorkingDir)
	}

	profile.Lanes = normalizeStringList(profile.Lanes)
	if len(profile.Lanes) == 0 {
		profile.Lanes = append([]string(nil), cfg.Lanes...)
	}

	if profile.DefaultLane == "" || !containsString(profile.Lanes, profile.DefaultLane) {
		if containsString(profile.Lanes, cfg.DefaultLane) {
			profile.DefaultLane = cfg.DefaultLane
		} else {
			profile.DefaultLane = profile.Lanes[0]
		}
	}

	if profile.SecretEnv == "" {
		profile.SecretEnv = cfg.SecretEnv
	}
	if profile.SecretPrompt == "" {
		switch {
		case cfg.SecretPrompt != "" && profile.SecretEnv == cfg.SecretEnv:
			profile.SecretPrompt = cfg.SecretPrompt
		case profile.SecretEnv != "":
			profile.SecretPrompt = "Secret value"
		}
	}

	if len(profile.Steps) == 0 {
		profile.Steps = cloneSteps(cfg.Steps)
	} else {
		profile.Steps = cloneSteps(profile.Steps)
	}
	if len(profile.Steps) == 0 {
		return fmt.Errorf("flow %q must define at least one step", profile.Name)
	}

	if err := sanitizeSteps(profile.Steps, profile.WorkingDir); err != nil {
		return err
	}
	if err := sanitizeSteps(profile.BeforeAll, profile.WorkingDir); err != nil {
		return err
	}
	if err := sanitizeSteps(profile.AfterAll, profile.WorkingDir); err != nil {
		return err
	}
	return sanitizeSteps(profile.OnFailure, profile.WorkingDir)
}

func sanitizeSteps(steps []wrapperStep, workingDir string) error {
	for i := range steps {
		step := &steps[i]
		step.ID = strings.TrimSpace(step.ID)
		step.Title = strings.TrimSpace(step.Title)
		step.Detail = strings.TrimSpace(step.Detail)
		step.Dir = strings.TrimSpace(step.Dir)

		if step.ID == "" {
			step.ID = fmt.Sprintf("step-%d", i+1)
		}
		if step.Title == "" {
			step.Title = step.ID
		}
		if step.Detail == "" {
			step.Detail = strings.Join(step.Command, " ")
		}
		if len(step.Command) == 0 {
			return fmt.Errorf("step %q is missing command", step.ID)
		}
		if step.Dir == "" {
			step.Dir = workingDir
		} else if !filepath.IsAbs(step.Dir) {
			step.Dir = filepath.Join(workingDir, step.Dir)
		}
		step.Lanes = normalizeStringList(step.Lanes)
		for j := range step.Artifacts {
			step.Artifacts[j].Name = strings.TrimSpace(step.Artifacts[j].Name)
			step.Artifacts[j].Path = strings.TrimSpace(step.Artifacts[j].Path)
			step.Artifacts[j].Description = strings.TrimSpace(step.Artifacts[j].Description)
			if step.Artifacts[j].Path == "" {
				return fmt.Errorf("step %q defines an artifact with empty path", step.ID)
			}
		}
		if step.Env == nil {
			step.Env = map[string]string{}
		}
	}

	return nil
}

func (cfg wrapperConfig) namedPlans() []wrapperProfile {
	if len(cfg.Flows) > 0 {
		return cfg.Flows
	}
	return cfg.Profiles
}

func (cfg wrapperConfig) namedPlanNames() []string {
	plans := cfg.namedPlans()
	names := make([]string, 0, len(plans))
	for _, plan := range plans {
		names = append(names, plan.Name)
	}
	return names
}

func (cfg wrapperConfig) defaultPlanName() string {
	if cfg.DefaultFlow != "" {
		return cfg.DefaultFlow
	}
	if cfg.DefaultProfile != "" {
		return cfg.DefaultProfile
	}

	plans := cfg.namedPlans()
	if len(plans) == 0 {
		return ""
	}
	return plans[0].Name
}

func (cfg wrapperConfig) planFor(name string) wrapperPlan {
	plans := cfg.namedPlans()
	if len(plans) == 0 {
		return wrapperPlan{
			Name:            "",
			Description:     cfg.Description,
			Lanes:           append([]string(nil), cfg.Lanes...),
			DefaultLane:     cfg.DefaultLane,
			RequireApproval: cfg.RequireApproval,
			SecretPrompt:    cfg.SecretPrompt,
			SecretEnv:       cfg.SecretEnv,
			WorkingDir:      cfg.WorkingDir,
			BeforeAll:       cloneSteps(cfg.BeforeAll),
			AfterAll:        cloneSteps(cfg.AfterAll),
			OnFailure:       cloneSteps(cfg.OnFailure),
			Steps:           cloneSteps(cfg.Steps),
		}
	}

	selected := name
	if selected == "" {
		selected = cfg.defaultPlanName()
	}

	for _, plan := range plans {
		if plan.Name != selected {
			continue
		}

		requireApproval := cfg.RequireApproval
		if plan.RequireApproval != nil {
			requireApproval = *plan.RequireApproval
		}
		description := plan.Description
		if description == "" {
			description = cfg.Description
		}

		return wrapperPlan{
			Name:            plan.Name,
			Description:     description,
			Lanes:           append([]string(nil), plan.Lanes...),
			DefaultLane:     plan.DefaultLane,
			RequireApproval: requireApproval,
			SecretPrompt:    plan.SecretPrompt,
			SecretEnv:       plan.SecretEnv,
			WorkingDir:      plan.WorkingDir,
			BeforeAll:       append(cloneSteps(cfg.BeforeAll), cloneSteps(plan.BeforeAll)...),
			AfterAll:        append(cloneSteps(cfg.AfterAll), cloneSteps(plan.AfterAll)...),
			OnFailure:       append(cloneSteps(cfg.OnFailure), cloneSteps(plan.OnFailure)...),
			Steps:           cloneSteps(plan.Steps),
		}
	}

	return cfg.planFor(cfg.defaultPlanName())
}

func cloneSteps(steps []wrapperStep) []wrapperStep {
	cloned := make([]wrapperStep, len(steps))
	for i, step := range steps {
		cloned[i] = step
		if step.Command != nil {
			cloned[i].Command = append([]string(nil), step.Command...)
		}
		if step.Lanes != nil {
			cloned[i].Lanes = append([]string(nil), step.Lanes...)
		}
		if step.Env != nil {
			cloned[i].Env = make(map[string]string, len(step.Env))
			for key, value := range step.Env {
				cloned[i].Env[key] = value
			}
		}
		if step.Artifacts != nil {
			cloned[i].Artifacts = append([]ArtifactDeclaration(nil), step.Artifacts...)
		}
	}
	return cloned
}

func containsProfileName(profiles []wrapperProfile, needle string) bool {
	for _, profile := range profiles {
		if profile.Name == needle {
			return true
		}
	}
	return false
}

func normalizeStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
