package parsing

import (
	"io/ioutil"
	"sync"
	"testing"
)

func TestInvalidLinesSouldFailParsing(t *testing.T) {
	cases := []struct{ in, errorMessage string }{
		{"2013-10-23T10:12:35.271Z 2013-10-23T10:12:35.471Z eckakaau service6null->bm6il56t", "Missing value"},
		{"2013-10-23T10:12:35.271Z 2013-10-23T10:12:35.471Z eckakaau service6 null->bm6il56t ", "Too many fields"},
		{"2013-10-23T10:12:35.271Z 2013-10-23T10:12:35.471Z eckakaau service6 nullbm6il56t ", "Missing ->"},
	}

	for _, c := range cases {
		parseSucceeded, _ := tryParseLine(c.in)
		if parseSucceeded {
			t.Errorf("Test %q failed", c.errorMessage)
		}
	}
}

func TestValidParsing(t *testing.T) {
	in := "2013-10-23T10:12:35.271Z 2013-10-23T10:12:35.471Z eckakaau service6 null->bm6il56t"
	out := LogLine{
		Start:       "2013-10-23 10:12:35.271",
		End:         "2013-10-23 10:12:35.471",
		Id:          "eckakaau",
		ServiceName: "service6",
		Caller:      "null",
		Callee:      "bm6il56t",
	}

	parseSucceeded, logLine := tryParseLine(in)

	if !parseSucceeded {
		t.Error("Parsing faiiled")
	}

	if logLine.Callee != out.Callee {
		t.Error("Invalid callee")
	}

	if logLine.Caller != out.Caller {
		t.Error("Invalid caller")
	}

	if logLine.ServiceName != out.ServiceName {
		t.Error("Invalid ServiceName")
	}

	if logLine.Id != out.Id {
		t.Error("Invalid id")
	}

	if logLine.Start != out.Start {
		t.Error("Invalid end date")
	}

	if logLine.End != out.End {
		t.Error("Invalid start date")
	}

}

func TestBuildTree(t *testing.T) {
	testLog := []string{
		"2013-10-23T10:12:35.298Z 2013-10-23T10:12:35.300Z eckakaau service3 d6m3shqy->62d45qeh",
		"2013-10-23T10:12:35.293Z 2013-10-23T10:12:35.302Z eckakaau service7 zfjlsiev->d6m3shqy",
		"2013-10-23T10:12:35.286Z 2013-10-23T10:12:35.302Z eckakaau service9 bm6il56t->zfjlsiev",
		"2013-10-23T10:12:35.339Z 2013-10-23T10:12:35.339Z eckakaau service1 nhxtegwv->4zhimp35",
		"2013-10-23T10:12:35.339Z 2013-10-23T10:12:35.342Z eckakaau service1 22buxmqp->nhxtegwv",
		"2013-10-23T10:12:35.345Z 2013-10-23T10:12:35.361Z eckakaau service5 22buxmqp->3wos67cv",
		"2013-10-23T10:12:35.318Z 2013-10-23T10:12:35.370Z eckakaau service3 bm6il56t->22buxmqp",
		"2013-10-23T10:12:35.271Z 2013-10-23T10:12:35.471Z eckakaau service6 null->bm6il56t",
		"2013-10-23T10:12:35.318Z 2013-10-23T10:12:35.370Z ddeekkk service3 bm6il56t->22buxmqp",
	}

	expectedMalformedLog := "2013-10-23T10:12:35.318Z 2013-10-23T10:12:35.370Z malformed service3 bm6il56t22buxmqp"
	testLog = append(testLog, expectedMalformedLog)
	expectedLog, err := ioutil.ReadFile("../resources/TestBuildTree.json")
	expectedOrphanLog, err := ioutil.ReadFile("../resources/TestBuildTreeOrphan.json")
	if err != nil {
		t.Error("Failed to read test data")
	}

	lines := make(chan string)
	malformedLines := make(chan string)
	transformedLogs := make(chan string)
	orphanLogs := make(chan string)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for _, value := range testLog {
			lines <- value
		}
		lines <- StopSignal

	}()
	go TransformLogs(lines, malformedLines, transformedLogs, orphanLogs)
	go func() {
		defer wg.Done()
		for {
			select {
			case transformed := <-transformedLogs:
				if transformed == StopSignal {
					return
				}

				if string(expectedLog) != transformed {
					t.Errorf("Expected log %q, got %q", string(expectedLog), transformed)
				}
			case orphan := <-orphanLogs:
				if orphan != StopSignal && string(expectedOrphanLog) != orphan {
					t.Errorf("Expected log %q, got %q", string(expectedOrphanLog), orphan)
				}

			case malformedLog := <-malformedLines:
				if expectedMalformedLog != malformedLog {
					t.Errorf("Expected log %q, got %q", expectedMalformedLog, malformedLog)
				}
			}
		}
	}()
	wg.Wait()
}
