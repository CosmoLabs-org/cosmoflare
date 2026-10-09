package cli

import (
	"os"
	"strings"
	"testing"
)

func TestIsAgentEnvironment(t *testing.T) {
 saved := map[string]string{}
 for _, key := range agentEnvVars {
  saved[key] = os.Getenv(key)
  os.Unsetenv(key)
 }
 t.Cleanup(func() {
  for k, v := range saved {
   if v == "" {
    os.Unsetenv(k)
   } else {
    os.Setenv(k, v)
   }
  }
 })

 if IsAgentEnvironment() {
  t.Error("no markers set: should not detect an agent")
 }
 t.Setenv("CLAUDECODE", "1")
 if !IsAgentEnvironment() {
  t.Error("CLAUDECODE set: should detect an agent")
 }
 os.Unsetenv("CLAUDECODE")
 t.Setenv("AGENT", "yes")
 if !IsAgentEnvironment() {
  t.Error("generic AGENT marker: should detect an agent")
 }
}

func TestAgentHelpPreambleContract(t *testing.T) {
 for _, want := range []string{"--json", "cosmoflare search", "error_code"} {
  if !strings.Contains(AgentHelpPreamble, want) {
   t.Errorf("preamble missing %q", want)
  }
 }
}
