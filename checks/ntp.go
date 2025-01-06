package checks

import (
	"time"

	"github.com/beevik/ntp"
)

// Chris Marotta's NTP implementation

type Ntp struct {
	checkBase
	CheckTimeAccuracy           bool
	AcceptableOffsetMillisecond int
}

func (c Ntp) Run(teamID uint, boxIp string, res chan Result) {
	response, err := ntp.Query(boxIp)
	if err != nil {
		res <- Result{
			Error: "Error Querying NTP Sever.",
			Debug: err.Error(),
		}
		return
	}
	if c.CheckTimeAccuracy {
		if response.ClockOffset.Abs() >= time.Duration(c.AcceptableOffsetMillisecond)*time.Millisecond {
			res <- Result{
				Error: "Time server not providing time within expected range.",
				Debug: err.Error(),
			}
			return
		}
	}
	res <- Result{
		Status: true,
		Debug:  "Recieved expected time from NTP Server.",
	}
}
