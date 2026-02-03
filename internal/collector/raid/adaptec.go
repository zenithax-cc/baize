package raid

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const (
	snFile  = "/host0/scsi_host/host0/serial_number"
	arcconf = "/usr/local/hwtool/tool/arcconf"
)

type adaptecController struct {
	ctrl *controller
	cid  string
}

func collectAdaptec(ctx context.Context, i int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	arcCtr := &adaptecController{
		ctrl: c,
	}

	output := execute.Command("dmidecode", "-s", "system-manufacturer")
	if output.Err != nil {
		return output.Err
	}

	if bytes.HasPrefix(bytes.TrimSpace(output.Stdout), []byte("HP")) {
		return collectHPE(ctx, i, c)
	}

	if !arcCtr.isFound(i) {
		return fmt.Errorf("adaptec controller %s not found", c.PCIe.PCIAddr)
	}

	if err := arcCtr.collect(ctx); err != nil {
		return err
	}

	return nil
}

func (ac *adaptecController) isFound(i int) bool {
	sn, err := os.ReadFile(sysfsDevicesPath + ac.ctrl.PCIe.PCIAddr + snFile)
	if err != nil {
		return false
	}

	for j := 0; j < i+1; j++ {
		output := execute.Command(arcconf, "GETCONFIG", strconv.Itoa(i), "AD")
		if output.Err != nil {
			continue
		}
		if len(output.Stdout) > 0 && bytes.Contains(output.Stdout, bytes.TrimSpace(sn)) {
			ac.cid = strconv.Itoa(j)
			return true
		}
	}

	return false
}

func arcconfCmd(ctx context.Context, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	output := execute.CommandWithContext(ctx, arcconf+" GETCONFIG", args...)
	if output.Err != nil {
		return nil, output.Err
	}

	return output.Stdout, nil
}

func (ac *adaptecController) collect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var errs []error
	if err := ac.parseCtrlCard(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := ac.collectCtrlPD(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := ac.collectCtrlLD(ctx); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (ac *adaptecController) parseCtrlCard(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := arcconfCmd(ctx, ac.cid, "AD")
	if err != nil {
		return err
	}

}
