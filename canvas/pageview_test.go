package canvas

import (
	"math"
	"testing"
	"time"
)

func TestConvDateTime(t *testing.T) {
	tolerance, err := time.ParseDuration("1s")
	if err != nil {
		t.Error(err)
	}

	now := time.Now()
	nowFrm := now.Format(TimeFmt)
	nowParse, err := time.Parse(TimeFmt, nowFrm)
	if err != nil {
		t.Error(err)
	}

	diff := math.Abs(nowParse.Sub(now).Seconds())
	if diff > tolerance.Seconds() {
		t.Error(now, " != ", nowParse)
	}
}