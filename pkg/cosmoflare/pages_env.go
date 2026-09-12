package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// PagesEnvVar represents a single environment variable for a Pages deployment
// environment. Type is either "plain" or "secret". Secret values are
// write-only: the Cloudflare API never returns their value, so Value is
// empty on entries returned by ListEnvVars when Type is "secret".
type PagesEnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
	Type  string `json:"type"`
}

const (
	pagesEnvVarTypePlain  = "plain"
	pagesEnvVarTypeSecret = "secret"
)

func validatePagesEnv(op, env string) error {
	if env != "production" && env != "preview" {
		return validationError(op, fmt.Sprintf("env must be %q or %q, got %q", "production", "preview", env))
	}
	return nil
}

// ListEnvVars returns the environment variables configured for a Pages
// project's production or preview deployment environment.
func (s *PagesService) ListEnvVars(ctx context.Context, project, env string) ([]PagesEnvVar, error) {
	const op = "PagesService.ListEnvVars"
	if project == "" {
		return nil, validationError(op, "project name is required")
	}
	if err := validatePagesEnv(op, env); err != nil {
		return nil, err
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetPagesProject(ctx, rc, project)
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to get project %q", project), err)
	}

	envConfig := result.DeploymentConfigs.Production
	if env == "preview" {
		envConfig = result.DeploymentConfigs.Preview
	}

	vars := make([]PagesEnvVar, 0, len(envConfig.EnvVars))
	for key, v := range envConfig.EnvVars {
		if v == nil {
			continue
		}
		vars = append(vars, mapPagesEnvVar(key, v))
	}
	return vars, nil
}

// SetEnvVars creates or updates one or more environment variables for a
// Pages project's production or preview deployment environment. Existing
// variables not included in vars are left unchanged.
func (s *PagesService) SetEnvVars(ctx context.Context, project, env string, vars []PagesEnvVar) error {
	const op = "PagesService.SetEnvVars"
	if project == "" {
		return validationError(op, "project name is required")
	}
	if err := validatePagesEnv(op, env); err != nil {
		return err
	}
	if len(vars) == 0 {
		return validationError(op, "at least one env var is required")
	}
	for _, v := range vars {
		if v.Key == "" {
			return validationError(op, "env var key is required")
		}
		if v.Type != pagesEnvVarTypePlain && v.Type != pagesEnvVarTypeSecret {
			return validationError(op, fmt.Sprintf("env var type must be %q or %q, got %q", pagesEnvVarTypePlain, pagesEnvVarTypeSecret, v.Type))
		}
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	existing, err := s.cf.GetPagesProject(ctx, rc, project)
	if err != nil {
		return newError(op, fmt.Sprintf("failed to get project %q", project), err)
	}

	configs := existing.DeploymentConfigs
	envVars := configs.Production.EnvVars
	if env == "preview" {
		envVars = configs.Preview.EnvVars
	}
	if envVars == nil {
		envVars = cloudflare.EnvironmentVariableMap{}
	}
	for _, v := range vars {
		envVars[v.Key] = &cloudflare.EnvironmentVariable{
			Value: v.Value,
			Type:  toCloudflareEnvVarType(v.Type),
		}
	}

	if env == "preview" {
		configs.Preview.EnvVars = envVars
	} else {
		configs.Production.EnvVars = envVars
	}

	_, err = s.cf.UpdatePagesProject(ctx, rc, cloudflare.UpdatePagesProjectParams{
		ID:                project,
		DeploymentConfigs: configs,
	})
	if err != nil {
		return newError(op, fmt.Sprintf("failed to set env vars for project %q", project), err)
	}
	return nil
}

// DeleteEnvVar removes a single environment variable from a Pages project's
// production or preview deployment environment.
func (s *PagesService) DeleteEnvVar(ctx context.Context, project, env, key string) error {
	const op = "PagesService.DeleteEnvVar"
	if project == "" {
		return validationError(op, "project name is required")
	}
	if err := validatePagesEnv(op, env); err != nil {
		return err
	}
	if key == "" {
		return validationError(op, "env var key is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	existing, err := s.cf.GetPagesProject(ctx, rc, project)
	if err != nil {
		return newError(op, fmt.Sprintf("failed to get project %q", project), err)
	}

	configs := existing.DeploymentConfigs
	envVars := configs.Production.EnvVars
	if env == "preview" {
		envVars = configs.Preview.EnvVars
	}
	if envVars != nil {
		delete(envVars, key)
	}

	if env == "preview" {
		configs.Preview.EnvVars = envVars
	} else {
		configs.Production.EnvVars = envVars
	}

	_, err = s.cf.UpdatePagesProject(ctx, rc, cloudflare.UpdatePagesProjectParams{
		ID:                project,
		DeploymentConfigs: configs,
	})
	if err != nil {
		return newError(op, fmt.Sprintf("failed to delete env var %q for project %q", key, project), err)
	}
	return nil
}

func toCloudflareEnvVarType(t string) cloudflare.EnvVarType {
	if t == pagesEnvVarTypeSecret {
		return cloudflare.SecretText
	}
	return cloudflare.PlainText
}

func mapPagesEnvVar(key string, v *cloudflare.EnvironmentVariable) PagesEnvVar {
	varType := pagesEnvVarTypePlain
	if v.Type == cloudflare.SecretText {
		varType = pagesEnvVarTypeSecret
	}
	value := v.Value
	if varType == pagesEnvVarTypeSecret {
		value = ""
	}
	return PagesEnvVar{
		Key:   key,
		Value: value,
		Type:  varType,
	}
}
