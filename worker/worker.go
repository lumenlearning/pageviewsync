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
	"fmt"
	"time"
	"sync"
)

func UpdateUserPageviews(userID string, workerBuffer chan bool, wg *sync.WaitGroup) error {
	// go do the update stuff
	fmt.Println("Fetching user: "+userID)
	time.Sleep(time.Second * 1)
	
	// Indicate that we're done and free up space for another worker.
	<- workerBuffer
	wg.Done()

	return nil
}