package conservation

import (
	"testing"

	"task189-cfdbc/internal/model"
)

func TestCheckFlowBalanceMassFlow(t *testing.T) {
	bcs := []*model.BC{
		{Type: model.BCInletMassFlow, Unit: model.UnitSI, Value: 2.0},
		{Type: model.BCOutletMassFlow, Unit: model.UnitSI, Value: 1.99},
	}
	bal := CheckFlowBalance(bcs, 0.01)
	if !bal.Balanced {
		t.Fatalf("expected balanced, inflow=%.3f outflow=%.3f rel_err=%.3f", bal.Inflow, bal.Outflow, bal.RelErr)
	}
}

func TestCheckFlowBalanceImbalanced(t *testing.T) {
	bcs := []*model.BC{
		{Type: model.BCInletMassFlow, Unit: model.UnitSI, Value: 2.0},
		{Type: model.BCOutletMassFlow, Unit: model.UnitSI, Value: 1.0},
	}
	bal := CheckFlowBalance(bcs, 0.01)
	if bal.Balanced {
		t.Fatalf("expected imbalanced")
	}
}

func TestCheckFlowBalanceUnitConversion(t *testing.T) {
	// 1.5 kg/s（SI）与 1500 g/s（CGS）等价
	bcs := []*model.BC{
		{Type: model.BCInletMassFlow, Unit: model.UnitSI, Value: 1.5},
		{Type: model.BCOutletMassFlow, Unit: model.UnitCGS, Value: 1500},
	}
	bal := CheckFlowBalance(bcs, 0.001)
	if !bal.Balanced {
		t.Fatalf("expected balanced after unit conversion, inflow=%.4f outflow=%.4f", bal.Inflow, bal.Outflow)
	}
}

func TestCheckReferencePressure(t *testing.T) {
	m := &model.PhysicsModel{FlowType: model.FlowIncompressible, ReferencePressureRequired: true}
	chk := CheckReferencePressure(m, true)
	if chk.Missing {
		t.Fatal("expected fulfilled")
	}
	chk2 := CheckReferencePressure(m, false)
	if !chk2.Missing {
		t.Fatal("expected missing")
	}
}

func TestCheckInterfaceUnitMismatch(t *testing.T) {
	face := &model.Face{ID: "f1", Kind: model.KindInterface}
	sideA := &model.BC{Type: model.BCInterfaceSideA, Unit: model.UnitSI, Value: 1.5}
	sideB := &model.BC{Type: model.BCInterfaceSideB, Unit: model.UnitCGS, Value: 150}
	res := CheckInterface(FromBCs(face, sideA, sideB), 0.01)
	if res.Conserved || res.UnitOK {
		t.Fatalf("expected unit mismatch, got %+v", res)
	}
}

func TestCheckInterfaceMissingSide(t *testing.T) {
	face := &model.Face{ID: "f1", Kind: model.KindInterface}
	res := CheckInterface(FromBCs(face, nil, nil), 0.01)
	if res.Conserved {
		t.Fatal("expected unconserved")
	}
}
