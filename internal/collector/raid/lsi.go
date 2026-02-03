package raid

import "context"

func collectLSI(ctx context.Context, i int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}
