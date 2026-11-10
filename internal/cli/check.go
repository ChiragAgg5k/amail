package cli

import "github.com/ChiragAgg5k/mailslot/internal/failure"

// checkStatus is the outcome of one doctor check.
type checkStatus string

const (
	checkPassed  checkStatus = "ok"
	checkFailed  checkStatus = "fail"
	checkWarned  checkStatus = "warn"
	checkSkipped checkStatus = "skip"
)

// check is one line of the doctor's report.
type check struct {
	Name   string      `json:"name"`
	Status checkStatus `json:"status"`
	Detail string      `json:"detail"`
	Hint   string      `json:"hint,omitempty"`
	// code is the exit code a failed check stands for.
	code failure.Code
}

func passed(name, detail string) check {
	return check{Name: name, Status: checkPassed, Detail: detail}
}

func skipped(name, detail string) check {
	return check{Name: name, Status: checkSkipped, Detail: detail}
}

func warned(name, detail, hint string) check {
	return check{Name: name, Status: checkWarned, Detail: detail, Hint: hint}
}

// failed turns an error into a failed check that carries its exit code and hint.
func failed(name string, err error) check {
	return check{Name: name, Status: checkFailed, Detail: err.Error(), Hint: failure.HintOf(err), code: failure.CodeOf(err)}
}
