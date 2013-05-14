/******************************************************************************
pageviewsync Source Code
Copyright (C) 2013 Lumen LLC.

This file is part of the pageviewsync Source Code.

pageviewsync is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

pageviewsync is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with pageviewsync.  If not, see <http://www.gnu.org/licenses/>.
*****************************************************************************/

package canvas

import (
	"errors"
	"io/ioutil"
	"time"
)

var ISO8601TimeFmt string = "2006-01-02T15:04:05-07:00"
var UnixTimeFmt string = "2006-01-02 15:04:05"

func ReadFile(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	
	str := string(b)
	return str, nil
}

func TimeFromISO8601 (dateTime string) (time.Time, error) {
	t, err := time.Parse(ISO8601TimeFmt, dateTime)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func TimeFromUnix (dateTime string) (time.Time, error) {
	t, err := time.Parse(UnixTimeFmt, dateTime)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func UnixFromTime (t time.Time) (string, error) {
	u := t.Format(UnixTimeFmt)
	if u == "" {
		return "", errors.New("Unable to format as:"+UnixTimeFmt)
	}
	return u, nil
}

func ISO8601FromTime (t time.Time) (string, error) {
	i := t.Format(ISO8601TimeFmt)
	if i == "" {
		return "", errors.New("Unable to format as:"+UnixTimeFmt)
	}
	return i, nil
}
