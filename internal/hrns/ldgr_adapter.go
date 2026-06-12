package hrns

import (
	"fmt"
	"os"
	"os/exec"
)

var runLdgrVerify = defaultRunLdgrVerify

func defaultRunLdgrVerify() error {
	cmd := exec.Command("ldgr", "verify", "--target", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ldgr verify: %w", err)
	}
	return nil
}
