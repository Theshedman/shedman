package core

// DoctorChecks holds check functions for testing
type DoctorChecks struct {
	CheckConnection func() bool
	CheckDiskSpace  func(path string) float64
	CheckServices   func() []string
	CheckLockFile   func() bool
}

type DiagnoseStatus int

const (
	DiagnoseStatusOk DiagnoseStatus = iota
	DiagnoseStatusWarn
	DiagnoseStatusFail
)

type DiagnoseItem struct {
	Name    string
	Status  DiagnoseStatus
	Message string
}

type SystemReport struct {
	Items []DiagnoseItem
}

func (r SystemReport) IsHealthy() bool {
	for _, i := range r.Items {
		if i.Status == DiagnoseStatusFail {
			return false
		}
	}
	return true
}
