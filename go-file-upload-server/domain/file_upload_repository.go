package domain

type FileUploadRepository interface {
	UploadFile(fileUpload FileUpload, fileData []byte) (FileUpload, error)
	GetFileUploadByID(id string) (FileUpload, error)
	GetFileUploadDataByID(id string) ([]byte, error)
	ListFileUploads() ([]FileUpload, error)
	DeleteFileUpload(id string) error
}
