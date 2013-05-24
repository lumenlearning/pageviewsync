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

package worker

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	lmnHttp "lumenlearning.com/util/http"
	lmnCanvas "lumenlearning.com/util/canvas/api"
	lmnTime "lumenlearning.com/util/time"
	"net/http"
	"sync"
)

type Worker struct {
	UserID string
	DBInfo DBConnectInfo
	APIInfo APIConnectInfo
	
	WorkGrp chan bool
	WaitGrp *sync.WaitGroup
	Logger chan interface{}
}

type DBConnectInfo struct {
	User string
	Pass string
	Schema string
	Table string
}

func (d *DBConnectInfo) String() string {
	return "User:"+d.User+",Pass:"+d.Pass+",Schema:"+d.Schema+",Table:"+d.Table
}

type APIConnectInfo struct {
	Host string
	Auth string
	Client *http.Client
}

func (a *APIConnectInfo) String() string {
	return "Host:"+a.Host+",Auth:"+a.Auth+",Client:"+fmt.Sprint(a.Client)
}

// Indicate that this goroutine is finished.
// Free up space for another worker to start.
func (w *Worker) Done() {
	w.WaitGrp.Done()
	<- w.WorkGrp
}

func (w *Worker) Log(format string, args... interface{}) {
	w.Logger <- fmt.Sprintf("%v: %v", w.UserID, fmt.Sprintf(format, args...))
}

func (w *Worker) RunPageviewUpdate() error {
	// Find out what we need to do to get this user up-to-date.
	// We'll find out what the last timestamp we have recorded is,
	// if any.
	lastRequestID, lastTimestamp, err := GetUpdateReqs(w.UserID, w.DBInfo)
	if err != nil {
		w.Done()
		w.Log("Terminating. worker.GetUpdateReqs => %v", err.Error())
		return errors.New("worker.GetUpdateReqs => " + err.Error())
	}
	var fullUpdate bool = (lastTimestamp == "" || lastRequestID == "")

	w.Log("lastRequestID=%v, lastRequestID=%v, fullUpdate=%v", lastRequestID, lastTimestamp, fullUpdate)
	
	// Get the information needed to update the local DB
	url := "https://"+w.APIInfo.Host+"/api/v1/users/"+w.UserID+"/page_views?per_page=1000"

	// Go get as many pages as we need
	var pvs []lmnCanvas.Pageview
	keepGoing := true
	for keepGoing {
		w.Log("Fetching %v", url)
		nextLink, results, err := ParsePage(url, w.APIInfo)
		if err != nil {
			w.Done()
			w.Log("Terminating. worker.ParsePage => %v", err.Error())
			return errors.New("worker.ParsePage => " + err.Error())
		}

		// Update url to be the next link
		url = nextLink
		
		if !fullUpdate {
			// Find out if we need to keep going
			for _, r := range results {
				ts, _ := r["created_at"]
				id, _ := r["request_id"]

				tISO, err := lmnTime.TimeFromISO8601Full(ts.(string))
				if err != nil {
					w.Done()
					w.Log("Terminated. lmnTime.TimeFromISO8601 => %v", err)
					return errors.New("lmnTime.TimeFromISO8601 => " + err.Error())
				}
				tUnix, err := lmnTime.ISO8601BasicFromTime(tISO)
				if err != nil {
					w.Done()
					w.Log("Terminated. lmnTime.ISO8601BasicFromTime => %v", err)
					return errors.New("lmnTime.ISO8601BasicFromTime => " + err.Error())
				}
				
				if id.(string) == lastRequestID || tUnix < lastTimestamp {
					keepGoing = false
					break
				} else {
					pvs = append(pvs, r)
				}
			}
		} else {
			// Add everything from this page
			pvs = append(pvs, results...)
		}

		// Is there anything left to grab?
		if nextLink == "" {
			keepGoing = false
			break
		}
	}
	
	insertCount, err := UpdateDB(&w.DBInfo, &pvs)
	if err != nil {
		w.Done()
		w.Log("Terminated. worker.UpdateDB => %v", err.Error())
		return errors.New("worker.UpdateDB => " + err.Error())
	}

	// If we got to this point, no errors!
	w.Done()
	w.Log("Updated with %v rows.", insertCount)
	return nil
}

func UpdateDB(dbInfo *DBConnectInfo, pvs *[]lmnCanvas.Pageview) (int64, error) {
	//// send the results to the local database ////
	// make the connection
	con, err := sql.Open("mysql", dbInfo.User+":"+dbInfo.Pass+"@unix(/var/run/mysqld/mysqld.sock)"+"/"+dbInfo.Schema)
	defer con.Close()
	if err != nil {
		return 0, errors.New("sql.Open => " + err.Error())
	}

	// turn off autocommit to keep our updates atomic
	// This is unnecessary.
/*	_, err = con.Exec("SET autocommit=0;")
	if err != nil {
		return 0, errors.New("con.Exec => " + err.Error())
 }*/ 
	
	// start a transaction
	tx, err := con.Begin()
	if err != nil {
		return 0, errors.New("con.Begin => " + err.Error())
	}
	
	// Keep track of total pageview records updated
	var insertCount int64
	
	// convert each lmnCanvas.Pageview to an insert statement
	for _, pv := range *pvs {
		ins := "INSERT INTO "+dbInfo.Table+" ("
		val := " VALUES ("
		i := 0
		for k, v := range pv {
			// Get the string representation of the value
			strVal := fmt.Sprint(v)
			if strVal == "<nil>" {
				strVal = "NULL"
			}

			// build insert and value strings for this item
			insI, valI := BuildInsertAndValues(k, strVal, i)
			ins = ins + insI
			val = val + valI
			i += 1

			// If it's the "created_at" field, we also need to make a copy
			//   in YYYY-mm-dd HH:MM:SS format.  It's NOT "Unix" time format,
			//   but that's what I called it for lack of a better term.
			// Also for the "updated_at" field.
			if k == "created_at" {
				insI, valI, err := GetDateTimeValue("created_at_datetime", strVal, i)
				if err != nil {
					return 0, errors.New("worker.GetDateTimeValue => " + err.Error())
				}
				ins = ins + insI
				val = val + valI
				i += 1
			}
			if k == "updated_at" {
				insI, valI, err := GetDateTimeValue("updated_at_datetime", strVal, i)
				if err != nil {
					return 0, errors.New("worker.GetDateTimeValue => " + err.Error())
				}
				ins = ins + insI
				val = val + valI
				i += 1
			}

		}
		
		// Finish up the query bits
		ins = ins + ")"
		val = val + ")"
		
		// add each insert to the transaction
		qu := ins + " " + val
		res, err := tx.Exec(qu)
		if err != nil {
			err1 := err
			errorString := "tx.Exec => "+err1.Error()

			err := tx.Rollback()
			if err != nil {
				errorString += "\n"+err.Error()
			}
			
			return 0, errors.New(errorString)
		}

		// Record number of rows inserted
		rowsAffected, err := res.(sql.Result).RowsAffected()
		if err != nil {
			return 0, errors.New("res.RowsAffected => " + err.Error())
		}
		insertCount += rowsAffected
	}
	
	// commit the transaction
	err = tx.Commit()
	if err != nil {
		return 0, errors.New("tx.Commit => " + err.Error())
	}

	return insertCount, nil
}

func GetDateTimeValue(k, strVal string, i int) (string, string, error) {
	t, err := lmnTime.TimeFromISO8601Full(strVal)
	if err != nil {
		return "", "", errors.New("lmnCanvas.TimeFromISO8601 => " + err.Error())
	}
	
	tu, err := lmnTime.ISO8601BasicFromTime(t)
	if err != nil {
		return "", "", errors.New("lmnCanvas.UnixFromTime => " + err.Error())
	}
	
	insI, valI := BuildInsertAndValues(k, tu, i)
	return insI, valI, nil
}

func BuildInsertAndValues(k, strVal string, i int) (string, string) {
	ins := ""
	val := ""
	
	// Change the quote character to handle NULLs, which shouldn't be in quotes.
	qc := "\""
	if strVal == "NULL" {
		qc = ""
	}

	if i > 0 {
		ins = fmt.Sprintf(",`%v`", k)
		val  = fmt.Sprintf(",%v%v%v", qc, strVal, qc)
	} else {
		ins = fmt.Sprintf("`%v`", k)
		val = fmt.Sprintf("%v%v%v", qc, strVal, qc)
	}

	return ins, val
}


// Call the API and parse the results
// Return  the "next" link
// Return a []lmnCanvas.Pageview
func ParsePage(url string, apiInfo APIConnectInfo) (string, []lmnCanvas.Pageview, error) {
	// Call the API and get the response
	resp, body, err := GetResponse(url, apiInfo)
	if err != nil {
		return "", nil, errors.New("worker.GetResponse" + err.Error())
	}

	// Get the Link header, check for rel="next"
	nextLink, err := lmnCanvas.GetNextLink(resp)
	if err != nil {
		return "", nil, errors.New("lmnCanvas.GetNextLink => " + err.Error())
	}

	// Get a map representation of the data returned
	obj, err := lmnCanvas.GetObjFromJSON(body)
	if err != nil {
		return "", nil, errors.New("lmnCanvas.GetObjFromJSON => " + err.Error())
	}

	var results []interface{}

	switch vt := (*obj).(type) {
	case []interface{}:
		results = (*obj).([]interface{})
	default:
		return "", nil, errors.New(fmt.Sprintf("Expecting an array, received %v", vt))
	}

	var pvs []lmnCanvas.Pageview

	for _, v := range results {
		switch v.(type) {
		case map[string]interface{}:
			pageviewResult := v.(map[string]interface{})
			var pv lmnCanvas.Pageview = make(lmnCanvas.Pageview)
			for k, v := range pageviewResult {
				pv[k] = v
			}
			pvs = append(pvs, pv)
		}
	}

	return nextLink, pvs, nil
}

func GetResponse(url string, apiInfo APIConnectInfo) (*http.Response, *[]byte, error) {
	resp, err := lmnCanvas.AuthorizedCall(url, apiInfo.Auth, apiInfo.Client)
	if err != nil {
		return nil, nil, errors.New("lmnCanvas.AuthorizedCall => " + err.Error())
	}

	body, err := lmnHttp.ReadResponseBody(resp)
	if err != nil {
		return nil, nil, errors.New("lmnCanvas.ReadResponse => " + err.Error())
	}

	return resp, body, nil
}

func GetUpdateReqs(userID string, dbInfo DBConnectInfo) (string, string, error) {
	// Connect to the DB and the appropriate schema
	con, err := sql.Open("mysql", dbInfo.User+":"+dbInfo.Pass+"@unix(/var/run/mysqld/mysqld.sock)"+"/"+dbInfo.Schema)
	defer con.Close()
	if err != nil {
		return "", "", errors.New("sql.Open => " + err.Error())
	}

	// Build the query
	query := "SELECT request_id, created_at FROM pageviews WHERE user_id = '"+userID+"' ORDER BY created_at DESC LIMIT 1"

	// Find out the last pageview timestamp for this userID
	row := con.QueryRow(query)
	var request_id string
	var created_at string
	err = row.Scan(&request_id, &created_at)

	if err != nil {
		if err == sql.ErrNoRows {
			request_id = ""
			created_at = ""
		} else {
			return "", "", errors.New("row.Scan => " + err.Error())
		}
	}

	return request_id, created_at, nil
}
