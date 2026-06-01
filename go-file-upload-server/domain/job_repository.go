package domain

type JobRepository interface {
    CreateJob(job Job) (Job, error)
    ListJobs() ([]Job, error)
    GetJobByID(id string) (Job, error)
    UpdateJob(job Job) (Job, error)

    CreateTask(task Task) (Task, error)
    ListTasksByJob(jobID string) ([]Task, error)
    GetTaskByID(id string) (Task, error)
    UpdateTask(task Task) (Task, error)

    CreateTimeEntry(entry TimeEntry) (TimeEntry, error)
    GetTimeEntryByID(id string) (TimeEntry, error)
    GetActiveTimeEntry(taskID, userID string) (TimeEntry, error)
    UpdateTimeEntry(entry TimeEntry) (TimeEntry, error)
    ListTimeEntriesByTask(taskID string) ([]TimeEntry, error)
    ListTimeEntriesByUser(userID string) ([]TimeEntry, error)

    CreateEnergyAudit(audit EnergyAudit) (EnergyAudit, error)
    ListEnergyAuditsByTask(taskID string) ([]EnergyAudit, error)

    UploadImage(image ImageUpload, imageData []byte) (ImageUpload, error)
    ListImageUploadsByTask(taskID string) ([]ImageUpload, error)
    GetImageUploadByID(id string) (ImageUpload, error)
    GetImageUploadDataByID(id string) ([]byte, error)
}
