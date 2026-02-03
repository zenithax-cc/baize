package raid

import "context"

func collectIntel(ctx context.Context, i int, c *controller) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}
