package profiling

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Register pprof handlers
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

// ProfileType represents different types of profiling
type ProfileType string

const (
	ProfileTypeCPU     ProfileType = "cpu"
	ProfileTypeMemory  ProfileType = "memory"
	ProfileTypeGoroutine ProfileType = "goroutine"
	ProfileTypeBlock   ProfileType = "block"
	ProfileTypeMutex   ProfileType = "mutex"
	ProfileTypeTrace   ProfileType = "trace"
)

// ProfilingConfig holds configuration for performance profiling
type ProfilingConfig struct {
	// EnableCPUProfiling enables CPU profiling
	EnableCPUProfiling bool
	// EnableMemoryProfiling enables memory profiling
	EnableMemoryProfiling bool
	// EnableBlockProfiling enables block profiling
	EnableBlockProfiling bool
	// EnableMutexProfiling enables mutex profiling
	EnableMutexProfiling bool
	// SampleRate for profiling (0 = default, 1 = all operations)
	SampleRate int
	// ProfileDuration for timed profiles
	ProfileDuration time.Duration
	// OutputDir for profile files
	OutputDir string
	// HTTPEndpoint enables pprof HTTP endpoint
	HTTPEndpoint string
	// AutoProfile enables automatic profiling triggers
	AutoProfile bool
	// ProfileThresholds for automatic profiling
	ProfileThresholds ProfileThresholds
}

// ProfileThresholds defines thresholds for automatic profiling
type ProfileThresholds struct {
	// CPUThreshold triggers CPU profiling when CPU usage exceeds this percentage
	CPUThreshold float64
	// MemoryThreshold triggers memory profiling when memory usage exceeds this (in bytes)
	MemoryThreshold uint64
	// GoroutineThreshold triggers goroutine profiling when count exceeds this
	GoroutineThreshold int
	// OperationLatencyThreshold triggers profiling when operations exceed this duration
	OperationLatencyThreshold time.Duration
}

// DefaultProfilingConfig returns a default profiling configuration
func DefaultProfilingConfig() ProfilingConfig {
	return ProfilingConfig{
		EnableCPUProfiling:    true,
		EnableMemoryProfiling: true,
		EnableBlockProfiling:  false, // Can be expensive
		EnableMutexProfiling:  false, // Can be expensive
		SampleRate:           0, // Use default
		ProfileDuration:      time.Second * 30,
		OutputDir:           "./profiles",
		HTTPEndpoint:        "", // Disabled by default
		AutoProfile:         false,
		ProfileThresholds: ProfileThresholds{
			CPUThreshold:              80.0,
			MemoryThreshold:           500 * 1024 * 1024, // 500MB
			GoroutineThreshold:        10000,
			OperationLatencyThreshold: time.Second * 5,
		},
	}
}

// Profiler manages performance profiling for the ENCX library
type Profiler struct {
	config          ProfilingConfig
	isRunning       bool
	profileSessions map[ProfileType]*ProfileSession
	mutex           sync.RWMutex
	httpServer      *http.Server

	// Metrics for auto-profiling
	operationCount   int64
	totalLatency     time.Duration
	profileTriggers  map[string]int64
}

// ProfileSession represents an active profiling session
type ProfileSession struct {
	Type      ProfileType `json:"type"`
	StartTime time.Time   `json:"start_time"`
	Duration  time.Duration `json:"duration"`
	FilePath  string      `json:"file_path,omitempty"`
	Active    bool        `json:"active"`
}

// NewProfiler creates a new profiler instance
func NewProfiler(config ProfilingConfig) *Profiler {
	return &Profiler{
		config:          config,
		profileSessions: make(map[ProfileType]*ProfileSession),
		profileTriggers: make(map[string]int64),
	}
}

// Start initializes and starts the profiler
func (p *Profiler) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isRunning {
		return fmt.Errorf("profiler is already running")
	}

	// Set up runtime profiling configurations
	if p.config.EnableBlockProfiling {
		runtime.SetBlockProfileRate(1)
	}

	if p.config.EnableMutexProfiling {
		runtime.SetMutexProfileFraction(1)
	}

	// Start HTTP endpoint for pprof if configured
	if p.config.HTTPEndpoint != "" {
		p.httpServer = &http.Server{
			Addr: p.config.HTTPEndpoint,
		}

		go func() {
			if err := p.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Profiler HTTP server error: %v\n", err)
			}
		}()
	}

	p.isRunning = true
	return nil
}

// Stop stops the profiler and cleans up resources
func (p *Profiler) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.isRunning {
		return nil
	}

	// Stop any active profiling sessions
	for profileType, session := range p.profileSessions {
		if session.Active {
			p.stopProfile(profileType)
		}
	}

	// Stop HTTP server if running
	if p.httpServer != nil {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		p.httpServer.Shutdown(ctx)
		p.httpServer = nil
	}

	// Reset runtime profiling
	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(0)

	p.isRunning = false
	return nil
}

// StartProfile starts profiling for a specific type
func (p *Profiler) StartProfile(profileType ProfileType) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.isRunning {
		return fmt.Errorf("profiler is not running")
	}

	if session, exists := p.profileSessions[profileType]; exists && session.Active {
		return fmt.Errorf("profile type %s is already active", profileType)
	}

	session := &ProfileSession{
		Type:      profileType,
		StartTime: time.Now(),
		Duration:  p.config.ProfileDuration,
		Active:    true,
	}

	var err error
	switch profileType {
	case ProfileTypeCPU:
		err = p.startCPUProfile(session)
	case ProfileTypeMemory:
		err = p.startMemoryProfile(session)
	case ProfileTypeGoroutine:
		err = p.startGoroutineProfile(session)
	case ProfileTypeBlock:
		if !p.config.EnableBlockProfiling {
			return fmt.Errorf("block profiling is not enabled")
		}
		err = p.startBlockProfile(session)
	case ProfileTypeMutex:
		if !p.config.EnableMutexProfiling {
			return fmt.Errorf("mutex profiling is not enabled")
		}
		err = p.startMutexProfile(session)
	default:
		return fmt.Errorf("unsupported profile type: %s", profileType)
	}

	if err != nil {
		return fmt.Errorf("failed to start %s profile: %w", profileType, err)
	}

	p.profileSessions[profileType] = session

	// Auto-stop after duration
	go func() {
		time.Sleep(session.Duration)
		p.StopProfile(profileType)
	}()

	return nil
}

// StopProfile stops profiling for a specific type
func (p *Profiler) StopProfile(profileType ProfileType) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.stopProfile(profileType)
}

// stopProfile stops profiling (internal, assumes lock is held)
func (p *Profiler) stopProfile(profileType ProfileType) error {
	session, exists := p.profileSessions[profileType]
	if !exists || !session.Active {
		return fmt.Errorf("no active profile session for type %s", profileType)
	}

	var err error
	switch profileType {
	case ProfileTypeCPU:
		pprof.StopCPUProfile()
	case ProfileTypeMemory, ProfileTypeGoroutine, ProfileTypeBlock, ProfileTypeMutex:
		err = p.writeProfile(profileType)
	}

	session.Active = false
	session.Duration = time.Since(session.StartTime)

	return err
}

// startCPUProfile starts CPU profiling
func (p *Profiler) startCPUProfile(session *ProfileSession) error {
	filename := p.getProfileFilename(ProfileTypeCPU)
	session.FilePath = filename

	file, err := createProfileFile(filename)
	if err != nil {
		return err
	}

	return pprof.StartCPUProfile(file)
}

// startMemoryProfile starts memory profiling
func (p *Profiler) startMemoryProfile(session *ProfileSession) error {
	// Memory profiling is always active in Go, just need to write it out later
	session.FilePath = p.getProfileFilename(ProfileTypeMemory)
	return nil
}

// startGoroutineProfile starts goroutine profiling
func (p *Profiler) startGoroutineProfile(session *ProfileSession) error {
	session.FilePath = p.getProfileFilename(ProfileTypeGoroutine)
	return nil
}

// startBlockProfile starts block profiling
func (p *Profiler) startBlockProfile(session *ProfileSession) error {
	session.FilePath = p.getProfileFilename(ProfileTypeBlock)
	return nil
}

// startMutexProfile starts mutex profiling
func (p *Profiler) startMutexProfile(session *ProfileSession) error {
	session.FilePath = p.getProfileFilename(ProfileTypeMutex)
	return nil
}

// writeProfile writes a profile to disk
func (p *Profiler) writeProfile(profileType ProfileType) error {
	session := p.profileSessions[profileType]
	if session == nil {
		return fmt.Errorf("no session found for profile type %s", profileType)
	}

	file, err := createProfileFile(session.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var profile *pprof.Profile
	switch profileType {
	case ProfileTypeMemory:
		runtime.GC() // Force GC before memory profile
		profile = pprof.Lookup("heap")
	case ProfileTypeGoroutine:
		profile = pprof.Lookup("goroutine")
	case ProfileTypeBlock:
		profile = pprof.Lookup("block")
	case ProfileTypeMutex:
		profile = pprof.Lookup("mutex")
	default:
		return fmt.Errorf("unsupported profile type for writing: %s", profileType)
	}

	if profile == nil {
		return fmt.Errorf("profile %s not found", profileType)
	}

	return profile.WriteTo(file, 0)
}

// getProfileFilename generates a filename for a profile
func (p *Profiler) getProfileFilename(profileType ProfileType) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s/encx-%s-%s.prof", p.config.OutputDir, profileType, timestamp)
}

// ProfileOperation profiles a specific operation
func (p *Profiler) ProfileOperation(ctx context.Context, operationName string, operation func() error) error {
	if !p.isRunning {
		return operation() // Just run the operation if profiler is not running
	}

	startTime := time.Now()

	// Check if we should trigger auto-profiling based on thresholds
	if p.config.AutoProfile {
		p.checkAutoProfileTriggers(operationName)
	}

	// Execute the operation
	err := operation()

	// Record operation metrics
	duration := time.Since(startTime)
	p.recordOperationMetrics(operationName, duration)

	// Check if we should trigger profiling based on operation latency
	if p.config.AutoProfile && duration > p.config.ProfileThresholds.OperationLatencyThreshold {
		p.triggerLatencyProfiling(operationName, duration)
	}

	return err
}

// recordOperationMetrics records metrics for operations
func (p *Profiler) recordOperationMetrics(operationName string, duration time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.operationCount++
	p.totalLatency += duration
}

// checkAutoProfileTriggers checks if profiling should be triggered based on system metrics
func (p *Profiler) checkAutoProfileTriggers(operationName string) {
	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.Alloc > p.config.ProfileThresholds.MemoryThreshold {
		p.triggerAutoProfiling("memory_threshold", ProfileTypeMemory)
	}

	// Check goroutine count
	if runtime.NumGoroutine() > p.config.ProfileThresholds.GoroutineThreshold {
		p.triggerAutoProfiling("goroutine_threshold", ProfileTypeGoroutine)
	}
}

// triggerLatencyProfiling triggers profiling when operation latency is high
func (p *Profiler) triggerLatencyProfiling(operationName string, duration time.Duration) {
	triggerKey := fmt.Sprintf("latency_%s", operationName)
	p.triggerAutoProfiling(triggerKey, ProfileTypeCPU)
}

// triggerAutoProfiling triggers automatic profiling if not already active
func (p *Profiler) triggerAutoProfiling(triggerReason string, profileType ProfileType) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if this profile type is already active
	if session, exists := p.profileSessions[profileType]; exists && session.Active {
		return
	}

	// Increment trigger count
	p.profileTriggers[triggerReason]++

	// Start profiling (unlock first to avoid deadlock)
	p.mutex.Unlock()
	go p.StartProfile(profileType)
	p.mutex.Lock()
}

// GetActiveProfiles returns information about active profiling sessions
func (p *Profiler) GetActiveProfiles() map[ProfileType]*ProfileSession {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	active := make(map[ProfileType]*ProfileSession)
	for profileType, session := range p.profileSessions {
		if session.Active {
			sessionCopy := *session
			active[profileType] = &sessionCopy
		}
	}

	return active
}

// GetProfileHistory returns information about all profiling sessions
func (p *Profiler) GetProfileHistory() map[ProfileType]*ProfileSession {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	history := make(map[ProfileType]*ProfileSession)
	for profileType, session := range p.profileSessions {
		sessionCopy := *session
		history[profileType] = &sessionCopy
	}

	return history
}

// GetStats returns profiling statistics
func (p *Profiler) GetStats() ProfilerStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var activeCount int
	for _, session := range p.profileSessions {
		if session.Active {
			activeCount++
		}
	}

	var avgLatency time.Duration
	if p.operationCount > 0 {
		avgLatency = p.totalLatency / time.Duration(p.operationCount)
	}

	return ProfilerStats{
		IsRunning:         p.isRunning,
		ActiveProfiles:    activeCount,
		TotalSessions:     len(p.profileSessions),
		OperationCount:    p.operationCount,
		AverageLatency:    avgLatency,
		ProfileTriggers:   copyTriggers(p.profileTriggers),
		HTTPEndpoint:      p.config.HTTPEndpoint,
	}
}

// ProfilerStats contains statistics about the profiler
type ProfilerStats struct {
	IsRunning       bool              `json:"is_running"`
	ActiveProfiles  int               `json:"active_profiles"`
	TotalSessions   int               `json:"total_sessions"`
	OperationCount  int64             `json:"operation_count"`
	AverageLatency  time.Duration     `json:"average_latency"`
	ProfileTriggers map[string]int64  `json:"profile_triggers"`
	HTTPEndpoint    string            `json:"http_endpoint"`
}

// Helper functions

// createProfileFile creates a profile file and ensures the directory exists
func createProfileFile(filename string) (*os.File, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profile directory: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile file: %w", err)
	}

	return file, nil
}

// copyTriggers creates a copy of the triggers map
func copyTriggers(triggers map[string]int64) map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range triggers {
		copy[k] = v
	}
	return copy
}

// Global profiler instance
var globalProfiler *Profiler
var globalProfilerOnce sync.Once

// GetGlobalProfiler returns the global profiler instance
func GetGlobalProfiler() *Profiler {
	globalProfilerOnce.Do(func() {
		globalProfiler = NewProfiler(DefaultProfilingConfig())
	})
	return globalProfiler
}

// InitializeGlobalProfiler initializes the global profiler with custom config
func InitializeGlobalProfiler(config ProfilingConfig) *Profiler {
	globalProfilerOnce.Do(func() {
		globalProfiler = NewProfiler(config)
	})
	return globalProfiler
}

// StartGlobalProfiling starts the global profiler
func StartGlobalProfiling(ctx context.Context) error {
	return GetGlobalProfiler().Start(ctx)
}

// StopGlobalProfiling stops the global profiler
func StopGlobalProfiling(ctx context.Context) error {
	return GetGlobalProfiler().Stop(ctx)
}

// ProfileGlobalOperation profiles an operation using the global profiler
func ProfileGlobalOperation(ctx context.Context, operationName string, operation func() error) error {
	return GetGlobalProfiler().ProfileOperation(ctx, operationName, operation)
}