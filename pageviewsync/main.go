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
	"fmt"
	"flag"
	"lumenlearning.com/pageviews/canvas"
	"lumenlearning.com/pageviews/worker"
	"strings"
	"sync"
)


var authFilePath *string = flag.String("auth", "", "A relative or absolute file path to the file containing the authorization token you want to use for API calls.")
var serverAddress *string = flag.String("srv", "", "The name of the server running the Canvas instance you want to interact with.")
//var apiEndpoint *string = flag.String("api", "", "The API endpoint you want to call.")
var userIDFilePath *string = flag.String("users", "", "The path to the file containing a list of comma separated user IDs.")
var maxWorkers *int = flag.Int("maxworkers", 10, "The maximum number of concurrent requests against the API server.")


func main (){
	// Get all of our command line arguments.
	flag.Parse()

	// Read in the authorization token
	auth, err := canvas.ReadFile(*authFilePath)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(auth)

	// Read in the user IDs file
	usersString, err := canvas.ReadFile(*userIDFilePath)
	if err != nil {
		panic(err.Error())
	}
	users := strings.Split(*usersString, ",")

	// send a goroutine off to load users into the queue
	rtnStart := make(chan bool, *maxWorkers - 1)
	rtnFinish := make(chan bool, *maxWorkers - 1)
	
	// just before we kick off a new goroutine, queue a bool in liveGrp
	rtnStart <- true
	go func(rtnFinish chan bool) {
		workGrp := make(chan bool, *maxWorkers)
		for _, u := range users {
			// This will block if we've got too many workers going already.
			workGrp <- true

			rtnStart <- true
			go worker.UpdateUserPageviews(u, workGrp, rtnFinish)
		}

		close(workGrp)

		// when we're done with this routine, read a bool from liveGrp
		rtnFinish <- true
	}(rtnFinish)

	// Here is where we wait for all of the goroutines to have finished.
	// When finishCount == startCount, we are done.
	// I'm really hoping that there isn't a race condition lurking here.
	var startCount, finishCount uint64 = 0, 0
	for {
		select {
		case <- rtnStart:
			startCount += 1
		case <- rtnFinish:
			finishCount += 1
		}

		if finishCount == startCount {
			break
		}
	}

	fmt.Println("Program complete.")
}