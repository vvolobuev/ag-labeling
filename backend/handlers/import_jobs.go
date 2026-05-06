package handlers

import (
	"crypto/rand"
	"net/http"
	"sync"
	"time"

	"my-app/middleware"

	"github.com/gin-gonic/gin"
)

// importJobs holds import jobs in-memory (single process / dev; resets on restart).
var importJobs sync.Map // jobID -> *importJob

type importJob struct {
	mu              sync.Mutex
	userID          string
	cancelRequested bool
	status          importJobStatus
}

type importJobStatus struct {
	Phase   string  `json:"phase"`
	Percent int     `json:"percent"` // 0-100 while running; -1 while queued
	Detail  string  `json:"detail,omitempty"`
	Done    bool    `json:"done"`
	Err     string  `json:"error,omitempty"`
	HTTP    int     `json:"http,omitempty"`
	Result  gin.H   `json:"result,omitempty"`
	RawErr  gin.H   `json:"error_body,omitempty"`
}

const jobIDBytes = 10

func scheduleJobExpire(jobID string) {
	time.AfterFunc(35*time.Minute, func() { importJobs.Delete(jobID) })
}

func newImportJob(userID string) string {
	b := make([]byte, jobIDBytes)
	if _, err := rand.Read(b); err != nil {
		// very rare; ID is still unique enough, but keep fallback path
		for i := range b {
			b[i] = byte(time.Now().UnixNano() >> (8 * (i % 8)))
		}
	}
	id := hexEncode(b)
	job := &importJob{
		userID: userID,
		status: importJobStatus{
			Phase:   "queued",
			Percent: -1,
		},
	}
	importJobs.Store(id, job)
	scheduleJobExpire(id)
	return id
}

func hexEncode(b []byte) string {
	const hx = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, v := range b {
		out = append(out, hx[v>>4], hx[v&0x0f])
	}
	return string(out)
}

func (j *importJob) applyProgress(phase string, pct int, detail string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status.Phase = phase
	j.status.Percent = pct
	j.status.Detail = detail
}

func (j *importJob) finishOK(result gin.H) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status.Phase = "done"
	j.status.Percent = 100
	j.status.Detail = ""
	j.status.Done = true
	j.status.Result = result
	j.status.Err = ""
	j.status.HTTP = 0
	j.status.RawErr = nil
}

func (j *importJob) finishErr(httpStatus int, msg string, raw gin.H) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status.Phase = "error"
	j.status.Percent = 100
	j.status.Done = true
	j.status.HTTP = httpStatus
	j.status.Err = msg
	j.status.RawErr = raw
	j.status.Result = nil
}

func (j *importJob) snapshot() importJobStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.status
}

func (j *importJob) requestCancel() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cancelRequested = true
	if !j.status.Done && j.status.Phase != "error" {
		j.status.Phase = "cancelling"
		if j.status.Percent < 0 {
			j.status.Percent = 0
		}
		j.status.Detail = "cancellation requested"
	}
}

func (j *importJob) isCancelRequested() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.cancelRequested
}

func importJobByID(jobID string) (*importJob, bool) {
	v, ok := importJobs.Load(jobID)
	if !ok {
		return nil, false
	}
	j, ok := v.(*importJob)
	return j, ok
}

// GetImportJob polls background import status.
func (_ *DatasetHandler) GetImportJob(c *gin.Context) {
	jid := c.Param("jid")
	job, ok := importJobByID(jid)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found or expired"})
		return
	}
	uid := middleware.UserID(c)
	job.mu.Lock()
	owner := job.userID
	job.mu.Unlock()
	if uid == "" || owner != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access to this job"})
		return
	}
	c.JSON(http.StatusOK, job.snapshot())
}

func (_ *DatasetHandler) CancelImportJob(c *gin.Context) {
	jid := c.Param("jid")
	job, ok := importJobByID(jid)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found or expired"})
		return
	}
	uid := middleware.UserID(c)
	job.mu.Lock()
	owner := job.userID
	job.mu.Unlock()
	if uid == "" || owner != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access to this job"})
		return
	}
	job.requestCancel()
	c.JSON(http.StatusOK, gin.H{"ok": true, "job_id": jid, "cancelling": true})
}

func asyncImportRequested(c *gin.Context) bool {
	return c.GetHeader("X-AlphaGuard-Import-Async") == "1"
}
