package mock

import "io"

// noopRSC implements io.ReadSeekCloser with no-op Close.
type noopRSC struct{ io.ReadSeeker }

func (noopRSC) Close() error { return nil }
