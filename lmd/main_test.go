package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestMainFunc(t *testing.T) {
	peer := StartTestPeer(1, 0, 0)
	PauseTestPeers(peer)

	res, err := peer.QueryString("GET backends\n\n")
	if err != nil {
		t.Fatal(err)
	}
	if err = assertEq("peer_key", res[0][0]); err != nil {
		t.Fatal(err)
	}
	if err = assertEq("mockid0", res[1][0]); err != nil {
		t.Fatal(err)
	}

	testRequestStrings := []string{
		"GET backends\n\n",
		"GET backends\nResponseHeader: fixed16\n\n",
		"GET backends\nResponseHeader: fixed16\nOutputFormat: json\n\n",
		"GET backends\nResponseHeader: fixed16\nOutputFormat: wrapped_json\n\n",
		"GET backends\nResponseHeader: fixed16\nFilter: peer_key = mockid0\n\n",
		"GET backends\nResponseHeader: fixed16\nFilter: peer_key = mockid0\n\n",
		"GET backends\nResponseHeader: fixed16\nFilter: peer_key ~~ mockid0\n\n",
		"GET backends\nResponseHeader: fixed16\nFilter: peer_key =~ mockid0\n\n",
		"GET backends\nResponseHeader: fixed16\nFilter: peer_key !=\n\n",
		"GET backends\nResponseHeader: fixed16\nFilter: peer_key != id2\n\n",
		"GET backends\nResponseHeader: fixed16\nFilter: peer_key !=~ id2\n\n",
		"GET backends\nResponseHeader: fixed16\nSort: peer_key asc\n\n",
		"GET backends\nResponseHeader: fixed16\nSort: peer_key desc\n\n",
		"GET backends\nResponseHeader: fixed16\nSort: bytes_send asc\nSort: bytes_received desc\n\n",
	}
	for _, str := range testRequestStrings {
		res, err = peer.QueryString(str)
		if err != nil {
			t.Fatal(err)
		}
		if err = assertEq("peer_key", res[0][0]); err != nil {
			t.Fatal(err)
		}
		if err = assertEq("mockid0", res[1][0]); err != nil {
			t.Fatal(err)
		}
	}

	// sort querys
	res, err = peer.QueryString("GET backends\nColumns: peer_key bytes_send bytes_received\nSort: bytes_send asc\nSort: bytes_received desc\n\n")
	if err != nil {
		t.Fatal(err)
	}
	if err = assertEq("mockid0", res[0][0]); err != nil {
		t.Fatal(err)
	}

	// stats querys
	res, err = peer.QueryString("GET backends\nStats: bytes_send > 0\nStats: avg bytes_send\nStats: sum bytes_send\nStats: min bytes_send\nStats: max bytes_send\n\n")
	if err != nil {
		t.Fatal(err)
	}
	if err = assertEq(1.0, res[0][0]); err != nil {
		t.Fatal(err)
	}

	// send commands
	_, err = peer.QueryString("COMMAND [123456] TEST\n\n")
	if err != nil {
		t.Fatal(err)
	}

	if err := StopTestPeer(peer); err != nil {
		panic(err.Error())
	}
}

func TestMainReload(t *testing.T) {
	StartMockMainLoop([]string{"mock0.sock"}, "")
	mainSignalChannel <- syscall.SIGHUP
	waitTimeout(TestPeerWaitGroup, 5*time.Second)
}

func TestAllOps(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping all ops test in short mode")
	}
	peer := StartTestPeer(1, 0, 0)
	PauseTestPeers(peer)

	ops := []string{"=", "!=", "=~", "!=~", "~", "!~", "~~", "!~~", "<", "<=", ">", ">=", "!>="}
	values := []string{"", " test", " 5", " 3.124", "{}"}

	res, err := peer.QueryString("GET columns\nColumns: table name description\n\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range res {
		if row[2].(string) == "" {
			t.Fatalf("got no description for %s in %s", row[1].(string), row[0].(string))
		}
		for _, op := range ops {
			for _, value := range values {
				testquery(t, peer, row[0].(string), row[1].(string), op, value)
			}
		}
	}

	if err := StopTestPeer(peer); err != nil {
		panic(err.Error())
	}
}

func testquery(t *testing.T, peer *Peer, table, column, op, value string) {
	query := fmt.Sprintf("GET %s\nColumns: %s\nFilter: %s %s%s\nSort: %s asc\n\n",
		table,
		column,
		column,
		op,
		value,
		column,
	)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
			t.Fatalf("paniced for query:\n%s", query)
		}
	}()
	buf := bufio.NewReader(bytes.NewBufferString(query))
	req, _, err := NewRequest(buf)
	if err == nil {
		if err = assertEq(query, req.String()); err != nil {
			t.Fatal(err)
		}
	}
	peer.QueryString(query)
}

func TestMainConfig(t *testing.T) {
	testConfig := []string{
		`Loglevel = "Warn"`,
		`Listen = ["test1.sock"]`,
		`Listen = ["test2.sock"]`,
	}

	ioutil.WriteFile("test1.ini", []byte(testConfig[0]), 0644)
	ioutil.WriteFile("test2.ini", []byte(testConfig[1]), 0644)
	ioutil.WriteFile("test3.ini", []byte(testConfig[2]), 0644)

	conf := ReadConfig([]string{"test1.ini"})
	if err := assertEq(len(conf.Listen), 0); err != nil {
		t.Error(err)
	}

	conf = ReadConfig([]string{"test1.ini", "test2.ini"})
	if err := assertEq(len(conf.Listen), 1); err != nil {
		t.Error(err)
	}

	conf = ReadConfig([]string{"test1.ini", "test2.ini", "test3.ini"})
	if err := assertEq(len(conf.Listen), 2); err != nil {
		t.Error(err)
	}

	os.Remove("test1.ini")
	os.Remove("test2.ini")
	os.Remove("test3.ini")
}
