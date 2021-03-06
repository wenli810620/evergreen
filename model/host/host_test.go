package host

import (
	"fmt"
	"testing"
	"time"

	"github.com/evergreen-ci/evergreen"
	"github.com/evergreen-ci/evergreen/db"
	"github.com/evergreen-ci/evergreen/model/distro"
	"github.com/evergreen-ci/evergreen/testutil"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
)

func init() {
	db.SetGlobalSessionProvider(db.SessionFactoryFromConfig(testutil.TestConfig()))
}

func hostIdInSlice(hosts []Host, id string) bool {
	for _, host := range hosts {
		if host.Id == id {
			return true
		}
	}
	return false
}

func TestGenericHostFinding(t *testing.T) {

	Convey("When finding hosts", t, func() {
		testutil.HandleTestingErr(db.Clear(Collection), t, "Error clearing"+
			" '%v' collection", Collection)

		Convey("when finding one host", func() {
			Convey("the matching host should be returned", func() {
				matchingHost := &Host{
					Id: "matches",
				}
				So(matchingHost.Insert(), ShouldBeNil)

				nonMatchingHost := &Host{
					Id: "nonMatches",
				}
				So(nonMatchingHost.Insert(), ShouldBeNil)

				found, err := FindOne(ById(matchingHost.Id))
				So(err, ShouldBeNil)
				So(found.Id, ShouldEqual, matchingHost.Id)

			})
		})

		Convey("when finding multiple hosts", func() {
			dId := fmt.Sprintf("%v.%v", DistroKey, distro.IdKey)

			Convey("the hosts matching the query should be returned", func() {
				matchingHostOne := &Host{
					Id:     "matches",
					Distro: distro.Distro{Id: "d1"},
				}
				So(matchingHostOne.Insert(), ShouldBeNil)

				matchingHostTwo := &Host{
					Id:     "matchesAlso",
					Distro: distro.Distro{Id: "d1"},
				}
				So(matchingHostTwo.Insert(), ShouldBeNil)

				nonMatchingHost := &Host{
					Id:     "nonMatches",
					Distro: distro.Distro{Id: "d2"},
				}
				So(nonMatchingHost.Insert(), ShouldBeNil)

				found, err := Find(db.Query(bson.M{dId: "d1"}))
				So(err, ShouldBeNil)
				So(len(found), ShouldEqual, 2)
				So(hostIdInSlice(found, matchingHostOne.Id), ShouldBeTrue)
				So(hostIdInSlice(found, matchingHostTwo.Id), ShouldBeTrue)

			})

			Convey("when querying two hosts for running tasks", func() {
				matchingHost := &Host{Id: "task", Status: evergreen.HostRunning, RunningTask: "t1"}
				So(matchingHost.Insert(), ShouldBeNil)
				nonMatchingHost := &Host{Id: "nope", Status: evergreen.HostRunning}
				So(nonMatchingHost.Insert(), ShouldBeNil)
				Convey("the host with the running task should be returned", func() {
					found, err := Find(IsRunningTask)
					So(err, ShouldBeNil)
					So(len(found), ShouldEqual, 1)
					So(found[0].Id, ShouldEqual, matchingHost.Id)
				})
			})

			Convey("the specified projection, sort, skip, and limit should be used", func() {
				matchingHostOne := &Host{
					Id:     "matches",
					Host:   "hostOne",
					Distro: distro.Distro{Id: "d1"},
					Tag:    "2",
				}
				So(matchingHostOne.Insert(), ShouldBeNil)

				matchingHostTwo := &Host{
					Id:     "matchesAlso",
					Host:   "hostTwo",
					Distro: distro.Distro{Id: "d1"},
					Tag:    "1",
				}
				So(matchingHostTwo.Insert(), ShouldBeNil)

				matchingHostThree := &Host{
					Id:     "stillMatches",
					Host:   "hostThree",
					Distro: distro.Distro{Id: "d1"},
					Tag:    "3",
				}
				So(matchingHostThree.Insert(), ShouldBeNil)

				// find the hosts, removing the host field from the projection,
				// sorting by tag, skipping one, and limiting to one

				found, err := Find(db.Query(bson.M{dId: "d1"}).
					WithoutFields(DNSKey).
					Sort([]string{TagKey}).
					Skip(1).Limit(1))
				So(err, ShouldBeNil)
				So(len(found), ShouldEqual, 1)
				So(found[0].Id, ShouldEqual, matchingHostOne.Id)
				So(found[0].Host, ShouldEqual, "") // filtered out in projection
			})
		})
	})
}

func TestFindingHostsWithRunningTasks(t *testing.T) {
	Convey("With a host with no running task that is not terminated", t, func() {
		testutil.HandleTestingErr(db.Clear(Collection), t, "Error clearing"+
			" '%v' collection", Collection)
		h := Host{
			Id:     "sample_host",
			Status: evergreen.HostRunning,
		}
		So(h.Insert(), ShouldBeNil)
		found, err := Find(IsRunningTask)
		So(err, ShouldBeNil)
		So(len(found), ShouldEqual, 0)
		Convey("with a host that is terminated with no running task", func() {
			testutil.HandleTestingErr(db.Clear(Collection), t, "Error clearing"+
				" '%v' collection", Collection)
			h1 := Host{
				Id:     "another",
				Status: evergreen.HostTerminated,
			}
			So(h1.Insert(), ShouldBeNil)
			found, err = Find(IsRunningTask)
			So(err, ShouldBeNil)
			So(len(found), ShouldEqual, 0)
		})
	})

}

func TestMonitorHosts(t *testing.T) {
	Convey("With a host with no reachability check", t, func() {
		testutil.HandleTestingErr(db.Clear(Collection), t, "Error clearing"+
			" '%v' collection", Collection)
		now := time.Now()
		h := Host{
			Id:        "sample_host",
			Status:    evergreen.HostRunning,
			StartedBy: evergreen.User,
		}
		So(h.Insert(), ShouldBeNil)
		found, err := Find(ByNotMonitoredSince(now))
		So(err, ShouldBeNil)
		So(len(found), ShouldEqual, 1)
		Convey("a host that has a running task and no reachability check should not return", func() {
			testutil.HandleTestingErr(db.Clear(Collection), t, "Error clearing"+
				" '%v' collection", Collection)
			anotherHost := Host{
				Id:          "anotherHost",
				Status:      evergreen.HostRunning,
				StartedBy:   evergreen.User,
				RunningTask: "id",
			}
			So(anotherHost.Insert(), ShouldBeNil)
			found, err := Find(ByNotMonitoredSince(now))
			So(err, ShouldBeNil)
			So(len(found), ShouldEqual, 0)
		})
	})
}

func TestUpdatingHostStatus(t *testing.T) {

	Convey("With a host", t, func() {
		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		var err error

		host := &Host{
			Id: "hostOne",
		}

		So(host.Insert(), ShouldBeNil)

		Convey("setting the host's status should update both the in-memory"+
			" and database versions of the host", func() {

			So(host.SetStatus(evergreen.HostRunning), ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostRunning)

			host, err = FindOne(ById(host.Id))
			So(err, ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostRunning)

		})

		Convey("if the host is terminated, the status update should fail"+
			" with an error", func() {

			So(host.SetStatus(evergreen.HostTerminated), ShouldBeNil)
			So(host.SetStatus(evergreen.HostRunning), ShouldNotBeNil)
			So(host.Status, ShouldEqual, evergreen.HostTerminated)

			host, err = FindOne(ById(host.Id))
			So(err, ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostTerminated)

		})

	})

}

func TestSetHostTerminated(t *testing.T) {

	Convey("With a host", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		var err error

		host := &Host{
			Id: "hostOne",
		}

		So(host.Insert(), ShouldBeNil)

		Convey("setting the host as terminated should set the status and the"+
			" termination time in both the in-memory and database copies of"+
			" the host", func() {

			So(host.Terminate(), ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostTerminated)
			So(host.TerminationTime.IsZero(), ShouldBeFalse)

			host, err = FindOne(ById(host.Id))
			So(err, ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostTerminated)
			So(host.TerminationTime.IsZero(), ShouldBeFalse)

		})

	})
}

func TestHostSetDNSName(t *testing.T) {
	var err error

	Convey("With a host", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		host := &Host{
			Id: "hostOne",
		}

		So(host.Insert(), ShouldBeNil)

		Convey("setting the hostname should update both the in-memory and"+
			" database copies of the host", func() {

			So(host.SetDNSName("hostname"), ShouldBeNil)
			So(host.Host, ShouldEqual, "hostname")
			host, err = FindOne(ById(host.Id))
			So(err, ShouldBeNil)
			So(host.Host, ShouldEqual, "hostname")

			// if the host is already updated, no new updates should work
			So(host.SetDNSName("hostname2"), ShouldBeNil)
			So(host.Host, ShouldEqual, "hostname")

			host, err = FindOne(ById(host.Id))
			So(err, ShouldBeNil)
			So(host.Host, ShouldEqual, "hostname")

		})

	})
}

func TestMarkAsProvisioned(t *testing.T) {

	Convey("With a host", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		var err error

		host := &Host{
			Id: "hostOne",
		}

		So(host.Insert(), ShouldBeNil)

		Convey("marking the host as provisioned should update the status,"+
			" provisioned, and host name fields in both the in-memory and"+
			" database copies of the host", func() {

			So(host.MarkAsProvisioned(), ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostRunning)
			So(host.Provisioned, ShouldEqual, true)

			host, err = FindOne(ById(host.Id))
			So(err, ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostRunning)
			So(host.Provisioned, ShouldEqual, true)

		})

	})
}

func TestHostCreateSecret(t *testing.T) {
	Convey("With a host with no secret", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t,
			"Error clearing '%v' collection", Collection)

		host := &Host{Id: "hostOne"}
		So(host.Insert(), ShouldBeNil)

		Convey("creating a secret", func() {
			So(host.Secret, ShouldEqual, "")
			So(host.CreateSecret(), ShouldBeNil)

			Convey("should update the host in memory", func() {
				So(host.Secret, ShouldNotEqual, "")

				Convey("and in the database", func() {
					dbHost, err := FindOne(ById(host.Id))
					So(err, ShouldBeNil)
					So(dbHost.Secret, ShouldEqual, host.Secret)
				})
			})
		})
	})
}

func TestHostSetExpirationTime(t *testing.T) {

	Convey("With a host", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		initialExpirationTime := time.Now()
		notifications := make(map[string]bool)
		notifications["2h"] = true

		memHost := &Host{
			Id:             "hostOne",
			ExpirationTime: initialExpirationTime,
			Notifications:  notifications,
		}
		So(memHost.Insert(), ShouldBeNil)

		Convey("setting the expiration time for the host should change the "+
			" expiration time for both the in-memory and database"+
			" copies of the host and unset the notifications", func() {

			dbHost, err := FindOne(ById(memHost.Id))

			// ensure the db entries are as expected
			So(err, ShouldBeNil)
			So(memHost.ExpirationTime.Round(time.Second).Equal(
				initialExpirationTime.Round(time.Second)), ShouldBeTrue)
			So(dbHost.ExpirationTime.Round(time.Second).Equal(
				initialExpirationTime.Round(time.Second)), ShouldBeTrue)
			So(memHost.Notifications, ShouldResemble, notifications)
			So(dbHost.Notifications, ShouldResemble, notifications)

			// now update the expiration time
			newExpirationTime := time.Now()
			So(memHost.SetExpirationTime(newExpirationTime), ShouldBeNil)

			dbHost, err = FindOne(ById(memHost.Id))

			// ensure the db entries are as expected
			So(err, ShouldBeNil)
			So(memHost.ExpirationTime.Round(time.Second).Equal(
				newExpirationTime.Round(time.Second)), ShouldBeTrue)
			So(dbHost.ExpirationTime.Round(time.Second).Equal(
				newExpirationTime.Round(time.Second)), ShouldBeTrue)
			So(memHost.Notifications, ShouldResemble, make(map[string]bool))
			So(dbHost.Notifications, ShouldEqual, nil)
		})
	})
}

func TestFindRunningSpawnedHosts(t *testing.T) {
	testConfig := testutil.TestConfig()
	db.SetGlobalSessionProvider(db.SessionFactoryFromConfig(testConfig))

	testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
		" clearing '%v' collection", Collection)

	Convey("With calling FindRunningSpawnedHosts...", t, func() {
		Convey("if there are no spawned hosts, nothing should be returned",
			func() {
				spawnedHosts, err := Find(IsRunningAndSpawned)
				So(err, ShouldBeNil)
				// make sure we only returned no document
				So(len(spawnedHosts), ShouldEqual, 0)

			})

		Convey("if there are spawned hosts, they should be returned", func() {
			host := &Host{}
			host.Id = "spawned-1"
			host.Status = "running"
			host.StartedBy = "user1"
			testutil.HandleTestingErr(host.Insert(), t, "error from "+
				"FindRunningSpawnedHosts")
			spawnedHosts, err := Find(IsRunningAndSpawned)
			testutil.HandleTestingErr(err, t, "error from "+
				"FindRunningSpawnedHosts: %v", err)
			// make sure we only returned no document
			So(len(spawnedHosts), ShouldEqual, 1)

		})
	})
}
func TestSetExpirationNotification(t *testing.T) {

	Convey("With a host", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		notifications := make(map[string]bool)
		notifications["2h"] = true

		memHost := &Host{
			Id:            "hostOne",
			Notifications: notifications,
		}
		So(memHost.Insert(), ShouldBeNil)

		Convey("setting the expiration notification for the host should change "+
			" the expiration notification for both the in-memory and database"+
			" copies of the host and unset the notifications", func() {

			dbHost, err := FindOne(ById(memHost.Id))

			// ensure the db entries are as expected
			So(err, ShouldBeNil)
			So(memHost.Notifications, ShouldResemble, notifications)
			So(dbHost.Notifications, ShouldResemble, notifications)

			// now update the expiration notification
			notifications["4h"] = true
			So(memHost.SetExpirationNotification("4h"), ShouldBeNil)
			dbHost, err = FindOne(ById(memHost.Id))
			// ensure the db entries are as expected
			So(err, ShouldBeNil)
			So(memHost.Notifications, ShouldResemble, notifications)
			So(dbHost.Notifications, ShouldResemble, notifications)
		})
	})
}

func TestHostClearRunningTask(t *testing.T) {

	Convey("With a host", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		var err error
		var count int

		host := &Host{
			Id:          "hostOne",
			RunningTask: "taskId",
			StartedBy:   evergreen.User,
			Status:      evergreen.HostRunning,
			Pid:         "12345",
		}

		So(host.Insert(), ShouldBeNil)

		Convey("host statistics should properly count this host as active"+
			" but not idle", func() {
			count, err = Count(IsActive)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 1)
			count, err = Count(IsIdle)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)
		})

		Convey("clearing the running task should clear the running task, pid,"+
			" and task dispatch time fields from both the in-memory and"+
			" database copies of the host", func() {

			So(host.ClearRunningTask("prevTask", time.Now()), ShouldBeNil)
			So(host.RunningTask, ShouldEqual, "")
			So(host.LastTaskCompleted, ShouldEqual, "prevTask")

			host, err = FindOne(ById(host.Id))
			So(err, ShouldBeNil)

			So(host.RunningTask, ShouldEqual, "")
			So(host.LastTaskCompleted, ShouldEqual, "prevTask")

			Convey("the count of idle hosts should go up", func() {
				count, err := Count(IsIdle)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 1)

				Convey("but the active host count should remain the same", func() {
					count, err = Count(IsActive)
					So(err, ShouldBeNil)
					So(count, ShouldEqual, 1)
				})
			})

		})

	})
}

func TestUpdateHostRunningTask(t *testing.T) {
	Convey("With a host", t, func() {
		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)
		oldTaskId := "oldId"
		newTaskId := "newId"
		h := Host{
			Id:          "test",
			RunningTask: oldTaskId,
			Status:      evergreen.HostRunning,
		}
		So(h.Insert(), ShouldBeNil)
		Convey("updating the running task id should set proper fields", func() {
			_, err := h.UpdateRunningTask(oldTaskId, newTaskId, time.Now())
			So(err, ShouldBeNil)
			found, err := FindOne(ById(h.Id))
			So(err, ShouldBeNil)
			So(found.RunningTask, ShouldEqual, newTaskId)
			So(found.LastTaskCompleted, ShouldEqual, oldTaskId)
			runningTaskHosts, err := Find(IsRunningTask)
			So(err, ShouldBeNil)
			So(len(runningTaskHosts), ShouldEqual, 1)
		})
		Convey("updating the running task to an empty string should error out", func() {
			_, err := h.UpdateRunningTask(newTaskId, "", time.Now())
			So(err, ShouldNotBeNil)
		})
	})
}

func TestUpsert(t *testing.T) {

	Convey("With a host", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		host := &Host{
			Id:     "hostOne",
			Host:   "host",
			User:   "user",
			Distro: distro.Distro{Id: "distro"},
			Status: evergreen.HostRunning,
		}

		var err error

		Convey("Performing a host upsert should upsert correctly", func() {
			_, err = host.Upsert()
			So(err, ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostRunning)

			host, err = FindOne(ById(host.Id))
			So(err, ShouldBeNil)
			So(host.Status, ShouldEqual, evergreen.HostRunning)

		})

		Convey("Updating some fields of an already inserted host should cause "+
			"those fields to be updated but should leave status unchanged",
			func() {
				_, err := host.Upsert()
				So(err, ShouldBeNil)
				So(host.Status, ShouldEqual, evergreen.HostRunning)

				host, err = FindOne(ById(host.Id))
				So(err, ShouldBeNil)
				So(host.Status, ShouldEqual, evergreen.HostRunning)
				So(host.Host, ShouldEqual, "host")

				err = UpdateOne(
					bson.M{
						IdKey: host.Id,
					},
					bson.M{
						"$set": bson.M{
							StatusKey: evergreen.HostDecommissioned,
						},
					},
				)
				So(err, ShouldBeNil)

				// update the hostname and status
				host.Host = "host2"
				host.Status = evergreen.HostRunning
				_, err = host.Upsert()
				So(err, ShouldBeNil)

				// host db status should remain unchanged
				host, err = FindOne(ById(host.Id))
				So(err, ShouldBeNil)
				So(host.Status, ShouldEqual, evergreen.HostDecommissioned)
				So(host.Host, ShouldEqual, "host2")

			})
	})
}

func TestDecommissionHostsWithDistroId(t *testing.T) {

	Convey("With a multiple hosts of different distros", t, func() {

		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)

		distroA := "distro_a"
		distroB := "distro_b"

		// Insert 10 of distro a and 10 of distro b

		for i := 0; i < 10; i++ {
			hostWithDistroA := &Host{
				Id:     fmt.Sprintf("hostA%v", i),
				Host:   "host",
				User:   "user",
				Distro: distro.Distro{Id: distroA},
				Status: evergreen.HostRunning,
			}
			hostWithDistroB := &Host{
				Id:     fmt.Sprintf("hostB%v", i),
				Host:   "host",
				User:   "user",
				Distro: distro.Distro{Id: distroB},
				Status: evergreen.HostRunning,
			}

			testutil.HandleTestingErr(hostWithDistroA.Insert(), t, "Error inserting"+
				"host into database")
			testutil.HandleTestingErr(hostWithDistroB.Insert(), t, "Error inserting"+
				"host into database")
		}

		Convey("When decommissioning hosts of type distro_a", func() {
			err := DecommissionHostsWithDistroId(distroA)
			So(err, ShouldBeNil)

			Convey("Distro should be marked as decommissioned accordingly", func() {
				hostsTypeA, err := Find(ByDistroId(distroA))
				So(err, ShouldBeNil)

				hostsTypeB, err := Find(ByDistroId(distroB))
				So(err, ShouldBeNil)
				for _, host := range hostsTypeA {

					So(host.Status, ShouldEqual, evergreen.HostDecommissioned)
				}

				for _, host := range hostsTypeB {
					So(host.Status, ShouldEqual, evergreen.HostRunning)

				}
			})

		})

	})
}

func TestFindByLCT(t *testing.T) {
	Convey("with the a given time for checking and an empty hosts collection", t, func() {
		testutil.HandleTestingErr(db.Clear(Collection), t, "Error"+
			" clearing '%v' collection", Collection)
		now := time.Now()
		Convey("with a host that has no last communication time", func() {
			h := Host{
				Id:     "id",
				Status: evergreen.HostRunning,
			}
			So(h.Insert(), ShouldBeNil)
			hosts, err := Find(ByRunningWithTimedOutLCT(time.Now()))
			So(err, ShouldBeNil)
			So(len(hosts), ShouldEqual, 1)
			So(hosts[0].Id, ShouldEqual, "id")
		})

		Convey("with a host with a last communication time > 10 mins", func() {
			anotherHost := Host{
				Id: "anotherID",
				LastCommunicationTime: now.Add(-time.Duration(20) * time.Minute),
				Status:                evergreen.HostRunning,
			}
			So(anotherHost.Insert(), ShouldBeNil)
			hosts, err := Find(ByRunningWithTimedOutLCT(now))
			So(err, ShouldBeNil)
			So(len(hosts), ShouldEqual, 1)
			So(hosts[0].Id, ShouldEqual, anotherHost.Id)
		})

		Convey("with a host with a normal LCT", func() {
			anotherHost := Host{
				Id: "testhost",
				LastCommunicationTime: now.Add(time.Duration(5) * time.Minute),
				Status:                evergreen.HostRunning,
			}
			So(anotherHost.Insert(), ShouldBeNil)
			hosts, err := Find(ByRunningWithTimedOutLCT(now))
			So(err, ShouldBeNil)
			So(len(hosts), ShouldEqual, 0)
			Convey("after resetting the LCT", func() {
				So(anotherHost.ResetLastCommunicated(), ShouldBeNil)
				So(anotherHost.LastCommunicationTime, ShouldResemble, time.Unix(0, 0))
				h, err := Find(ByRunningWithTimedOutLCT(now))
				So(err, ShouldBeNil)
				So(len(h), ShouldEqual, 1)
				So(h[0].Id, ShouldEqual, "testhost")
			})
		})
		Convey("with a terminated host that has no LCT", func() {
			h := Host{
				Id:     "h",
				Status: evergreen.HostTerminated,
			}
			So(h.Insert(), ShouldBeNil)
			hosts, err := Find(ByRunningWithTimedOutLCT(now))
			So(err, ShouldBeNil)
			So(len(hosts), ShouldEqual, 0)
		})

	})
}
