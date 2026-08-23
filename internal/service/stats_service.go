package service

import (
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/store"
)

// StatsService 统计汇总（健康检查与概览）。
type StatsService struct {
	db *store.DB
}

// NewStatsService 构造统计服务。
func NewStatsService(db *store.DB) *StatsService { return &StatsService{db: db} }

// Stats 统计输出。
type Stats struct {
	Regions      map[string]int `json:"regions"`
	Faces        int            `json:"faces"`
	Conditions   map[string]int `json:"conditions"`
	Packages     map[string]int `json:"packages"`
	ValidationRuns int          `json:"validation_runs"`
	ConfigVersion int           `json:"config_version"`
}

// Collect 收集统计。
func (s *StatsService) Collect() (*Stats, error) {
	rs, err := store.NewRegionStore(s.db).CountByStatus()
	if err != nil {
		return nil, err
	}
	faces, err := store.NewFaceStore(s.db).ListAll()
	if err != nil {
		return nil, err
	}
	bc, err := store.NewBCStore(s.db).CountByType()
	if err != nil {
		return nil, err
	}
	ps, err := store.NewPackageStore(s.db).CountByStatus()
	if err != nil {
		return nil, err
	}
	runs, err := store.NewRunStore(s.db).ListRuns(10000)
	if err != nil {
		return nil, err
	}
	cv, err := s.db.NextConfigVersion()
	if err != nil {
		return nil, err
	}
	return &Stats{
		Regions:        regionCounts(rs),
		Faces:          len(faces),
		Conditions:     bcCounts(bc),
		Packages:       pkgCounts(ps),
		ValidationRuns: len(runs),
		ConfigVersion:  cv - 1,
	}, nil
}

func regionCounts(m map[model.RegionStatus]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[string(k)] = v
	}
	return out
}

func bcCounts(m map[model.BCType]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[string(k)] = v
	}
	return out
}

func pkgCounts(m map[model.PackageStatus]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[string(k)] = v
	}
	return out
}
