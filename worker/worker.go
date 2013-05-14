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
	"lumenlearning.com/pageviews/canvas"
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
		w.Log("Terminating. GetUpdateReqs says: %v", err.Error())
		return err
	}
	var fullUpdate bool = (lastTimestamp == "" || lastRequestID == "")

	w.Log("lastRequestID=%v, lastRequestID=%v, fullUpdate=%v", lastRequestID, lastTimestamp, fullUpdate)
	
	// Get the information needed to update the local DB
	url := "https://"+w.APIInfo.Host+"/api/v1/users/"+w.UserID+"/page_views?per_page=1000"
	w.Log("First URL is %v", url)

	// Go get as many pages as we need
	var pvs []canvas.Pageview
	keepGoing := true
	for keepGoing {
		w.Log("Fetching %v", url)
		nextLink, results, err := ParsePage(url, w.APIInfo)
		if err != nil {
			w.Done()
			w.Log("Terminating. ParsePage says: %v", err.Error())
			return err
		}

		// Update url to be the next link
		w.Log("Next page: %v", nextLink)
		url = nextLink
		
		if !fullUpdate {
			// Find out if we need to keep going
			for _, r := range results {
				ts, _ := r["created_at"]
				id, _ := r["request_id"]
				w.Log("Pageview at %v (%v)", ts, id)

				tISO, err := canvas.TimeFromISO8601(ts.(string))
				if err != nil {
					w.Done()
					w.Log("Terminated. TimeFromISO8601 says: %v", err)
					return err
				}
				tUnix, err := canvas.UnixFromTime(tISO)
				if err != nil {
					w.Done()
					w.Log("Termianted. UnixFromTime says: %v", err)
					return err
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
	
	//// send the results to the local database ////
	// make the connection
	con, err := sql.Open("mysql", w.DBInfo.User+":"+w.DBInfo.Pass+"@unix(/var/run/mysqld/mysqld.sock)"+"/"+w.DBInfo.Schema)
	defer con.Close()
	if err != nil {
		w.Done()
		w.Log("Terminating. sql.Open says: %v", err.Error())
		return err
	}
	
	// start a transaction
	tx, err := con.Begin()
	if err != nil {
		w.Done()
		w.Log("Terminating. con.Begin says: %v", err.Error())
		return err
	}
	
	// Keep track of total pageview records updated
	var insertCount int64
	
	// convert each canvas.Pageview to an insert statement
	for _, pv := range pvs {
		ins := "INSERT INTO "+w.DBInfo.Table+" ("
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
					w.Done()
					w.Log("Terminating. GetDateTimeValue says: %v", err.Error())
					return err
				}
				ins = ins + insI
				val = val + valI
				i += 1
			}
			if k == "updated_at" {
				insI, valI, err := GetDateTimeValue("updated_at_datetime", strVal, i)
				if err != nil {
					w.Done()
					w.Log("Terminating. GetDateTimeValue says: %v", err.Error())
					return err
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
		w.Log("Query: %v", qu)
		res, err := tx.Exec(qu)
		if err != nil {
			err1 := err
			err := tx.Rollback()
			w.Done()
			w.Log("Terminating. tx.Exec says: %v", err.Error())
			return errors.New(err1.Error()+"\n"+err.Error())
		}

		// Record number of rows inserted
		rowsAffected, err := res.(sql.Result).RowsAffected()
		if err != nil {
			w.Done()
			w.Log("Terminating. res.RowsAffected says: %v", err.Error())
			return err
		}
		insertCount += rowsAffected
	}

	// commit the transaction
	err = tx.Commit()
	if err != nil {
		w.Done()
		w.Log("Terminating. tx.Commit says: %v", err.Error())
		return err
	}

	// If we got to this point, no errors!
	w.Done()
	w.Log("Updated with %v rows.", insertCount)
	return nil
}

func GetDateTimeValue(k, strVal string, i int) (string, string, error) {
	t, err := canvas.TimeFromISO8601(strVal)
	if err != nil {
		return "", "", err
	}
	
	tu, err := canvas.UnixFromTime(t)
	if err != nil {
		return "", "", err
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
// Return a []canvas.Pageview
func ParsePage(url string, apiInfo APIConnectInfo) (string, []canvas.Pageview, error) {
	// Call the API and get the response
	resp, body, err := GetResponse(url, apiInfo)
	if err != nil {
		return "", nil, err
	}

	// Get the Link header, check for rel="next"
	nextLink, err := canvas.GetNextLink(resp)
	if err != nil {
		return "", nil, err
	}

	// Get a map representation of the data returned
	obj, err := canvas.GetObjFromJSON(body)
	if err != nil {
		return "", nil, err
	}

	var results []interface{}

	switch vt := (*obj).(type) {
	case []interface{}:
		results = (*obj).([]interface{})
	default:
		return "", nil, errors.New(fmt.Sprint("Expecting an array (received ", vt, ")"))
	}

	var pvs []canvas.Pageview

	for _, v := range results {
		switch v.(type) {
		case map[string]interface{}:
			pageviewResult := v.(map[string]interface{})
			var pv canvas.Pageview = make(canvas.Pageview)
			for k, v := range pageviewResult {
				if k == "created_at" {
					
				}
				pv[k] = v
			}
			pvs = append(pvs, pv)
		}
	}

	return nextLink, pvs, nil
}

func GetResponse(url string, apiInfo APIConnectInfo) (*http.Response, *[]byte, error) {
	resp, _, err := canvas.AuthorizedCall(url, apiInfo.Auth, apiInfo.Client)
	if err != nil {
		return nil, nil, err
	}

	body, _, err := canvas.ReadResponse(resp)
	if err != nil {
		return nil, nil, err
	}

	return resp, body, nil
}

func GetUpdateReqs(userID string, dbInfo DBConnectInfo) (string, string, error) {
	// Connect to the DB and the appropriate schema
	con, err := sql.Open("mysql", dbInfo.User+":"+dbInfo.Pass+"@unix(/var/run/mysqld/mysqld.sock)"+"/"+dbInfo.Schema)
	defer con.Close()
	if err != nil {
		return "", "", err
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
			return "", "", err
		}
	}

	return request_id, created_at, nil
}
