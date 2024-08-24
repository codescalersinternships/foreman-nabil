package foreman

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForeman(t *testing.T) {
	// Create necessary files for testing
	tempDir := t.TempDir()
	const (
		valid_procfile = `app1:
    cmd: ping -c 1 google.com | grep google
    checks:
        cmd: sleep 3
    deps: 
        - app2
app2:
    cmd: ping -c 10 yahoo.com | grep yahoo
    run_once: true
    checks:
        cmd: sleep 4
        tcp_ports: [8080]
        udp_ports: [80]

app3:
    run_once: true
    cmd: sleep 10
    checks:
        tcp_ports: ["8090"]
        udp_ports: ["90"]
    deps:
      - app1`
		cycle_procfile = `web:
  cmd: "echo Starting web server"
  deps: ["db"]

db:
  cmd: "echo Starting database"
  deps: ["web"]`
		cycle_procfiles2 = `app1:
    cmd: ping -c 1 google.com
    run_once: true
    checks:
        cmd: sleep 3
    deps: 
        - app2
app2:
    cmd: ping -c 10 yahoo.com | grep yahoo
    run_once: true
    deps:
        - app1
`
		invalid_command_procfile = `web:
  cmd: "nonexistentcommand"
  run_once: true`
		invalid_format_procfile = `web:
  cmd: "echo Starting web server"
  deps: ["db"
db:
  cmd: "echo Starting database"`
	)

	validProcfilePath := filepath.Join(tempDir, "valid_procfile.yaml")
	err := os.WriteFile(validProcfilePath, []byte(valid_procfile), 0644)
	assert.NoError(t, err)

	cycleProcfilePath := filepath.Join(tempDir, "cycle_procfile.yaml")
	err = os.WriteFile(cycleProcfilePath, []byte(cycle_procfile), 0644)
	assert.NoError(t, err)

	invalidCommandProcfilePath := filepath.Join(tempDir, "invalid_command_procfile.yaml")
	err = os.WriteFile(invalidCommandProcfilePath, []byte(invalid_command_procfile), 0644)
	assert.NoError(t, err)

	invalidformedProcfilePath := filepath.Join(tempDir, "malformed_procfile.yaml")
	err = os.WriteFile(invalidformedProcfilePath, []byte(invalid_format_procfile), 0644)
	assert.NoError(t, err)	

	cycleProcfile2Path := filepath.Join(tempDir, "malformed_procfile.yaml")
	err = os.WriteFile(invalidformedProcfilePath, []byte(cycle_procfiles2), 0644)
	assert.NoError(t, err)	

	// Define the test cases
	tests := []struct {
		name       string
		procfile   string
		expectErr  bool
	}{
		{
			name:      "ValidFile",
			procfile:  validProcfilePath,
			expectErr: false,
		},
		{
			name:      "DependenciesFormCycle",
			procfile:  cycleProcfilePath,
			expectErr: true,
		},
		{
			name:      "DependenciesFormCycle another example",
			procfile:  cycleProcfile2Path,
			expectErr: true,
		},
		{
			name:      "InvalidCommand",
			procfile:  invalidCommandProcfilePath,
			expectErr: true,
		},
		{
			name:      "FilepathNotFound",
			procfile:  filepath.Join(tempDir, "nonexistent_procfile.yaml"),
			expectErr: true,
		},
		{
			name:      "invalid formed Yaml",
			procfile:  invalidformedProcfilePath,
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			foreman, err := InitForeman(test.procfile)
			if err != nil {
				if !test.expectErr {
					t.Errorf("not expecting error but got: %v",err)
					return
				}else {
					return
				}
			}

			err = foreman.RunServices()
			if err != nil {
				if !test.expectErr {
					t.Errorf("not expecting error but got: %v",err)
					return
				}else {
					return
				}
			}
			if test.expectErr {
				t.Errorf("expecting error but got nil")
			}
		})
	}
}
