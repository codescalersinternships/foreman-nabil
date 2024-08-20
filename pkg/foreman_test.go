package foreman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestForeman(t *testing.T) {
	// Create necessary files for testing
	tempDir := t.TempDir()
	const (
		valid_procfile = `web:
  cmd: "echo Starting web server"
  deps: []`
		cycle_procfile = `web:
  cmd: "echo Starting web server"
  deps: ["db"]

db:
  cmd: "echo Starting database"
  deps: ["web"]`
		invalid_command_procfile = `web:
  cmd: "nonexistentcommand"
  run_once: true`
		invalid_format_procfile = `web:
  cmd: "echo Starting web server"
  deps: ["db"
db:
  cmd: "echo Starting database"`
// 		port_conflict_procfile = `web1:
//   cmd: "nc -l 8080"
//   run_once: true
//   checks:
//     tcp_ports: ["8080"]

// web2:
//   cmd: "nc -l 8080"
//     run_once: true
//   checks:
//     tcp_ports: ["8080"]`
	)

	validProcfilePath := filepath.Join(tempDir, "valid_procfile.yaml")
	os.WriteFile(validProcfilePath, []byte(valid_procfile), 0644)

	cycleProcfilePath := filepath.Join(tempDir, "cycle_procfile.yaml")
	os.WriteFile(cycleProcfilePath, []byte(cycle_procfile), 0644)

	invalidCommandProcfilePath := filepath.Join(tempDir, "invalid_command_procfile.yaml")
	os.WriteFile(invalidCommandProcfilePath, []byte(invalid_command_procfile), 0644)

	invalidformedProcfilePath := filepath.Join(tempDir, "malformed_procfile.yaml")
	os.WriteFile(invalidformedProcfilePath, []byte(invalid_format_procfile), 0644)

	// portConflictProcfilePath := filepath.Join(tempDir, "port_conflict_procfile.yaml")
	// os.WriteFile(portConflictProcfilePath, []byte(port_conflict_procfile), 0644)

	

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
		// {
		// 	name:      "PortConflict",
		// 	procfile:  portConflictProcfilePath,
		// 	expectErr: true,
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			foreman, err := InitForeman(test.procfile)
			cnt := 0;
			if err != nil {
				cnt++;
				if !test.expectErr {
					t.Errorf("not expecting error but got: %v",err)
					return
				}else {
					return
				}
			}

			err = foreman.RunServices()
			if err != nil {
				cnt++
			}
			if test.expectErr {
				if cnt <= 0{
					t.Errorf("expected error but got nil")
				}
			}
		})
	}
}
