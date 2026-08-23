package audit

import (
	"testing"

	"task189-cfdbc/internal/model"
)

func TestClassifyUnderconstrained(t *testing.T) {
	issues := []*model.Issue{
		{Type: "missing_bc", Severity: model.SeverityError},
		{Type: "missing_reference_pressure", Severity: model.SeverityError},
	}
	if got := Classify(issues); got != ResultUnderconstrained {
		t.Fatalf("expected underconstrained, got %s", got)
	}
}

func TestClassifyOverconstrained(t *testing.T) {
	issues := []*model.Issue{
		{Type: "flow_imbalance", Severity: model.SeverityError},
		{Type: "unconserved_interface", Severity: model.SeverityError},
	}
	if got := Classify(issues); got != ResultOverconstrained {
		t.Fatalf("expected overconstrained, got %s", got)
	}
}

func TestClassifySolvable(t *testing.T) {
	if got := Classify(nil); got != ResultSolvable {
		t.Fatalf("expected solvable, got %s", got)
	}
}

func TestRunBuilderCounts(t *testing.T) {
	b := NewRunBuilder("run-1", "m1", 3)
	b.Add("missing_bc", model.SeverityError, "r1", "f1", "x")
	b.Add("flow_imbalance", model.SeverityWarning, "", "", "y")
	errs, warns := b.Counts()
	if errs != 1 || warns != 1 {
		t.Fatalf("expected 1/1, got %d/%d", errs, warns)
	}
}
