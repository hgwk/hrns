package hrns

import (
	"errors"
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
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("ldgr verify: ldgr not found; install @hgwk/ldgr or run without --with-ldgr")
		}
		return fmt.Errorf("ldgr verify: %w; initialize this target with `ldgr init` or run without --with-ldgr for hrns-only audits", err)
	}
	return nil
}
