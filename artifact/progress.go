package artifact

import "time"

const progressInterval = 200 * time.Millisecond

type progressReporter struct {
	callback func(Progress)
	total    int64
	last     time.Time
}

func newProgressReporter(callback func(Progress), total int64) *progressReporter {
	return &progressReporter{callback: callback, total: total}
}

func (r *progressReporter) forAsset(asset string, completed int64) func(int64) {
	return func(written int64) {
		if r.callback == nil {
			return
		}
		now := time.Now()
		current := completed + written
		if current < r.total && now.Sub(r.last) < progressInterval {
			return
		}
		r.last = now
		r.callback(Progress{Asset: asset, DownloadedBytes: current, TotalBytes: r.total})
	}
}

func (r *progressReporter) complete() {
	if r.callback != nil {
		r.callback(Progress{DownloadedBytes: r.total, TotalBytes: r.total})
	}
}
