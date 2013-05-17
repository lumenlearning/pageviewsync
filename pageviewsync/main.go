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

package main

import (
	"flag"
	"fmt"
	lmnIo "lumenlearning.com/util/io"
	"lumenlearning.com/pageviewsync/worker"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var authFilePath *string = flag.String("auth", "", "A relative or absolute file path to the file containing the authorization token you want to use for API calls.")
var serverAddress *string = flag.String("srv", "", "The name of the server running the Canvas instance you want to interact with.")
var userIDFilePath *string = flag.String("users", "", "The path to the file containing a list of comma separated user IDs.")
var maxWorkers *int = flag.Int("maxworkers", 10, "The maximum number of concurrent requests against the API server.")
var maxProcs *int = flag.Int("maxprocs", runtime.NumCPU(), "The maximum number of processor cores to use.")
var verbose *bool = flag.Bool("verbose", false, "Display debugging information.")

func main() {
	// Create a channel for communicating log information
	logChan := make(chan interface{})
	go monitorLog(logChan)

	// Set the maximum number of processors to use
	runtime.GOMAXPROCS(*maxProcs)
	if *verbose {
		logChan <- fmt.Sprintf("Using %v processors.", *maxProcs)
	}

	// Get all of our command line arguments.
	flag.Parse()
	if *verbose {
		logChan <- fmt.Sprintf("Command line options:")
		logChan <- fmt.Sprintf("\tauth: %v", *authFilePath)
		logChan <- fmt.Sprintf("\tsrv: %v", *serverAddress)
		logChan <- fmt.Sprintf("\tusers: %v", *userIDFilePath)
		logChan <- fmt.Sprintf("\tmaxworkers: %v", *maxWorkers)
		logChan <- fmt.Sprintf("\tmaxprocs: %v", *maxProcs)
		fmt.Println()
	}

	// Read in the authorization token
	auth, err := lmnIo.ReadFile(*authFilePath)
	if err != nil {
		panic(err.Error())
	}
	if *verbose {
		logChan <- fmt.Sprintf("Successfully read contents of auth file:\t\n%v", auth)
	}

	// Read in the user IDs file
	usersString, err := lmnIo.ReadFile(*userIDFilePath)
	if err != nil {
		panic(err.Error())
	}
	if *verbose {
		logChan <- fmt.Sprintf("Successfully read contents of users file:\t\n%v", usersString)
	}
	users := strings.Split(usersString, ",")

	// Create objects to store DB and API connection information.
	// Also, for use with all API requests, create a new http.Client
	dbInfo := worker.DBConnectInfo{User: "username", Pass: "password", Schema: "canvas_pageviews", Table: "pageviews"}
	apiInfo := worker.APIConnectInfo{Host: "canvas.instructure.com", Auth: auth, Client: new(http.Client)}
	if *verbose {
		logChan <- fmt.Sprintf("Database Connection Info: %+v", dbInfo)
		logChan <- fmt.Sprintf("API Connection Info: %+v", apiInfo)
	}

	// Kick off a goroutine to load users into the queue
	// In general, just before we kick off a new goroutine,
	//   we always increment the WaitGroup.
	var waitGrp sync.WaitGroup
	waitGrp.Add(1)

	go func(waitGrp *sync.WaitGroup) {
		workGrp := make(chan bool, *maxWorkers)
		for _, u := range users {
			// Get rid of superfluous whitespace
			u = strings.TrimSpace(u)

			logChan <- fmt.Sprintf("Processing user: %v", u)
			// This will block if we've got too many workers going already.
			workGrp <- true

			waitGrp.Add(1)
			wrk := worker.Worker{
				UserID:  u,
				DBInfo:  dbInfo,
				APIInfo: apiInfo,

				WorkGrp: workGrp,
				WaitGrp: waitGrp,
				Logger:  logChan,
			}

			go wrk.RunPageviewUpdate()
		}

		close(workGrp)

		// Indicate that this goroutine has finished
		waitGrp.Done()
	}(&waitGrp)

	// Here is where we wait for all of the goroutines to finish
	waitGrp.Wait()

	// Flush stdout so we see all the error messages
	for len(logChan) > 1 {
		time.Sleep(time.Second)
	}
	os.Stdout.Sync()
	time.Sleep(5 * time.Second)
}

func monitorLog(logChan chan interface{}) {
	for {
		fmt.Println(<-logChan)
	}
}
